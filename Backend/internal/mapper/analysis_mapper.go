package mapper

import (
	"Backend/domain/entity"
	"Backend/internal/models"
)

// AnalysisPhaseToEntity GORMモデルをドメインエンティティに変換する
func AnalysisPhaseToEntity(m *models.AnalysisPhase) *entity.AnalysisPhase {
	if m == nil {
		return nil
	}
	return &entity.AnalysisPhase{
		ID:           m.ID,
		PhaseName:    m.PhaseName,
		DisplayName:  m.DisplayName,
		PhaseOrder:   m.PhaseOrder,
		Description:  m.Description,
		MinQuestions: m.MinQuestions,
		MaxQuestions: m.MaxQuestions,
	}
}

// UserAnalysisProgressToEntity GORMモデルをドメインエンティティに変換する
func UserAnalysisProgressToEntity(m *models.UserAnalysisProgress) *entity.UserAnalysisProgress {
	if m == nil {
		return nil
	}
	e := &entity.UserAnalysisProgress{
		ID:              m.ID,
		UserID:          m.UserID,
		SessionID:       m.SessionID,
		PhaseID:         m.PhaseID,
		QuestionsAsked:  m.QuestionsAsked,
		ValidAnswers:    m.ValidAnswers,
		InvalidAnswers:  m.InvalidAnswers,
		CompletionScore: m.CompletionScore,
		IsCompleted:     m.IsCompleted,
		CompletedAt:     m.CompletedAt,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
	if m.Phase != nil {
		e.Phase = AnalysisPhaseToEntity(m.Phase)
	}
	return e
}

// UserWeightScoreToEntity GORMモデルをドメインエンティティに変換する
func UserWeightScoreToEntity(m *models.UserWeightScore) *entity.UserWeightScore {
	if m == nil {
		return nil
	}
	return &entity.UserWeightScore{
		ID:             m.ID,
		UserID:         m.UserID,
		SessionID:      m.SessionID,
		WeightCategory: m.WeightCategory,
		Score:          m.Score,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

// UserWeightScoresToEntities スライスを一括変換する
func UserWeightScoresToEntities(ms []models.UserWeightScore) []entity.UserWeightScore {
	result := make([]entity.UserWeightScore, 0, len(ms))
	for i := range ms {
		if e := UserWeightScoreToEntity(&ms[i]); e != nil {
			result = append(result, *e)
		}
	}
	return result
}
