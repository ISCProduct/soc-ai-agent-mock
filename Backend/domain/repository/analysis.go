package repository

import "Backend/internal/models"

// UserWeightScoreRepository はユーザースコアの永続化インターフェース。
type UserWeightScoreRepository interface {
	UpdateScore(userID uint, sessionID, category string, scoreIncrement int) error
	FindByUserAndSession(userID uint, sessionID string) ([]models.UserWeightScore, error)
	FindTopCategories(userID uint, sessionID string, limit int) ([]models.UserWeightScore, error)
	FindByUserSessionAndCategory(userID uint, sessionID, category string) (*models.UserWeightScore, error)
	CountByUserAndSession(userID uint, sessionID string) (int64, error)
}

// AnalysisPhaseRepository は分析フェーズ定義の永続化インターフェース。
type AnalysisPhaseRepository interface {
	FindAll() ([]models.AnalysisPhase, error)
	FindByID(id uint) (*models.AnalysisPhase, error)
	FindByName(name string) (*models.AnalysisPhase, error)
}

// UserAnalysisProgressRepository はユーザーの分析進捗の永続化インターフェース。
type UserAnalysisProgressRepository interface {
	FindByUserAndSession(userID uint, sessionID string) ([]models.UserAnalysisProgress, error)
	FindOrCreate(userID uint, sessionID string, phaseID uint) (*models.UserAnalysisProgress, error)
	Update(progress *models.UserAnalysisProgress) error
	GetCurrentPhase(userID uint, sessionID string) (*models.UserAnalysisProgress, error)
}

// UserCompanyMatchRepository はユーザーと企業のマッチング結果の永続化インターフェース。
type UserCompanyMatchRepository interface {
	CreateOrUpdate(match *models.UserCompanyMatch) error
	FindTopMatchesByUserAndSession(userID uint, sessionID string, limit int) ([]*models.UserCompanyMatch, error)
	FindByID(id uint) (*models.UserCompanyMatch, error)
	MarkAsViewed(matchID uint) error
	ToggleFavorite(matchID uint) error
	MarkAsApplied(matchID uint) error
	FindFavoritesByUser(userID uint, sessionID string) ([]*models.UserCompanyMatch, error)
	GetMatchStatistics(userID uint, sessionID string) (map[string]interface{}, error)
}
