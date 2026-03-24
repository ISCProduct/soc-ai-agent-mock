package routes

import (
	"Backend/internal/controllers"
	"net/http"
	"strings"
)

// SetupCompanyRoutes 企業関連のルーティング設定
func SetupCompanyRoutes(relationController *controllers.CompanyRelationController) {
	http.HandleFunc("/api/companies", relationController.GetCompanies)
	http.HandleFunc("/api/companies/relations", relationController.GetAllCompanyRelations)
	http.HandleFunc("/api/companies/market-info", relationController.GetAllMarketInfo)
	http.HandleFunc("/api/companies/web-search", relationController.WebSearchCompanies)
	http.HandleFunc("/api/companies/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/companies/")
		if strings.HasSuffix(path, "/job-positions") {
			relationController.GetCompanyJobPositions(w, r)
		} else if !strings.Contains(path, "/") {
			// /api/companies/{id}
			relationController.GetCompanyByID(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
}
