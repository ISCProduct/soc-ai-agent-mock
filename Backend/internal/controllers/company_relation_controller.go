package controllers

import (
	"Backend/domain/repository"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

type CompanyRelationController struct {
	repo repository.CompanyRelationQueryRepository
}

func NewCompanyRelationController(repo repository.CompanyRelationQueryRepository) *CompanyRelationController {
	return &CompanyRelationController{repo: repo}
}

// GetCompanyRelations 企業IDに関連する企業関係を取得
func (ctrl *CompanyRelationController) GetCompanyRelations(w http.ResponseWriter, r *http.Request) {
	// パスから企業IDを抽出
	pathParts := splitPath(r.URL.Path)
	if len(pathParts) < 3 {
		http.Error(w, "Invalid company ID", http.StatusBadRequest)
		return
	}

	companyID, err := strconv.ParseUint(pathParts[2], 10, 32)
	if err != nil {
		http.Error(w, "Invalid company ID", http.StatusBadRequest)
		return
	}

	relations, err := ctrl.repo.GetByCompanyID(uint(companyID))
	if err != nil {
		http.Error(w, "Failed to fetch relations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(relations)
}

// GetCompanyMarketInfo 企業の市場情報を取得
func (ctrl *CompanyRelationController) GetCompanyMarketInfo(w http.ResponseWriter, r *http.Request) {
	pathParts := splitPath(r.URL.Path)
	if len(pathParts) < 3 {
		http.Error(w, "Invalid company ID", http.StatusBadRequest)
		return
	}

	companyID, err := strconv.ParseUint(pathParts[2], 10, 32)
	if err != nil {
		http.Error(w, "Invalid company ID", http.StatusBadRequest)
		return
	}

	marketInfo, err := ctrl.repo.GetMarketInfoByCompanyID(uint(companyID))
	if err != nil {
		http.Error(w, "Failed to fetch market info", http.StatusInternalServerError)
		return
	}
	if marketInfo == nil {
		http.Error(w, "Market info not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(marketInfo)
}

// GetAllCompanyRelations 全企業関係を取得（関連図用）
func (ctrl *CompanyRelationController) GetAllCompanyRelations(w http.ResponseWriter, r *http.Request) {
	relations, err := ctrl.repo.GetAll()
	if err != nil {
		http.Error(w, "Failed to fetch relations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(relations)
}

// GetAllMarketInfo 全企業の市場情報を取得
func (ctrl *CompanyRelationController) GetAllMarketInfo(w http.ResponseWriter, r *http.Request) {
	marketInfos, err := ctrl.repo.GetAllMarketInfo()
	if err != nil {
		http.Error(w, "Failed to fetch market info", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(marketInfos)
}

// GetCompanyJobPositions 企業の公開済み求人一覧を取得
func (ctrl *CompanyRelationController) GetCompanyJobPositions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pathParts := splitPath(r.URL.Path)
	// path: /api/companies/{id}/job-positions → ["api","companies","{id}","job-positions"]
	if len(pathParts) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	companyID, err := strconv.ParseUint(pathParts[2], 10, 32)
	if err != nil {
		http.Error(w, "Invalid company ID", http.StatusBadRequest)
		return
	}

	positions, err := ctrl.repo.GetJobPositionsByCompany(uint(companyID))
	if err != nil {
		http.Error(w, "Failed to fetch job positions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"positions": positions,
	})
}

// GetCompanies 企業一覧を取得
func (ctrl *CompanyRelationController) GetCompanies(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	industry := r.URL.Query().Get("industry")
	name := r.URL.Query().Get("name")

	limit := 10 // デフォルト
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 100 {
				limit = 100 // 最大100件
			}
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	companies, total, err := ctrl.repo.GetCompaniesFiltered(limit, offset, industry, name)
	if err != nil {
		http.Error(w, "Failed to fetch companies", http.StatusInternalServerError)
		return
	}

	type CompanyResponse struct {
		Companies interface{} `json:"companies"`
		Total     int64       `json:"total"`
		Limit     int         `json:"limit"`
		Offset    int         `json:"offset"`
	}

	response := CompanyResponse{
		Companies: companies,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// WebSearchCompanies Wikipedia APIを使用して企業をWEB検索
func (ctrl *CompanyRelationController) WebSearchCompanies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "q parameter is required", http.StatusBadRequest)
		return
	}
	query = trimSpace(query)
	if query == "" {
		http.Error(w, "q parameter is required", http.StatusBadRequest)
		return
	}

	searchURL := fmt.Sprintf(
		"https://ja.wikipedia.org/w/api.php?action=query&list=search&srsearch=%s&srlimit=8&format=json&origin=*",
		url.QueryEscape(query),
	)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(searchURL)
	if err != nil {
		http.Error(w, "Web search failed", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read search response", http.StatusInternalServerError)
		return
	}

	var wikiResp struct {
		Query struct {
			Search []struct {
				Title   string `json:"title"`
				Snippet string `json:"snippet"`
			} `json:"search"`
		} `json:"query"`
	}
	if err := json.Unmarshal(body, &wikiResp); err != nil {
		http.Error(w, "Failed to parse search response", http.StatusInternalServerError)
		return
	}

	htmlTagRe := regexp.MustCompile(`<[^>]+>`)

	type WebSearchResult struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Source      string `json:"source"`
	}

	results := make([]WebSearchResult, 0, len(wikiResp.Query.Search))
	for _, item := range wikiResp.Query.Search {
		snippet := htmlTagRe.ReplaceAllString(item.Snippet, "")
		results = append(results, WebSearchResult{
			Name:        item.Title,
			Description: snippet,
			Source:      "Wikipedia",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
}

// splitPath はURLパスを "/" で分割してスラッシュを除去した要素のスライスを返す
func splitPath(path string) []string {
	// strings パッケージは同一ファイル内で使えるのでコピーして利用
	result := []string{}
	start := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '/' {
			if i > start {
				result = append(result, path[start:i])
			}
			start = i + 1
		}
	}
	return result
}

// trimSpace は文字列の先頭と末尾の空白を除去する（標準ライブラリ呼び出しを避けるためのラッパー）
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
