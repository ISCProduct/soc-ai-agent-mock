package controllers

import (
	"Backend/internal/models"
	"Backend/internal/repositories"
	"Backend/internal/scraper"
	"Backend/internal/services"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// AdminCompanyGraphController exposes the multi-source scraping pipeline via HTTP.
type AdminCompanyGraphController struct {
	pipeline    *scraper.Pipeline
	companyRepo *repositories.CompanyRepository
	audit       *services.AuditLogService
}

func NewAdminCompanyGraphController(pipeline *scraper.Pipeline, companyRepo *repositories.CompanyRepository, audit *services.AuditLogService) *AdminCompanyGraphController {
	return &AdminCompanyGraphController{pipeline: pipeline, companyRepo: companyRepo, audit: audit}
}

// TargetYear handles GET /api/admin/company-graph/target-year
func (c *AdminCompanyGraphController) TargetYear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	override := 0
	if v := r.URL.Query().Get("year"); v != "" {
		override, _ = strconv.Atoi(v)
	}
	y := scraper.ResolveYear(override)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"target_year": y})
}

// Crawl handles POST /api/admin/company-graph/crawl
func (c *AdminCompanyGraphController) Crawl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Sites     []string `json:"sites"`
		Query     string   `json:"query"`
		Pages     int      `json:"pages"`
		Year      int      `json:"year"`
		Threshold float64  `json:"threshold"`
	}
	req.Sites = []string{"rikunabi", "career_tasu"}
	req.Query = "IT"
	req.Pages = 2
	req.Threshold = 0.75

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// company-graph コンテナ経由でクロール（未設定の場合は埋め込みパイプラインを使用）
	var nodes map[string]*scraper.CompanyNode
	var logs string
	var targetYear int

	companyGraphURL := os.Getenv("COMPANY_GRAPH_URL")
	if companyGraphURL != "" {
		n, l, y, err := c.crawlViaService(r, companyGraphURL, req.Sites, req.Query, req.Pages, req.Year, req.Threshold)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}
		nodes, logs, targetYear = n, l, y
	} else {
		p := *c.pipeline
		if req.Threshold > 0 {
			p.Threshold = req.Threshold
		}
		result, err := p.Run(r.Context(), scraper.RunRequest{
			Sites:    req.Sites,
			Query:    req.Query,
			MaxPages: req.Pages,
			Year:     req.Year,
		})
		if err != nil && (result == nil || len(result.Nodes) == 0) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			l := ""
			if result != nil {
				l = strings.Join(result.Logs, "\n")
			}
			json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error(), "logs": l})
			return
		}
		if result != nil {
			nodes = result.Nodes
			logs = strings.Join(result.Logs, "\n")
			targetYear = result.TargetYear
		}
	}

	// DB 保存
	saved, skipped := c.upsertNodes(nodes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ok":          true,
		"logs":        logs,
		"nodes":       len(nodes),
		"saved":       saved,
		"skipped":     skipped,
		"target_year": targetYear,
	})

	adminEmail := r.Header.Get("X-Admin-Email")
	if adminEmail != "" && c.audit != nil {
		c.audit.Record(adminEmail, "company_graph_crawl", "pipeline", 0, map[string]interface{}{
			"sites": req.Sites,
			"query": req.Query,
			"nodes": len(nodes),
			"saved": saved,
		})
	}
}

// crawlViaService は company-graph コンテナを呼び出してノードデータを取得する。
func (c *AdminCompanyGraphController) crawlViaService(
	r *http.Request,
	baseURL string,
	sites []string, query string, pages, year int, threshold float64,
) (map[string]*scraper.CompanyNode, string, int, error) {

	payload, _ := json.Marshal(map[string]any{
		"sites":     sites,
		"query":     query,
		"pages":     pages,
		"year":      year,
		"threshold": threshold,
	})

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, baseURL+"/crawl", bytes.NewReader(payload))
	if err != nil {
		return nil, "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 310 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		OK         bool                          `json:"ok"`
		Error      string                        `json:"error"`
		Logs       string                        `json:"logs"`
		TargetYear int                           `json:"target_year"`
		Nodes      map[string]*scraper.CompanyNode `json:"nodes"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, "", 0, err
	}
	if !result.OK {
		return nil, result.Logs, result.TargetYear, errors.New(result.Error)
	}
	return result.Nodes, result.Logs, result.TargetYear, nil
}

// upsertNodes は CompanyNode を companies テーブルへ upsert する。
// 法人番号が既存のレコードと一致する場合は更新、なければ新規作成。
// 戻り値: (saved件数, skipped件数)
func (c *AdminCompanyGraphController) upsertNodes(nodes map[string]*scraper.CompanyNode) (int, int) {
	if c.companyRepo == nil {
		return 0, len(nodes)
	}
	now := time.Now()
	saved, skipped := 0, 0

	for _, node := range nodes {
		if node == nil || strings.HasPrefix(node.CorporateNumber, "UNKNOWN_") {
			skipped++
			continue
		}

		existing, err := c.companyRepo.FindByCorporateNumber(node.CorporateNumber)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			skipped++
			continue
		}

		sourceURL := ""
		if len(node.SourceURLs) > 0 {
			sourceURL = node.SourceURLs[0]
		}

		if existing == nil || errors.Is(err, gorm.ErrRecordNotFound) {
			company := &models.Company{
				Name:             node.OfficialName,
				CorporateNumber:  node.CorporateNumber,
				Industry:         node.BusinessCategory,
				Location:         node.Address,
				WebsiteURL:       node.Website,
				SourceType:       "job_site",
				SourceURL:        sourceURL,
				SourceFetchedAt:  &now,
				IsProvisional:    node.NeedsReview,
				DataStatus:       "draft",
				GBizLastSyncedAt: &now,
				GBizSyncStatus:   "success",
			}
			if err := c.companyRepo.Create(company); err != nil {
				skipped++
				continue
			}
		} else {
			if node.BusinessCategory != "" {
				existing.Industry = node.BusinessCategory
			}
			if node.Address != "" {
				existing.Location = node.Address
			}
			if node.Website != "" {
				existing.WebsiteURL = node.Website
			}
			if sourceURL != "" {
				existing.SourceURL = sourceURL
			}
			existing.IsProvisional = node.NeedsReview
			existing.SourceFetchedAt = &now
			existing.GBizLastSyncedAt = &now
			existing.GBizSyncStatus = "success"
			if err := c.companyRepo.Update(existing); err != nil {
				skipped++
				continue
			}
		}
		saved++
	}
	return saved, skipped
}
