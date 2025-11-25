package routes

import (
	"Backend/internal/controllers"
	"net/http"
)

// SetupAuthRoutes 認証関連のルーティング設定
func SetupAuthRoutes(authController *controllers.AuthController, oauthController *controllers.OAuthController) {
	// 認証エンドポイント
	http.HandleFunc("/api/auth/register", authController.Register)
	http.HandleFunc("/api/auth/login", authController.Login)
	http.HandleFunc("/api/auth/guest", authController.CreateGuest)
	http.HandleFunc("/api/auth/user", authController.GetUser)

	// OAuth エンドポイント
	http.HandleFunc("/api/auth/google", oauthController.GoogleLogin)
	http.HandleFunc("/api/auth/google/callback", oauthController.GoogleCallback)
	http.HandleFunc("/api/auth/github", oauthController.GitHubLogin)
	http.HandleFunc("/api/auth/github/callback", oauthController.GitHubCallback)
}
