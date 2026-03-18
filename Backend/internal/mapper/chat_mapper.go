package mapper

import (
	"Backend/domain/entity"
	"Backend/internal/models"
)

// ChatMessageToEntity GORMモデルをドメインエンティティに変換する
func ChatMessageToEntity(m *models.ChatMessage) *entity.ChatMessage {
	if m == nil {
		return nil
	}
	return &entity.ChatMessage{
		ID:               m.ID,
		SessionID:        m.SessionID,
		UserID:           m.UserID,
		Role:             m.Role,
		Content:          m.Content,
		QuestionWeightID: m.QuestionWeightID,
		CreatedAt:        m.CreatedAt,
	}
}

// ChatMessagesToEntities スライスを一括変換する
func ChatMessagesToEntities(ms []models.ChatMessage) []entity.ChatMessage {
	result := make([]entity.ChatMessage, 0, len(ms))
	for i := range ms {
		if e := ChatMessageToEntity(&ms[i]); e != nil {
			result = append(result, *e)
		}
	}
	return result
}
