package routes

import (
	"Backend/internal/controllers"
	"net/http"
)

func SetupESRoutes(esRewriteController *controllers.ESRewriteController, esReviewController *controllers.ESReviewController) {
	http.HandleFunc("/api/es/rewrite", esRewriteController.Rewrite)
	http.HandleFunc("/api/es/review", esReviewController.Review)
}
