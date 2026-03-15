package routes

import (
	"Backend/internal/controllers"
	"net/http"
)

// SetupAuthRoutes 認証関連のルーティング設定
func SetupAuthRoutes(authController *controllers.AuthController, oauthController *controllers.OAuthController) {
	// 認証エンドポイント
	http.HandleFunc("/api/auth/request-registration", authController.RequestRegistration)
	http.HandleFunc("/api/auth/verify-registration", authController.VerifyRegistration)
	http.HandleFunc("/api/auth/register", authController.Register)
	http.HandleFunc("/api/auth/login", authController.Login)
	http.HandleFunc("/api/auth/guest", authController.CreateGuest)
	http.HandleFunc("/api/auth/user", authController.GetUser)
	http.HandleFunc("/api/auth/profile", authController.UpdateProfile)
	http.HandleFunc("/api/auth/verify-email", authController.VerifyEmail)
	http.HandleFunc("/api/auth/forgot-password", authController.RequestPasswordReset)
	http.HandleFunc("/api/auth/reset-password", authController.ResetPassword)

	// OAuth エンドポイント
	http.HandleFunc("/api/auth/google", oauthController.GoogleLogin)
	http.HandleFunc("/api/auth/google/callback", oauthController.GoogleCallback)
	http.HandleFunc("/api/auth/github", oauthController.GitHubLogin)
	http.HandleFunc("/api/auth/github/callback", oauthController.GitHubCallback)
}
