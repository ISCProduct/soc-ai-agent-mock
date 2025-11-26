package controllers

import (
	"Backend/internal/services"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
)

type OAuthController struct {
	oauthService *services.OAuthService
}

func NewOAuthController(oauthService *services.OAuthService) *OAuthController {
	return &OAuthController{oauthService: oauthService}
}

// GoogleLogin Google OAuth認証開始
func (c *OAuthController) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state := generateStateToken()
	// 本番環境ではセッションやクッキーに保存してCSRF対策を行う
	url := c.oauthService.GetGoogleAuthURL(state)

	// リダイレクトURLをJSON形式で返す（フロントエンド側でリダイレクト）
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"auth_url": url,
		"state":    state,
	})
}

// GoogleCallback Google OAuth認証コールバック
func (c *OAuthController) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Authorization code not found", http.StatusBadRequest)
		return
	}

	// 本番環境ではstateの検証を行う
	// state := r.URL.Query().Get("state")

	resp, err := c.oauthService.HandleGoogleCallback(r.Context(), code)
	if err != nil {
		// エラー時はフロントエンドにリダイレクトしてエラーを表示
		http.Redirect(w, r, "http://localhost:3000?error="+err.Error(), http.StatusTemporaryRedirect)
		return
	}

	// 認証成功時はユーザー情報をクエリパラメータとしてフロントエンドに渡してリダイレクト
	userData, _ := json.Marshal(resp)
	redirectURL := "http://localhost:3000/auth/callback?provider=google&code=" + code + "&user=" + base64.URLEncoding.EncodeToString(userData)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// GitHubLogin GitHub OAuth認証開始
func (c *OAuthController) GitHubLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state := generateStateToken()
	url := c.oauthService.GetGitHubAuthURL(state)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"auth_url": url,
		"state":    state,
	})
}

// GitHubCallback GitHub OAuth認証コールバック
func (c *OAuthController) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Authorization code not found", http.StatusBadRequest)
		return
	}

	resp, err := c.oauthService.HandleGitHubCallback(r.Context(), code)
	if err != nil {
		// エラー時はフロントエンドにリダイレクトしてエラーを表示
		http.Redirect(w, r, "http://localhost:3000?error="+err.Error(), http.StatusTemporaryRedirect)
		return
	}

	// 認証成功時はユーザー情報をクエリパラメータとしてフロントエンドに渡してリダイレクト
	userData, _ := json.Marshal(resp)
	redirectURL := "http://localhost:3000/auth/callback?provider=github&code=" + code + "&user=" + base64.URLEncoding.EncodeToString(userData)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// generateStateToken CSRF対策用のランダムなstateトークンを生成
func generateStateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
