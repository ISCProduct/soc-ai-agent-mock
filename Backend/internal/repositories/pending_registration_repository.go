package repositories

import (
	"Backend/internal/models"
	"time"

	"gorm.io/gorm"
)

type PendingRegistrationRepository struct {
	db *gorm.DB
}

func NewPendingRegistrationRepository(db *gorm.DB) *PendingRegistrationRepository {
	return &PendingRegistrationRepository{db: db}
}

func (r *PendingRegistrationRepository) Create(p *models.PendingRegistration) error {
	return r.db.Create(p).Error
}

func (r *PendingRegistrationRepository) FindByToken(token string) (*models.PendingRegistration, error) {
	var p models.PendingRegistration
	err := r.db.Where("token = ? AND expires_at > ?", token, time.Now()).First(&p).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &p, err
}

func (r *PendingRegistrationRepository) DeleteByEmail(email string) error {
	return r.db.Where("email = ?", email).Delete(&models.PendingRegistration{}).Error
}

func (r *PendingRegistrationRepository) DeleteExpired() error {
	return r.db.Where("expires_at <= ?", time.Now()).Delete(&models.PendingRegistration{}).Error
}
