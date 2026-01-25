package repositories

import (
	"Backend/internal/models"

	"gorm.io/gorm"
)

type CompanyRelationRepository struct {
	db *gorm.DB
}

func NewCompanyRelationRepository(db *gorm.DB) *CompanyRelationRepository {
	return &CompanyRelationRepository{db: db}
}

func (r *CompanyRelationRepository) UpsertBusinessRelation(fromID, toID uint, relationType, description string) error {
	relation := models.CompanyRelation{
		FromID:       &fromID,
		ToID:         &toID,
		RelationType: relationType,
		Description:  description,
		IsActive:     true,
	}
	return r.db.FirstOrCreate(&relation, models.CompanyRelation{
		FromID:       &fromID,
		ToID:         &toID,
		RelationType: relationType,
		Description:  description,
	}).Error
}
