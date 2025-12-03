package repositories

import (
	"Backend/internal/models"
	"gorm.io/gorm"
)

type AnalysisPhaseRepository struct {
	db *gorm.DB
}

func NewAnalysisPhaseRepository(db *gorm.DB) *AnalysisPhaseRepository {
	return &AnalysisPhaseRepository{db: db}
}

// FindAll 全フェーズを順序順に取得
func (r *AnalysisPhaseRepository) FindAll() ([]models.AnalysisPhase, error) {
	var phases []models.AnalysisPhase
	err := r.db.Order("phase_order ASC").Find(&phases).Error
	return phases, err
}

// FindByID IDでフェーズを取得
func (r *AnalysisPhaseRepository) FindByID(id uint) (*models.AnalysisPhase, error) {
	var phase models.AnalysisPhase
	err := r.db.First(&phase, id).Error
	return &phase, err
}

// FindByName 名前でフェーズを取得
func (r *AnalysisPhaseRepository) FindByName(name string) (*models.AnalysisPhase, error) {
	var phase models.AnalysisPhase
	err := r.db.Where("phase_name = ?", name).First(&phase).Error
	return &phase, err
}

type UserAnalysisProgressRepository struct {
	db *gorm.DB
}

func NewUserAnalysisProgressRepository(db *gorm.DB) *UserAnalysisProgressRepository {
	return &UserAnalysisProgressRepository{db: db}
}

// FindByUserAndSession ユーザーとセッションで進捗を取得
func (r *UserAnalysisProgressRepository) FindByUserAndSession(userID uint, sessionID string) ([]models.UserAnalysisProgress, error) {
	var progress []models.UserAnalysisProgress
	err := r.db.Preload("Phase").
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		Order("phase_id ASC").
		Find(&progress).Error
	return progress, err
}

// FindOrCreate フェーズの進捗を取得または作成
func (r *UserAnalysisProgressRepository) FindOrCreate(userID uint, sessionID string, phaseID uint) (*models.UserAnalysisProgress, error) {
	var progress models.UserAnalysisProgress
	err := r.db.Where("user_id = ? AND session_id = ? AND phase_id = ?", userID, sessionID, phaseID).
		FirstOrCreate(&progress, models.UserAnalysisProgress{
			UserID:    userID,
			SessionID: sessionID,
			PhaseID:   phaseID,
		}).Error
	if err != nil {
		return nil, err
	}
	// Phaseをロード
	r.db.Preload("Phase").First(&progress, progress.ID)
	return &progress, nil
}

// Update 進捗を更新
func (r *UserAnalysisProgressRepository) Update(progress *models.UserAnalysisProgress) error {
	return r.db.Save(progress).Error
}

// GetCurrentPhase 現在進行中のフェーズを取得
func (r *UserAnalysisProgressRepository) GetCurrentPhase(userID uint, sessionID string) (*models.UserAnalysisProgress, error) {
	var progress models.UserAnalysisProgress
	err := r.db.Preload("Phase").
		Where("user_id = ? AND session_id = ? AND is_completed = ?", userID, sessionID, false).
		Order("phase_id ASC").
		First(&progress).Error
	return &progress, err
}
