package controllers

import (
	"Backend/internal/models"
	"Backend/internal/repositories"
	"Backend/internal/services"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type AdminJobController struct {
	companyRepo  *repositories.CompanyRepository
	jobCategory  *repositories.JobCategoryRepository
	graduateRepo *repositories.GraduateEmploymentRepository
	audit        *services.AuditLogService
}

func NewAdminJobController(companyRepo *repositories.CompanyRepository, jobCategory *repositories.JobCategoryRepository, graduateRepo *repositories.GraduateEmploymentRepository, audit *services.AuditLogService) *AdminJobController {
	return &AdminJobController{
		companyRepo:  companyRepo,
		jobCategory:  jobCategory,
		graduateRepo: graduateRepo,
		audit:        audit,
	}
}

func (c *AdminJobController) JobCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	categories, err := c.jobCategory.FindAll()
	if err != nil {
		http.Error(w, "failed to fetch job categories", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_categories": categories,
	})
}

func (c *AdminJobController) JobPositions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		c.listJobPositions(w, r)
	case http.MethodPost:
		c.createJobPosition(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (c *AdminJobController) GraduateEmployments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		c.listGraduateEmployments(w, r)
	case http.MethodPost:
		c.createGraduateEmployment(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (c *AdminJobController) listJobPositions(w http.ResponseWriter, r *http.Request) {
	var companyID *uint
	if idStr := strings.TrimSpace(r.URL.Query().Get("company_id")); idStr != "" {
		if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			value := uint(id)
			companyID = &value
		}
	}
	limit := 50
	if limitStr := strings.TrimSpace(r.URL.Query().Get("limit")); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			limit = v
		}
	}
	positions, err := c.companyRepo.ListJobPositions(companyID, limit)
	if err != nil {
		http.Error(w, "failed to fetch job positions", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"positions": positions,
	})
}

func (c *AdminJobController) createJobPosition(w http.ResponseWriter, r *http.Request) {
	var payload models.CompanyJobPosition
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if payload.CompanyID == 0 || strings.TrimSpace(payload.Title) == "" {
		http.Error(w, "company_id and title are required", http.StatusBadRequest)
		return
	}
	if payload.JobCategoryID == 0 {
		http.Error(w, "job_category_id is required", http.StatusBadRequest)
		return
	}
	payload.IsActive = true
	if err := c.companyRepo.CreateJobPosition(&payload); err != nil {
		http.Error(w, "failed to create job position", http.StatusInternalServerError)
		return
	}
	actor := r.Header.Get("X-Admin-Email")
	c.audit.Record(actor, "job_position.create", "company_job_position", payload.ID, map[string]interface{}{
		"company_id": payload.CompanyID,
		"title":      payload.Title,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

func (c *AdminJobController) listGraduateEmployments(w http.ResponseWriter, r *http.Request) {
	var companyID *uint
	if idStr := strings.TrimSpace(r.URL.Query().Get("company_id")); idStr != "" {
		if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			value := uint(id)
			companyID = &value
		}
	}
	limit := 50
	if limitStr := strings.TrimSpace(r.URL.Query().Get("limit")); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			limit = v
		}
	}
	entries, err := c.graduateRepo.List(companyID, limit)
	if err != nil {
		http.Error(w, "failed to fetch graduate employments", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": entries,
	})
}

func (c *AdminJobController) createGraduateEmployment(w http.ResponseWriter, r *http.Request) {
	type payloadRequest struct {
		CompanyID      uint   `json:"company_id"`
		JobPositionID  *uint  `json:"job_position_id"`
		GraduateName   string `json:"graduate_name"`
		GraduationYear int    `json:"graduation_year"`
		SchoolName     string `json:"school_name"`
		Department     string `json:"department"`
		HiredAt        string `json:"hired_at"`
		Note           string `json:"note"`
	}
	var payload payloadRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if payload.CompanyID == 0 {
		http.Error(w, "company_id is required", http.StatusBadRequest)
		return
	}
	var hiredAt *time.Time
	if strings.TrimSpace(payload.HiredAt) != "" {
		if parsed, err := time.Parse("2006-01-02", payload.HiredAt); err == nil {
			hiredAt = &parsed
		}
	}
	entry := &models.GraduateEmployment{
		CompanyID:      payload.CompanyID,
		JobPositionID:  payload.JobPositionID,
		GraduateName:   strings.TrimSpace(payload.GraduateName),
		GraduationYear: payload.GraduationYear,
		SchoolName:     strings.TrimSpace(payload.SchoolName),
		Department:     strings.TrimSpace(payload.Department),
		HiredAt:        hiredAt,
		Note:           strings.TrimSpace(payload.Note),
	}
	if err := c.graduateRepo.Create(entry); err != nil {
		http.Error(w, "failed to create graduate employment", http.StatusInternalServerError)
		return
	}
	actor := r.Header.Get("X-Admin-Email")
	c.audit.Record(actor, "graduate_employment.create", "graduate_employment", entry.ID, map[string]interface{}{
		"company_id": entry.CompanyID,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}
