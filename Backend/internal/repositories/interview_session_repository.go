package repositories

import (
	"Backend/internal/models"
	"time"

	"gorm.io/gorm"
)

type InterviewSessionRepository struct {
	db *gorm.DB
}

func NewInterviewSessionRepository(db *gorm.DB) *InterviewSessionRepository {
	return &InterviewSessionRepository{db: db}
}

func (r *InterviewSessionRepository) Create(session *models.InterviewSession) error {
	return r.db.Create(session).Error
}

func (r *InterviewSessionRepository) FindByID(id uint) (*models.InterviewSession, error) {
	var session models.InterviewSession
	if err := r.db.First(&session, id).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *InterviewSessionRepository) Update(session *models.InterviewSession) error {
	return r.db.Save(session).Error
}

func (r *InterviewSessionRepository) ListByUser(userID uint, limit int, offset int) ([]models.InterviewSession, error) {
	var sessions []models.InterviewSession
	query := r.db.Where("user_id = ?", userID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *InterviewSessionRepository) ListAll(limit int, offset int) ([]models.InterviewSession, error) {
	var sessions []models.InterviewSession
	query := r.db.Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *InterviewSessionRepository) CountByUser(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.InterviewSession{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *InterviewSessionRepository) CountAll() (int64, error) {
	var count int64
	err := r.db.Model(&models.InterviewSession{}).Count(&count).Error
	return count, err
}

func (r *InterviewSessionRepository) CountByUserAndDay(userID uint, day time.Time) (int64, error) {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	end := start.Add(24 * time.Hour)
	var count int64
	err := r.db.Model(&models.InterviewSession{}).
		Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, start, end).
		Count(&count).Error
	return count, err
}
