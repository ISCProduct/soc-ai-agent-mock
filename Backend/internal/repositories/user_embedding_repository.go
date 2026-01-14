package repositories

import (
	"Backend/internal/models"

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
