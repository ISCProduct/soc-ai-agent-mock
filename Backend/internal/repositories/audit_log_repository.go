package repositories

import (
	"Backend/internal/models"

	"gorm.io/gorm"
)

type AuditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(log *models.AuditLog) error {
	return r.db.Create(log).Error
}

func (r *AuditLogRepository) List(limit int) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	query := r.db.Order("created_at desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&logs).Error
	return logs, err
}
