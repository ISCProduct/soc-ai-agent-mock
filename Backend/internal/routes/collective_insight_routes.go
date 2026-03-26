package routes

import (
	"Backend/internal/controllers"
	"net/http"
)

func SetupCollectiveInsightRoutes(controller *controllers.CollectiveInsightController) {
	http.HandleFunc("/api/collective-insights/recommendations", controller.Route)
	http.HandleFunc("/api/collective-insights/top-companies", controller.Route)
	http.HandleFunc("/api/collective-insights/consent", controller.Route)
	http.HandleFunc("/api/collective-insights/actions", controller.Route)
}
