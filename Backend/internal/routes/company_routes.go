package routes

import (
	"Backend/internal/controllers"
	"net/http"
)

// SetupCompanyRoutes 企業関連のルーティング設定
func SetupCompanyRoutes(relationController *controllers.CompanyRelationController) {
	http.HandleFunc("/api/companies", relationController.GetCompanies)
	http.HandleFunc("/api/companies/relations", relationController.GetAllCompanyRelations)
	http.HandleFunc("/api/companies/market-info", relationController.GetAllMarketInfo)
}
