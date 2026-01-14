package repositories

import (
	"Backend/internal/models"
	"time"

	"gorm.io/gorm"
)

type UserCompanyMatchRepository struct {
	db *gorm.DB
}

func NewUserCompanyMatchRepository(db *gorm.DB) *UserCompanyMatchRepository {
	return &UserCompanyMatchRepository{db: db}
}

// CreateOrUpdate マッチング結果を作成または更新
func (r *UserCompanyMatchRepository) CreateOrUpdate(match *models.UserCompanyMatch) error {
	var existing models.UserCompanyMatch
	err := r.db.Where("user_id = ? AND session_id = ? AND company_id = ?",
		match.UserID, match.SessionID, match.CompanyID).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 新規作成
		return r.db.Create(match).Error
	} else if err != nil {
		return err
	}

	// 更新（ID以外のフィールドを更新）
	match.ID = existing.ID
	match.CreatedAt = existing.CreatedAt
	match.UpdatedAt = time.Now()
	match.IsViewed = existing.IsViewed
	match.IsFavorited = existing.IsFavorited
	match.IsApplied = existing.IsApplied
	return r.db.Save(match).Error
}

// FindTopMatchesByUserAndSession マッチング度の高い順に企業を取得
func (r *UserCompanyMatchRepository) FindTopMatchesByUserAndSession(
	userID uint, sessionID string, limit int,
) ([]*models.UserCompanyMatch, error) {
	var matches []*models.UserCompanyMatch
	err := r.db.Where("user_id = ? AND session_id = ?", userID, sessionID).
		Order("match_score DESC").
		Limit(limit).
		Preload("Company").
		Preload("JobPosition").
		Find(&matches).Error

	if err != nil {
		return nil, err
	}
	return matches, nil
}

// FindByID IDでマッチング結果を取得
func (r *UserCompanyMatchRepository) FindByID(id uint) (*models.UserCompanyMatch, error) {
	var match models.UserCompanyMatch
	err := r.db.Preload("Company").
		Preload("JobPosition").
		First(&match, id).Error
	if err != nil {
		return nil, err
	}
	return &match, nil
}

// MarkAsViewed 閲覧済みにする
func (r *UserCompanyMatchRepository) MarkAsViewed(matchID uint) error {
	return r.db.Model(&models.UserCompanyMatch{}).
		Where("id = ?", matchID).
		Update("is_viewed", true).Error
}

// ToggleFavorite お気に入りをトグル
func (r *UserCompanyMatchRepository) ToggleFavorite(matchID uint) error {
	var match models.UserCompanyMatch
	if err := r.db.First(&match, matchID).Error; err != nil {
		return err
	}
	return r.db.Model(&match).Update("is_favorited", !match.IsFavorited).Error
}

// MarkAsApplied 応募済みにする
func (r *UserCompanyMatchRepository) MarkAsApplied(matchID uint) error {
	return r.db.Model(&models.UserCompanyMatch{}).
		Where("id = ?", matchID).
		Update("is_applied", true).Error
}

// FindFavoritesByUser ユーザーのお気に入り企業を取得
func (r *UserCompanyMatchRepository) FindFavoritesByUser(userID uint, sessionID string) ([]*models.UserCompanyMatch, error) {
	var matches []*models.UserCompanyMatch
	err := r.db.Where("user_id = ? AND session_id = ? AND is_favorited = ?", userID, sessionID, true).
		Order("match_score DESC").
		Preload("Company").
		Preload("JobPosition").
		Find(&matches).Error
	return matches, err
}

// GetMatchStatistics マッチング統計情報を取得
func (r *UserCompanyMatchRepository) GetMatchStatistics(userID uint, sessionID string) (map[string]interface{}, error) {
	var result struct {
		TotalMatches   int64
		ViewedCount    int64
		FavoritedCount int64
		AppliedCount   int64
		AvgMatchScore  float64
	}

	err := r.db.Model(&models.UserCompanyMatch{}).
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		Count(&result.TotalMatches).Error
	if err != nil {
		return nil, err
	}

	r.db.Model(&models.UserCompanyMatch{}).
		Where("user_id = ? AND session_id = ? AND is_viewed = ?", userID, sessionID, true).
		Count(&result.ViewedCount)

	r.db.Model(&models.UserCompanyMatch{}).
		Where("user_id = ? AND session_id = ? AND is_favorited = ?", userID, sessionID, true).
		Count(&result.FavoritedCount)

	r.db.Model(&models.UserCompanyMatch{}).
		Where("user_id = ? AND session_id = ? AND is_applied = ?", userID, sessionID, true).
		Count(&result.AppliedCount)

	r.db.Model(&models.UserCompanyMatch{}).
		Select("AVG(match_score)").
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		Scan(&result.AvgMatchScore)

	stats := map[string]interface{}{
		"total_matches":   result.TotalMatches,
		"viewed_count":    result.ViewedCount,
		"favorited_count": result.FavoritedCount,
		"applied_count":   result.AppliedCount,
		"avg_match_score": result.AvgMatchScore,
	}

	return stats, nil
}
