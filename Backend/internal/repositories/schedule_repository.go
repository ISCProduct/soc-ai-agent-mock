package repositories

import (
	"Backend/internal/models"
	"time"

	"gorm.io/gorm"
)

type ScheduleRepository struct {
	db *gorm.DB
}

func NewScheduleRepository(db *gorm.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

func (r *ScheduleRepository) Create(event *models.ScheduleEvent) error {
	return r.db.Create(event).Error
}

func (r *ScheduleRepository) FindByID(id uint) (*models.ScheduleEvent, error) {
	var event models.ScheduleEvent
	if err := r.db.First(&event, id).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *ScheduleRepository) Update(event *models.ScheduleEvent) error {
	return r.db.Save(event).Error
}

func (r *ScheduleRepository) Delete(id uint) error {
	return r.db.Delete(&models.ScheduleEvent{}, id).Error
}

func (r *ScheduleRepository) ListByUser(userID uint) ([]models.ScheduleEvent, error) {
	var events []models.ScheduleEvent
	if err := r.db.Where("user_id = ?", userID).Order("scheduled_at asc").Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

func (r *ScheduleRepository) ListByUserAndRange(userID uint, from, to time.Time) ([]models.ScheduleEvent, error) {
	var events []models.ScheduleEvent
	if err := r.db.
		Where("user_id = ? AND scheduled_at >= ? AND scheduled_at < ?", userID, from, to).
		Order("scheduled_at asc").
		Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}
