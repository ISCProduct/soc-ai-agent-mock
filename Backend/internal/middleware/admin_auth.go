package middleware

import (
	"Backend/internal/repositories"
	"net/http"
)

// AdminAuth X-Admin-Email ヘッダーでユーザーを検証し is_admin が true であることを確認する
func AdminAuth(userRepo *repositories.UserRepository, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email := r.Header.Get("X-Admin-Email")
		if email == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		user, err := userRepo.GetUserByEmail(email)
		if err != nil || user == nil || !user.IsAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AdminAuthFunc AdminAuth の http.HandlerFunc バージョン
func AdminAuthFunc(userRepo *repositories.UserRepository, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.Header.Get("X-Admin-Email")
		if email == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		user, err := userRepo.GetUserByEmail(email)
		if err != nil || user == nil || !user.IsAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
