package routes

import (
	"Backend/internal/controllers"
	"net/http"
)

func SetupAdminRoutes(
	adminCompanyController *controllers.AdminCompanyController,
	adminCrawlController *controllers.AdminCrawlController,
	adminJobController *controllers.AdminJobController,
	adminUserController *controllers.AdminUserController,
	adminAuditController *controllers.AdminAuditController,
) {
	http.HandleFunc("/api/admin/companies", adminCompanyController.ListOrCreate)
	http.HandleFunc("/api/admin/companies/", adminCompanyController.Detail)
	http.HandleFunc("/api/admin/crawl-sources", adminCrawlController.Sources)
	http.HandleFunc("/api/admin/crawl-sources/", adminCrawlController.SourceDetail)
	http.HandleFunc("/api/admin/crawl-runs", adminCrawlController.Runs)
	http.HandleFunc("/api/admin/job-categories", adminJobController.JobCategories)
	http.HandleFunc("/api/admin/job-positions", adminJobController.JobPositions)
	http.HandleFunc("/api/admin/graduate-employments", adminJobController.GraduateEmployments)
	http.HandleFunc("/api/admin/users", adminUserController.List)
	http.HandleFunc("/api/admin/users/", adminUserController.Update)
	http.HandleFunc("/api/admin/audit-logs", adminAuditController.List)
}
