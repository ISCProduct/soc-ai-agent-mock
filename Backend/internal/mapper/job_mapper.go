package mapper

import (
	"Backend/domain/entity"
	"Backend/internal/models"
)

// JobCategoryToEntity GORMモデルをドメインエンティティに変換する
func JobCategoryToEntity(m *models.JobCategory) *entity.JobCategory {
	if m == nil {
		return nil
	}
	return &entity.JobCategory{
		ID:           m.ID,
		ParentID:     m.ParentID,
		Code:         m.Code,
		Name:         m.Name,
		NameEn:       m.NameEn,
		Level:        m.Level,
		Path:         m.Path,
		Description:  m.Description,
		DisplayOrder: m.DisplayOrder,
		IsActive:     m.IsActive,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
