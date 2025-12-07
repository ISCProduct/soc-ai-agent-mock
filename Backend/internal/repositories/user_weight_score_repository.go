package repositories

import (
	"Backend/internal/models"

	"gorm.io/gorm"
)

type UserWeightScoreRepository struct {
	db *gorm.DB
}

func NewUserWeightScoreRepository(db *gorm.DB) *UserWeightScoreRepository {
	return &UserWeightScoreRepository{db: db}
}

// UpdateScore スコアを更新または作成
func (r *UserWeightScoreRepository) UpdateScore(userID uint, sessionID, category string, scoreIncrement int) error {
	var score models.UserWeightScore
	err := r.db.Where("user_id = ? AND session_id = ? AND weight_category = ?",
		userID, sessionID, category).First(&score).Error

	if err == gorm.ErrRecordNotFound {
		// 新規作成
		score = models.UserWeightScore{
			UserID:         userID,
			SessionID:      sessionID,
			WeightCategory: category,
			Score:          scoreIncrement,
		}
		return r.db.Create(&score).Error
	} else if err != nil {
		return err
	}

	// 既存レコードを更新
	return r.db.Model(&score).Update("score", gorm.Expr("score + ?", scoreIncrement)).Error
}

// FindByUserAndSession ユーザーとセッションに紐づく全スコアを取得
func (r *UserWeightScoreRepository) FindByUserAndSession(userID uint, sessionID string) ([]models.UserWeightScore, error) {
	var scores []models.UserWeightScore
	err := r.db.Where("user_id = ? AND session_id = ?", userID, sessionID).
		Find(&scores).Error
	return scores, err
}

// FindTopCategories トップNのカテゴリを取得
func (r *UserWeightScoreRepository) FindTopCategories(userID uint, sessionID string, limit int) ([]models.UserWeightScore, error) {
	var scores []models.UserWeightScore
	err := r.db.Where("user_id = ? AND session_id = ?", userID, sessionID).
		Order("score DESC").
		Limit(limit).
		Find(&scores).Error
	return scores, err
}

// FindByUserSessionAndCategory ユーザー、セッション、カテゴリで検索
func (r *UserWeightScoreRepository) FindByUserSessionAndCategory(userID uint, sessionID, category string) (*models.UserWeightScore, error) {
	var score models.UserWeightScore
	err := r.db.Where("user_id = ? AND session_id = ? AND weight_category = ?", userID, sessionID, category).
		First(&score).Error
	if err != nil {
		return nil, err
	}
	return &score, nil
}
