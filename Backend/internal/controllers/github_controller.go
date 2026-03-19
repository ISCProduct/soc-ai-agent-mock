package controllers

import (
	"Backend/internal/services"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// GitHubController GitHub連携APIのコントローラー
type GitHubController struct {
	githubService *services.GitHubService
}

func NewGitHubController(githubService *services.GitHubService) *GitHubController {
	return &GitHubController{githubService: githubService}
}

// GetProfile GitHubプロフィール・リポジトリ・言語統計を取得する
// GET /api/github/profile?user_id=<id>
func (c *GitHubController) GetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := parseUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	profile, err := c.githubService.GetProfile(userID)
	if err != nil {
		http.Error(w, "failed to get github profile", http.StatusInternalServerError)
		return
	}
	if profile == nil {
		http.Error(w, "github profile not found", http.StatusNotFound)
		return
	}

	repos, err := c.githubService.GetRepositories(userID)
	if err != nil {
		http.Error(w, "failed to get repositories", http.StatusInternalServerError)
		return
	}

	langStats, err := c.githubService.GetLanguageStats(userID)
	if err != nil {
		http.Error(w, "failed to get language stats", http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"profile":        profile,
		"repositories":   repos,
		"language_stats": langStats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Sync GitHubデータの非同期同期をトリガーする
// POST /api/github/sync?user_id=<id>
func (c *GitHubController) Sync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := parseUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	profile, err := c.githubService.GetProfile(userID)
	if err != nil || profile == nil {
		http.Error(w, "github profile not found: please connect your GitHub account", http.StatusNotFound)
		return
	}

	c.githubService.TriggerAsyncSync(userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "sync started",
	})
}

// SyncAndWait GitHubデータを同期してから結果を返す（同期的）
// POST /api/github/sync/wait?user_id=<id>
func (c *GitHubController) SyncAndWait(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := parseUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := c.githubService.SyncUserData(context.Background(), userID); err != nil {
		http.Error(w, "sync failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "sync completed",
	})
}

func parseUserID(r *http.Request) (uint, error) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		return 0, fmt.Errorf("user_id is required")
	}
	id, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid user_id")
	}
	return uint(id), nil
}
