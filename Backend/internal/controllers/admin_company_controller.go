package controllers

import (
	"Backend/internal/models"
	"Backend/internal/repositories"
	"Backend/internal/services"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type AdminCompanyController struct {
	repo  *repositories.CompanyRepository
	audit *services.AuditLogService
	gbiz  *services.GBizInfoService
}

func NewAdminCompanyController(repo *repositories.CompanyRepository, audit *services.AuditLogService, gbiz *services.GBizInfoService) *AdminCompanyController {
	return &AdminCompanyController{repo: repo, audit: audit, gbiz: gbiz}
}

func (c *AdminCompanyController) ListOrCreate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		c.list(w, r)
	case http.MethodPost:
		c.create(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (c *AdminCompanyController) Detail(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/companies/")
	idStr = strings.Trim(idStr, "/")
	if idStr == "" {
		http.Error(w, "company id is required", http.StatusBadRequest)
		return
	}
	if idStr == "search-gbiz" {
		c.searchGBiz(w, r)
		return
	}
	if strings.HasSuffix(idStr, "/gbiz-sync") {
		c.syncGBiz(w, r, strings.TrimSuffix(idStr, "/gbiz-sync"))
		return
	}
	if strings.HasSuffix(idStr, "/publish") {
		c.publish(w, r, strings.TrimSuffix(idStr, "/publish"))
		return
	}
	if strings.HasSuffix(idStr, "/reject") {
		c.reject(w, r, strings.TrimSuffix(idStr, "/reject"))
		return
	}
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "invalid company id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		c.get(w, uint(id))
	case http.MethodPut:
		c.update(w, r, uint(id))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (c *AdminCompanyController) publish(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, err := strconv.ParseUint(strings.Trim(idStr, "/"), 10, 32)
	if err != nil {
		http.Error(w, "invalid company id", http.StatusBadRequest)
		return
	}
	company, err := c.repo.FindByID(uint(id))
	if err != nil {
		http.Error(w, "company not found", http.StatusNotFound)
		return
	}
	company.DataStatus = "published"
	company.IsProvisional = false
	if err := c.repo.Update(company); err != nil {
		http.Error(w, "failed to publish company", http.StatusInternalServerError)
		return
	}
	actor := r.Header.Get("X-Admin-Email")
	c.audit.Record(actor, "company.publish", "company", company.ID, map[string]interface{}{
		"name": company.Name,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(company)
}

func (c *AdminCompanyController) reject(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, err := strconv.ParseUint(strings.Trim(idStr, "/"), 10, 32)
	if err != nil {
		http.Error(w, "invalid company id", http.StatusBadRequest)
		return
	}
	company, err := c.repo.FindByID(uint(id))
	if err != nil {
		http.Error(w, "company not found", http.StatusNotFound)
		return
	}
	company.IsActive = false
	if err := c.repo.Update(company); err != nil {
		http.Error(w, "failed to reject company", http.StatusInternalServerError)
		return
	}
	actor := r.Header.Get("X-Admin-Email")
	c.audit.Record(actor, "company.reject", "company", company.ID, map[string]interface{}{
		"name": company.Name,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
}

func (c *AdminCompanyController) list(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && v >= 0 {
		offset = v
	}
	companies, err := c.repo.FindAllActive(limit, offset)
	if err != nil {
		http.Error(w, "failed to fetch companies", http.StatusInternalServerError)
		return
	}
	total, _ := c.repo.CountActive()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"companies": companies,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

func (c *AdminCompanyController) create(w http.ResponseWriter, r *http.Request) {
	var payload models.Company
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	applyCompanyDefaults(&payload)
	if err := c.repo.Create(&payload); err != nil {
		http.Error(w, "failed to create company", http.StatusInternalServerError)
		return
	}
	actor := r.Header.Get("X-Admin-Email")
	c.audit.Record(actor, "company.create", "company", payload.ID, map[string]interface{}{
		"name": payload.Name,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

func (c *AdminCompanyController) get(w http.ResponseWriter, id uint) {
	company, err := c.repo.FindByID(id)
	if err != nil {
		http.Error(w, "company not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(company)
}

func (c *AdminCompanyController) update(w http.ResponseWriter, r *http.Request, id uint) {
	var payload models.Company
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	company, err := c.repo.FindByID(id)
	if err != nil {
		http.Error(w, "company not found", http.StatusNotFound)
		return
	}

	if err := mergeCompany(company, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := c.repo.Update(company); err != nil {
		http.Error(w, "failed to update company", http.StatusInternalServerError)
		return
	}
	actor := r.Header.Get("X-Admin-Email")
	c.audit.Record(actor, "company.update", "company", company.ID, map[string]interface{}{
		"name": company.Name,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(company)
}

func (c *AdminCompanyController) searchGBiz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if c.gbiz == nil {
		http.Error(w, "gbizinfo service not configured", http.StatusServiceUnavailable)
		return
	}
	results, err := c.gbiz.SearchByName(r.Context(), name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
}

func (c *AdminCompanyController) syncGBiz(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr = strings.Trim(idStr, "/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "invalid company id", http.StatusBadRequest)
		return
	}
	if c.gbiz == nil {
		http.Error(w, "gbizinfo service not configured", http.StatusServiceUnavailable)
		return
	}
	result, err := c.gbiz.SyncCompany(r.Context(), uint(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	actor := r.Header.Get("X-Admin-Email")
	c.audit.Record(actor, "company.gbiz_sync", "company", uint(id), map[string]interface{}{
		"status": result.Status,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func applyCompanyDefaults(company *models.Company) {
	if strings.TrimSpace(company.SourceType) == "" {
		company.SourceType = "manual"
	}
	if strings.TrimSpace(company.DataStatus) == "" {
		company.DataStatus = "draft"
	}
	if company.SourceFetchedAt == nil {
		now := time.Now()
		company.SourceFetchedAt = &now
	}
	if company.IsProvisional == false && strings.TrimSpace(company.SourceURL) == "" {
		company.IsProvisional = true
	}
	if strings.TrimSpace(company.Name) != "" && company.IsProvisional == false {
		company.IsVerified = true
	}
}

func mergeCompany(existing *models.Company, payload *models.Company) error {
	if strings.TrimSpace(payload.Name) != "" {
		existing.Name = payload.Name
	}
	if strings.TrimSpace(payload.Description) != "" {
		existing.Description = payload.Description
	}
	if strings.TrimSpace(payload.Industry) != "" {
		existing.Industry = payload.Industry
	}
	if strings.TrimSpace(payload.Location) != "" {
		existing.Location = payload.Location
	}
	if payload.EmployeeCount > 0 {
		existing.EmployeeCount = payload.EmployeeCount
	}
	if payload.FoundedYear > 0 {
		existing.FoundedYear = payload.FoundedYear
	}
	if strings.TrimSpace(payload.WebsiteURL) != "" {
		existing.WebsiteURL = payload.WebsiteURL
	}
	if strings.TrimSpace(payload.LogoURL) != "" {
		existing.LogoURL = payload.LogoURL
	}
	if strings.TrimSpace(payload.CorporateNumber) != "" {
		existing.CorporateNumber = payload.CorporateNumber
	}
	if strings.TrimSpace(payload.MainBusiness) != "" {
		existing.MainBusiness = payload.MainBusiness
	}
	if strings.TrimSpace(payload.Culture) != "" {
		existing.Culture = payload.Culture
	}
	if strings.TrimSpace(payload.WorkStyle) != "" {
		existing.WorkStyle = payload.WorkStyle
	}
	if strings.TrimSpace(payload.WelfareDetails) != "" {
		existing.WelfareDetails = payload.WelfareDetails
	}
	if strings.TrimSpace(payload.TechStack) != "" {
		existing.TechStack = payload.TechStack
	}
	if strings.TrimSpace(payload.DevelopmentStyle) != "" {
		existing.DevelopmentStyle = payload.DevelopmentStyle
	}
	if strings.TrimSpace(payload.SourceType) != "" {
		existing.SourceType = payload.SourceType
	}
	if strings.TrimSpace(payload.SourceURL) != "" {
		existing.SourceURL = payload.SourceURL
	}
	if payload.SourceFetchedAt != nil {
		existing.SourceFetchedAt = payload.SourceFetchedAt
	}
	if payload.DataStatus != "" {
		if payload.DataStatus != "draft" && payload.DataStatus != "published" {
			return errors.New("data_status must be draft or published")
		}
		existing.DataStatus = payload.DataStatus
	}
	existing.IsProvisional = payload.IsProvisional
	return nil
}
