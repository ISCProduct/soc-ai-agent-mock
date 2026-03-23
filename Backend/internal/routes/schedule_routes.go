package routes

import (
	"Backend/internal/controllers"
	"net/http"
)

func SetupScheduleRoutes(scheduleController *controllers.ScheduleController) {
	http.HandleFunc("/api/schedule/export/ics", scheduleController.ExportICS)
	http.HandleFunc("/api/schedule/", scheduleController.RouteByID)
	http.HandleFunc("/api/schedule", scheduleController.RouteList)
}
