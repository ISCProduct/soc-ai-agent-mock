package routes

import (
	"Backend/internal/controllers"
	"net/http"
	"strings"
)

// SetupApplicationRoutes 応募・選考ステータス管理のルーティング設定
func SetupApplicationRoutes(appController *controllers.ApplicationController) {
	// POST /api/applications       → 応募登録
	// GET  /api/applications       → 応募一覧取得
	// GET  /api/applications/correlation → 相関分析データ
	// PUT  /api/applications/{id}  → ステータス更新
	http.HandleFunc("/api/applications", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			appController.Apply(w, r)
		case http.MethodGet:
			appController.List(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/applications/correlation", appController.GetCorrelation)

	http.HandleFunc("/api/applications/", func(w http.ResponseWriter, r *http.Request) {
		// /api/applications/correlation は上で処理済みなのでスキップ
		if strings.HasSuffix(r.URL.Path, "/correlation") {
			appController.GetCorrelation(w, r)
			return
		}
		if r.Method == http.MethodPut {
			appController.UpdateStatus(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
}
