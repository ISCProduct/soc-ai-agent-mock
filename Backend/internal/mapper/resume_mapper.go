package mapper

import (
	"Backend/domain/entity"
	"Backend/internal/models"
)

// ResumeDocumentToEntity GORMモデルをドメインエンティティに変換する
func ResumeDocumentToEntity(m *models.ResumeDocument) *entity.ResumeDocument {
	if m == nil {
		return nil
	}
	return &entity.ResumeDocument{
		ID:               m.ID,
		UserID:           m.UserID,
		SessionID:        m.SessionID,
		SourceType:       m.SourceType,
		SourceURL:        m.SourceURL,
		OriginalFilename: m.OriginalFilename,
		StoredPath:       m.StoredPath,
		NormalizedPath:   m.NormalizedPath,
		AnnotatedPath:    m.AnnotatedPath,
		Status:           m.Status,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

// ResumeReviewToEntity GORMモデルをドメインエンティティに変換する
func ResumeReviewToEntity(m *models.ResumeReview) *entity.ResumeReview {
	if m == nil {
		return nil
	}
	return &entity.ResumeReview{
		ID:         m.ID,
		DocumentID: m.DocumentID,
		Score:      m.Score,
		Summary:    m.Summary,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

// ResumeReviewItemToEntity GORMモデルをドメインエンティティに変換する
func ResumeReviewItemToEntity(m *models.ResumeReviewItem) *entity.ResumeReviewItem {
	if m == nil {
		return nil
	}
	return &entity.ResumeReviewItem{
		ID:         m.ID,
		ReviewID:   m.ReviewID,
		PageNumber: m.PageNumber,
		BBox:       m.BBox,
		Severity:   m.Severity,
		Message:    m.Message,
		Suggestion: m.Suggestion,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}
