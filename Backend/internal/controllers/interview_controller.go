package controllers

import (
	"Backend/internal/services"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type InterviewController struct {
	interviewService *services.InterviewService
}

func NewInterviewController(interviewService *services.InterviewService) *InterviewController {
	return &InterviewController{interviewService: interviewService}
}

type interviewCreateRequest struct {
	UserID uint `json:"user_id"`
}

type interviewActionRequest struct {
	UserID uint `json:"user_id"`
}

type interviewUtteranceRequest struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	Text   string `json:"text"`
}

func (c *InterviewController) ListOrCreate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		c.List(w, r)
	case http.MethodPost:
		c.Create(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (c *InterviewController) Route(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/interviews/")
	if strings.HasSuffix(path, "/start") {
		c.Start(w, r)
		return
	}
	if strings.HasSuffix(path, "/finish") {
		c.Finish(w, r)
		return
	}
	if strings.HasSuffix(path, "/utterances") {
		c.AddUtterance(w, r)
		return
	}
	c.Get(w, r)
}

func (c *InterviewController) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req interviewCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.UserID == 0 {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}
	resp, err := c.interviewService.CreateSession(req.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (c *InterviewController) Start(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID, err := extractID(r.URL.Path, "/api/interviews/", "/start")
	if err != nil {
		http.Error(w, "invalid interview id", http.StatusBadRequest)
		return
	}
	var req interviewActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.UserID == 0 {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}
	resp, err := c.interviewService.StartSession(req.UserID, sessionID)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "forbidden" {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (c *InterviewController) Finish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID, err := extractID(r.URL.Path, "/api/interviews/", "/finish")
	if err != nil {
		http.Error(w, "invalid interview id", http.StatusBadRequest)
		return
	}
	var req interviewActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.UserID == 0 {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}
	resp, err := c.interviewService.FinishSession(req.UserID, sessionID)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "forbidden" {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (c *InterviewController) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}
	page := parseIntQuery(r, "page", 1)
	limit := parseIntQuery(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit
	all := r.URL.Query().Get("all") == "1" || strings.ToLower(r.URL.Query().Get("all")) == "true"
	sessions, total, err := c.interviewService.ListSessions(uint(userID), all, limit, offset)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "forbidden" {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessions": sessions,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

func (c *InterviewController) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID, err := extractID(r.URL.Path, "/api/interviews/", "")
	if err != nil {
		http.Error(w, "invalid interview id", http.StatusBadRequest)
		return
	}
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}
	resp, err := c.interviewService.GetSessionDetail(uint(userID), sessionID)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "forbidden" {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (c *InterviewController) AddUtterance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID, err := extractID(r.URL.Path, "/api/interviews/", "/utterances")
	if err != nil {
		http.Error(w, "invalid interview id", http.StatusBadRequest)
		return
	}
	var req interviewUtteranceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.UserID == 0 {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}
	if err := c.interviewService.SaveUtterance(req.UserID, sessionID, req.Role, req.Text); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "forbidden" {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func extractID(path string, prefix string, suffix string) (uint, error) {
	trimmed := strings.TrimPrefix(path, prefix)
	if suffix != "" && strings.HasSuffix(trimmed, suffix) {
		trimmed = strings.TrimSuffix(trimmed, suffix)
	}
	trimmed = strings.Trim(trimmed, "/")
	id, err := strconv.ParseUint(trimmed, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

func parseIntQuery(r *http.Request, key string, def int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return def
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
