package repository

import (
	"Backend/internal/models"
	"time"
)

// ScheduleRepository は選考スケジュールの永続化インターフェース。
type ScheduleRepository interface {
	Create(event *models.ScheduleEvent) error
	FindByID(id uint) (*models.ScheduleEvent, error)
	Update(event *models.ScheduleEvent) error
	Delete(id uint) error
	ListByUser(userID uint) ([]models.ScheduleEvent, error)
	ListByUserAndRange(userID uint, from, to time.Time) ([]models.ScheduleEvent, error)
}
