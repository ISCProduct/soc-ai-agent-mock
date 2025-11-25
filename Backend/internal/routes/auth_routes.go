package routes

import (
	"Backend/internal/controllers"
	"net/http"
)

// SetupAuthRoutes 認証関連のルーティング設定
func SetupAuthRoutes(authController *controllers.AuthController) {
	// 認証エンドポイント
	http.HandleFunc("/api/auth/register", authController.Register)
	http.HandleFunc("/api/auth/login", authController.Login)
	http.HandleFunc("/api/auth/guest", authController.CreateGuest)
	http.HandleFunc("/api/auth/user", authController.GetUser)
}
