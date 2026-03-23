package controllers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type ESReviewController struct{}

func NewESReviewController() *ESReviewController {
	return &ESReviewController{}
}

type esReviewRequest struct {
	ESText       string `json:"es_text"`
	QuestionType string `json:"question_type"`
	CompanyName  string `json:"company_name"`
}

func (c *ESReviewController) Review(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req esReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	req.ESText = strings.TrimSpace(req.ESText)
	if req.ESText == "" {
		http.Error(w, "es_text is required", http.StatusBadRequest)
		return
	}
	if req.QuestionType == "" {
		req.QuestionType = "その他"
	}

	ragURL := strings.TrimSpace(os.Getenv("RAG_REVIEW_URL"))
	if ragURL == "" {
		http.Error(w, "RAG_REVIEW_URL is not configured", http.StatusServiceUnavailable)
		return
	}

	body, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "Failed to encode request", http.StatusInternalServerError)
		return
	}

	url := strings.TrimRight(ragURL, "/") + "/es/review"
	log.Printf("es_review: rag request question_type=%q company=%q", req.QuestionType, req.CompanyName)

	ragReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "Failed to create RAG request", http.StatusInternalServerError)
		return
	}
	ragReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(ragReq)
	if err != nil {
		http.Error(w, "RAG service unavailable: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read RAG response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}
