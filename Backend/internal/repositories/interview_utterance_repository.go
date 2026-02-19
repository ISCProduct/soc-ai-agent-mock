package repositories

import (
	"Backend/internal/models"

	"gorm.io/gorm"
)

type InterviewUtteranceRepository struct {
	db *gorm.DB
}

func NewInterviewUtteranceRepository(db *gorm.DB) *InterviewUtteranceRepository {
	return &InterviewUtteranceRepository{db: db}
}

func (r *InterviewUtteranceRepository) Create(utterance *models.InterviewUtterance) error {
	return r.db.Create(utterance).Error
}

func (r *InterviewUtteranceRepository) FindBySessionID(sessionID uint) ([]models.InterviewUtterance, error) {
	var utterances []models.InterviewUtterance
	err := r.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&utterances).Error
	return utterances, err
}
