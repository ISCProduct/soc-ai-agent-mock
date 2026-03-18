package mapper

import (
	"Backend/domain/entity"
	"Backend/internal/models"
)

// InterviewSessionToEntity GORMモデルをドメインエンティティに変換する
func InterviewSessionToEntity(m *models.InterviewSession) *entity.InterviewSession {
	if m == nil {
		return nil
	}
	return &entity.InterviewSession{
		ID:               m.ID,
		UserID:           m.UserID,
		Status:           m.Status,
		Language:         m.Language,
		StartedAt:        m.StartedAt,
		EndedAt:          m.EndedAt,
		EstimatedCostUSD: m.EstimatedCostUSD,
		TemplateVersion:  m.TemplateVersion,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}
