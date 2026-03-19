package repositories

import (
	"Backend/domain/entity"
	"Backend/domain/mapper"
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

func (r *PendingRegistrationRepository) Create(p *entity.PendingRegistration) error {
	m := mapper.PendingRegistrationFromEntity(p)
	return r.db.Create(m).Error
}

func (r *PendingRegistrationRepository) FindByToken(token string) (*entity.PendingRegistration, error) {
	var m models.PendingRegistration
	err := r.db.Where("token = ? AND expires_at > ?", token, time.Now()).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mapper.PendingRegistrationToEntity(&m), nil
}

func (r *PendingRegistrationRepository) DeleteByEmail(email string) error {
	return r.db.Where("email = ?", email).Delete(&models.PendingRegistration{}).Error
}

func (r *PendingRegistrationRepository) DeleteExpired() error {
	return r.db.Where("expires_at <= ?", time.Now()).Delete(&models.PendingRegistration{}).Error
}
