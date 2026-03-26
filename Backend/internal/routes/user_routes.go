package routes

import (
	"Backend/internal/controllers"
	"net/http"
)

func SetupUserRoutes(profileController *controllers.IntegratedProfileController) {
	http.HandleFunc("/api/user/profile", profileController.GetProfile)
}
