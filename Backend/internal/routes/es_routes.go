package routes

import (
	"Backend/internal/controllers"
	"net/http"
)

func SetupESRoutes(esRewriteController *controllers.ESRewriteController) {
	http.HandleFunc("/api/es/rewrite", esRewriteController.Rewrite)
}
