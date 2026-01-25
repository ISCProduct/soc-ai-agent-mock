package repositories

import (
	"Backend/internal/models"

	"gorm.io/gorm"
)

type CompanyPopularityRepository struct {
	db *gorm.DB
}

func NewCompanyPopularityRepository(db *gorm.DB) *CompanyPopularityRepository {
	return &CompanyPopularityRepository{db: db}
}

func (r *CompanyPopularityRepository) Create(record *models.CompanyPopularityRecord) error {
	return r.db.Create(record).Error
}

func (r *CompanyPopularityRepository) ListByCompany(companyID uint, limit int) ([]models.CompanyPopularityRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	var records []models.CompanyPopularityRecord
	err := r.db.Where("company_id = ?", companyID).
		Order("fetched_at desc").
		Limit(limit).
		Find(&records).Error
	return records, err
}
