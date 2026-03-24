package routes

import (
	"Backend/internal/controllers"
	"net/http"
)

// SetupGitHubRoutes GitHub連携関連のルーティング設定
func SetupGitHubRoutes(githubController *controllers.GitHubController) {
	http.HandleFunc("/api/github/profile", githubController.GetProfile)
	http.HandleFunc("/api/github/sync", githubController.Sync)
	http.HandleFunc("/api/github/sync/wait", githubController.SyncAndWait)
	http.HandleFunc("/api/github/skills", githubController.GetSkills)
	http.HandleFunc("/api/github/repo/summaries", githubController.ListRepoSummaries)
	http.HandleFunc("/api/github/repo/summarize", githubController.SummarizeRepo)
}
