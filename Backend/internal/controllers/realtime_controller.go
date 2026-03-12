package controllers

import (
	"Backend/internal/services"
	"encoding/json"
	"net/http"
)

type RealtimeController struct {
	interviewService *services.InterviewService
}

func NewRealtimeController(interviewService *services.InterviewService) *RealtimeController {
	return &RealtimeController{interviewService: interviewService}
}

type realtimeTokenRequest struct {
	UserID      uint `json:"user_id"`
	InterviewID uint `json:"interview_id"`
}

type realtimeTokenResponse struct {
	ClientSecret string `json:"client_secret"`
}

func (c *RealtimeController) Token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req realtimeTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.UserID == 0 || req.InterviewID == 0 {
		http.Error(w, "user_id and interview_id are required", http.StatusBadRequest)
		return
	}
	secret, err := c.interviewService.CreateRealtimeToken(r.Context(), req.UserID, req.InterviewID)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "forbidden" {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(realtimeTokenResponse{ClientSecret: secret})
}
