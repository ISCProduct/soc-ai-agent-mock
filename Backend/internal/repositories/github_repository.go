package repositories

import (
	"Backend/internal/models"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GitHubRepository GitHub連携データのDB操作
type GitHubRepository struct {
	db *gorm.DB
}

func NewGitHubRepository(db *gorm.DB) *GitHubRepository {
	return &GitHubRepository{db: db}
}

// UpsertProfile GitHubプロフィールを保存/更新
func (r *GitHubRepository) UpsertProfile(profile *models.GitHubProfile) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"git_hub_login", "access_token", "total_contributions", "public_repos", "followers", "following", "synced_at", "updated_at"}),
	}).Create(profile).Error
}

// GetProfile ユーザーIDでGitHubプロフィールを取得
func (r *GitHubRepository) GetProfile(userID uint) (*models.GitHubProfile, error) {
	var profile models.GitHubProfile
	if err := r.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

// ReplaceRepositories ユーザーのリポジトリ一覧を全件置換
func (r *GitHubRepository) ReplaceRepositories(userID uint, repos []models.GitHubRepo) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.GitHubRepo{}).Error; err != nil {
			return err
		}
		if len(repos) == 0 {
			return nil
		}
		return tx.Create(&repos).Error
	})
}

// GetRepositories ユーザーのリポジトリ一覧を取得（star数降順）
func (r *GitHubRepository) GetRepositories(userID uint) ([]models.GitHubRepo, error) {
	var repos []models.GitHubRepo
	if err := r.db.Where("user_id = ?", userID).Order("stars desc").Find(&repos).Error; err != nil {
		return nil, err
	}
	return repos, nil
}

// ReplaceLanguageStats ユーザーの言語使用比率を全件置換
func (r *GitHubRepository) ReplaceLanguageStats(userID uint, stats []models.GitHubLanguageStat) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.GitHubLanguageStat{}).Error; err != nil {
			return err
		}
		if len(stats) == 0 {
			return nil
		}
		return tx.Create(&stats).Error
	})
}

// GetLanguageStats ユーザーの言語使用比率を取得（使用量降順）
func (r *GitHubRepository) GetLanguageStats(userID uint) ([]models.GitHubLanguageStat, error) {
	var stats []models.GitHubLanguageStat
	if err := r.db.Where("user_id = ?", userID).Order("bytes desc").Find(&stats).Error; err != nil {
		return nil, err
	}
	return stats, nil
}

// UpdateSyncedAt 最終同期日時を更新
func (r *GitHubRepository) UpdateSyncedAt(userID uint, t time.Time) error {
	return r.db.Model(&models.GitHubProfile{}).
		Where("user_id = ?", userID).
		Update("synced_at", t).Error
}
