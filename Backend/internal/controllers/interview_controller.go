package controllers

import (
	"Backend/internal/services"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
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
	UserID   uint   `json:"user_id"`
	Language string `json:"language"`
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
	if strings.HasSuffix(path, "/turn") {
		c.Turn(w, r)
		return
	}
	if strings.HasSuffix(path, "/start-turn") {
		c.StartTurn(w, r)
		return
	}
	if strings.HasSuffix(path, "/send-report") {
		c.SendReport(w, r)
		return
	}
	c.Get(w, r)
}

func (c *InterviewController) SendReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID, err := extractID(r.URL.Path, "/api/interviews/", "/send-report")
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}
	var req struct {
		UserID uint `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == 0 {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}
	if err := c.interviewService.SendReportEmail(req.UserID, sessionID); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "user not found" || err.Error() == "report not found" {
			status = http.StatusNotFound
		}
		if err.Error() == "guest users cannot receive email reports" {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "レポートをメールで送信しました"})
}

func (c *InterviewController) Turn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID, err := extractID(r.URL.Path, "/api/interviews/", "/turn")
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	// multipart から音声と履歴を取得
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	userIDStr := r.FormValue("user_id")
	var userID uint
	if id, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
		userID = uint(id)
	}
	historyStr := r.FormValue("history")
	var history []map[string]string
	if historyStr != "" {
		json.Unmarshal([]byte(historyStr), &history)
	}
	companyName := r.FormValue("company_name")
	position := r.FormValue("position")
	companyInfo := r.FormValue("company_info")

	audioFile, _, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, "audio file required", http.StatusBadRequest)
		return
	}
	defer audioFile.Close()
	audioData, err := io.ReadAll(audioFile)
	if err != nil {
		http.Error(w, "Failed to read audio", http.StatusInternalServerError)
		return
	}

	result, err := c.interviewService.Turn(r.Context(), userID, sessionID, audioData, history, companyName, position, companyInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// multipart レスポンス: JSON メタ + audio
	mw := multipart.NewWriter(w)
	w.Header().Set("Content-Type", "multipart/mixed; boundary="+mw.Boundary())

	metaPart, _ := mw.CreatePart(textproto.MIMEHeader{"Content-Type": {"application/json"}})
	json.NewEncoder(metaPart).Encode(map[string]string{
		"user_text": result.UserText,
		"ai_text":   result.AIText,
	})

	audioPart, _ := mw.CreatePart(textproto.MIMEHeader{"Content-Type": {"audio/mpeg"}})
	audioPart.Write(result.Audio)
	mw.Close()
}

func (c *InterviewController) StartTurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID, err := extractID(r.URL.Path, "/api/interviews/", "/start-turn")
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	var req struct {
		UserID      uint   `json:"user_id"`
		CompanyName string `json:"company_name"`
		Position    string `json:"position"`
		CompanyInfo string `json:"company_info"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	result, err := c.interviewService.StartTurn(r.Context(), req.UserID, sessionID, req.CompanyName, req.Position, req.CompanyInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mw := multipart.NewWriter(w)
	w.Header().Set("Content-Type", "multipart/mixed; boundary="+mw.Boundary())

	metaPart, _ := mw.CreatePart(textproto.MIMEHeader{"Content-Type": {"application/json"}})
	json.NewEncoder(metaPart).Encode(map[string]string{"ai_text": result.AIText})

	audioPart, _ := mw.CreatePart(textproto.MIMEHeader{"Content-Type": {"audio/mpeg"}})
	audioPart.Write(result.Audio)
	mw.Close()
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
	resp, err := c.interviewService.CreateSession(req.UserID, req.Language)
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
