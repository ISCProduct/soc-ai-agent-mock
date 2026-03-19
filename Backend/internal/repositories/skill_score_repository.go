package repositories

import (
	"Backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SkillScoreRepository スキルスコアのDB操作
type SkillScoreRepository struct {
	db *gorm.DB
}

func NewSkillScoreRepository(db *gorm.DB) *SkillScoreRepository {
	return &SkillScoreRepository{db: db}
}

// ReplaceScores ユーザーのスキルスコアを全件置換
func (r *SkillScoreRepository) ReplaceScores(scores []models.SkillScore) error {
	if len(scores) == 0 {
		return nil
	}
	userID := scores[0].UserID
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.SkillScore{}).Error; err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&scores).Error
	})
}

// GetScores ユーザーのスキルスコア一覧を取得
func (r *SkillScoreRepository) GetScores(userID uint) ([]models.SkillScore, error) {
	var scores []models.SkillScore
	if err := r.db.Where("user_id = ?", userID).Find(&scores).Error; err != nil {
		return nil, err
	}
	return scores, nil
}
