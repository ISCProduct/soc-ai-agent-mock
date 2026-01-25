package routes

import (
	"Backend/internal/controllers"
	"net/http"
)

func SetupResumeRoutes(resumeController *controllers.ResumeController) {
	http.HandleFunc("/api/resume/upload", resumeController.Upload)
	http.HandleFunc("/api/resume/review", resumeController.Review)
	http.HandleFunc("/api/resume/annotated", resumeController.Annotated)
}
