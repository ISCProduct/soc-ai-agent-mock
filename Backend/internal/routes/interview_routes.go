package routes

import (
	"Backend/internal/controllers"
	"net/http"
)

// SetupInterviewRoutes 面接関連のルーティング設定
func SetupInterviewRoutes(interviewController *controllers.InterviewController, realtimeController *controllers.RealtimeController) {
	http.HandleFunc("/api/interviews", interviewController.ListOrCreate)
	http.HandleFunc("/api/interviews/", interviewController.Route)
	http.HandleFunc("/api/realtime/token", realtimeController.Token)
}
