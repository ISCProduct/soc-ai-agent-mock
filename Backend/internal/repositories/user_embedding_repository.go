package repositories

import (
	"Backend/internal/models"
	"errors"

	"gorm.io/gorm"
)

type UserEmbeddingRepository struct {
	db *gorm.DB
}

func NewUserEmbeddingRepository(db *gorm.DB) *UserEmbeddingRepository {
	return &UserEmbeddingRepository{db: db}
}

func (r *UserEmbeddingRepository) FindByUserAndSession(userID uint, sessionID string) (*models.UserEmbedding, error) {
	var embedding models.UserEmbedding
	err := r.db.Where("user_id = ? AND session_id = ?", userID, sessionID).First(&embedding).Error
	if err != nil {
		return nil, err
	}
	return &embedding, nil
}

func (r *UserEmbeddingRepository) Upsert(userID uint, sessionID, profileText, embedding string) error {
	var existing models.UserEmbedding
	err := r.db.Where("user_id = ? AND session_id = ?", userID, sessionID).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			record := models.UserEmbedding{
				UserID:      userID,
				SessionID:   sessionID,
				ProfileText: profileText,
				Embedding:   embedding,
			}
			return r.db.Create(&record).Error
		}
		return err
	}

	existing.ProfileText = profileText
	existing.Embedding = embedding
	return r.db.Save(&existing).Error
}
