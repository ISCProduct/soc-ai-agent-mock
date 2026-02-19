package repositories

import (
	"Backend/internal/models"

	"gorm.io/gorm"
)

type InterviewReportRepository struct {
	db *gorm.DB
}

func NewInterviewReportRepository(db *gorm.DB) *InterviewReportRepository {
	return &InterviewReportRepository{db: db}
}

func (r *InterviewReportRepository) FindBySessionID(sessionID uint) (*models.InterviewReport, error) {
	var report models.InterviewReport
	if err := r.db.First(&report, "session_id = ?", sessionID).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *InterviewReportRepository) Upsert(report *models.InterviewReport) error {
	return r.db.Save(report).Error
}
