package repositories

import (
	"Backend/domain/entity"
	"Backend/domain/mapper"
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
func (r *UserWeightScoreRepository) FindByUserAndSession(userID uint, sessionID string) ([]entity.UserWeightScore, error) {
	var ms []models.UserWeightScore
	err := r.db.Where("user_id = ? AND session_id = ?", userID, sessionID).
		Find(&ms).Error
	if err != nil {
		return nil, err
	}
	return mapper.UserWeightScoresToEntities(ms), nil
}

// FindTopCategories トップNのカテゴリを取得
func (r *UserWeightScoreRepository) FindTopCategories(userID uint, sessionID string, limit int) ([]entity.UserWeightScore, error) {
	var ms []models.UserWeightScore
	err := r.db.Where("user_id = ? AND session_id = ?", userID, sessionID).
		Order("score DESC").
		Limit(limit).
		Find(&ms).Error
	if err != nil {
		return nil, err
	}
	return mapper.UserWeightScoresToEntities(ms), nil
}

// FindByUserSessionAndCategory ユーザー、セッション、カテゴリで検索
func (r *UserWeightScoreRepository) FindByUserSessionAndCategory(userID uint, sessionID, category string) (*entity.UserWeightScore, error) {
	var m models.UserWeightScore
	err := r.db.Where("user_id = ? AND session_id = ? AND weight_category = ?", userID, sessionID, category).
		First(&m).Error
	if err != nil {
		return nil, err
	}
	return mapper.UserWeightScoreToEntity(&m), nil
}

// CountByUserAndSession ユーザーとセッションに紐づくスコア数を取得
func (r *UserWeightScoreRepository) CountByUserAndSession(userID uint, sessionID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.UserWeightScore{}).
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		Count(&count).Error
	return count, err
}

// FindLatestByUser ユーザーの最新セッションのスコアを取得する
func (r *UserWeightScoreRepository) FindLatestByUser(userID uint) ([]entity.UserWeightScore, error) {
	// 最新の session_id を特定
	var latest models.UserWeightScore
	err := r.db.Where("user_id = ?", userID).
		Order("updated_at DESC").
		First(&latest).Error
	if err != nil {
		return nil, err
	}
	return r.FindByUserAndSession(userID, latest.SessionID)
}
