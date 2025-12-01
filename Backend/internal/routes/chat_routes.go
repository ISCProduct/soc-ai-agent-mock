package routes

import (
	"Backend/internal/controllers"
	"net/http"
)

// SetupChatRoutes チャット関連のルーティング設定
func SetupChatRoutes(chatController *controllers.ChatController, questionController *controllers.QuestionController) {
	// チャットエンドポイント
	http.HandleFunc("/api/chat", chatController.Chat)
	http.HandleFunc("/api/chat/history", chatController.GetHistory)
	http.HandleFunc("/api/chat/scores", chatController.GetScores)
	http.HandleFunc("/api/chat/recommendations", chatController.GetRecommendations)
	http.HandleFunc("/api/chat/sessions", chatController.GetSessions)

	// 質問管理エンドポイント
	http.HandleFunc("/api/questions/generate", questionController.GenerateQuestions)
	http.HandleFunc("/api/questions/create", questionController.CreateQuestion)
	http.HandleFunc("/api/questions/list", questionController.GetQuestionsByCategory)
}
