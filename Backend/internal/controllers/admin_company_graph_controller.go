package controllers

import (
	"Backend/internal/scraper"
	"Backend/internal/services"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// AdminCompanyGraphController exposes the multi-source scraping pipeline via HTTP.
type AdminCompanyGraphController struct {
	pipeline *scraper.Pipeline
	audit    *services.AuditLogService
}

func NewAdminCompanyGraphController(pipeline *scraper.Pipeline, audit *services.AuditLogService) *AdminCompanyGraphController {
	return &AdminCompanyGraphController{pipeline: pipeline, audit: audit}
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
		Sites    []string `json:"sites"`
		Query    string   `json:"query"`
		Pages    int      `json:"pages"`
		Year     int      `json:"year"`
		Threshold float64 `json:"threshold"`
	}
	req.Sites = []string{"rikunabi", "career_tasu"} // sensible defaults
	req.Query = "IT"
	req.Pages = 2
	req.Threshold = 0.75

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Apply request threshold to pipeline (copy-safe)
	p := *c.pipeline
	if req.Threshold > 0 {
		p.Threshold = req.Threshold
	}

	pipeReq := scraper.RunRequest{
		Sites:    req.Sites,
		Query:    req.Query,
		MaxPages: req.Pages,
		Year:     req.Year,
	}

	result, err := p.Run(r.Context(), pipeReq)

	w.Header().Set("Content-Type", "application/json")
	if err != nil && (result == nil || len(result.Nodes) == 0) {
		logs := ""
		if result != nil {
			logs = strings.Join(result.Logs, "\n")
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
			"logs":  logs,
		})
		return
	}

	logs := ""
	if result != nil {
		logs = strings.Join(result.Logs, "\n")
	}

	nodeCount := 0
	if result != nil {
		nodeCount = len(result.Nodes)
	}

	json.NewEncoder(w).Encode(map[string]any{
		"ok":          true,
		"logs":        logs,
		"nodes":       nodeCount,
		"target_year": result.TargetYear,
	})

	// Audit log (best-effort)
	adminEmail := r.Header.Get("X-Admin-Email")
	if adminEmail != "" && c.audit != nil {
		c.audit.Record(adminEmail, "company_graph_crawl", "pipeline", 0, map[string]interface{}{
			"sites": req.Sites,
			"query": req.Query,
			"nodes": nodeCount,
		})
	}
}
