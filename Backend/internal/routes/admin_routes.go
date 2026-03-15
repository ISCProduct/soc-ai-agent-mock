package routes

import (
	"Backend/internal/controllers"
	"Backend/internal/middleware"
	"Backend/internal/repositories"
	"net/http"
)

func SetupAdminRoutes(
	adminCompanyController *controllers.AdminCompanyController,
	adminCrawlController *controllers.AdminCrawlController,
	adminJobController *controllers.AdminJobController,
	adminUserController *controllers.AdminUserController,
	adminAuditController *controllers.AdminAuditController,
	adminCompanyGraphController *controllers.AdminCompanyGraphController,
	userRepo *repositories.UserRepository,
) {
	auth := func(f http.HandlerFunc) http.HandlerFunc {
		return middleware.AdminAuthFunc(userRepo, f)
	}

	http.HandleFunc("/api/admin/companies", auth(adminCompanyController.ListOrCreate))
	http.HandleFunc("/api/admin/companies/", auth(adminCompanyController.Detail))
	http.HandleFunc("/api/admin/crawl-sources", auth(adminCrawlController.Sources))
	http.HandleFunc("/api/admin/crawl-sources/", auth(adminCrawlController.SourceDetail))
	http.HandleFunc("/api/admin/crawl-runs", auth(adminCrawlController.Runs))
	http.HandleFunc("/api/admin/job-categories", auth(adminJobController.JobCategories))
	http.HandleFunc("/api/admin/job-positions", auth(adminJobController.JobPositions))
	http.HandleFunc("/api/admin/job-positions/", auth(adminJobController.JobPositionAction))
	http.HandleFunc("/api/admin/graduate-employments", auth(adminJobController.GraduateEmployments))
	http.HandleFunc("/api/admin/graduate-employments/", auth(adminJobController.GraduateEmploymentDetail))
	http.HandleFunc("/api/admin/users", auth(adminUserController.List))
	http.HandleFunc("/api/admin/users/", auth(adminUserController.Update))
	http.HandleFunc("/api/admin/audit-logs", auth(adminAuditController.List))

	// Company graph (scraping pipeline)
	http.HandleFunc("/api/admin/company-graph/target-year", adminCompanyGraphController.TargetYear)
	http.HandleFunc("/api/admin/company-graph/crawl", auth(adminCompanyGraphController.Crawl))
}
