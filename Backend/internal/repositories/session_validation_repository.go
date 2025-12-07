package repositories

import (
	"Backend/internal/models"
	"time"

	"gorm.io/gorm"
)

type SessionValidationRepository struct {
	db *gorm.DB
}

func NewSessionValidationRepository(db *gorm.DB) *SessionValidationRepository {
	return &SessionValidationRepository{db: db}
}

// GetOrCreate セッションのバリデーション情報を取得または作成
func (r *SessionValidationRepository) GetOrCreate(sessionID string) (*models.SessionValidation, error) {
	var validation models.SessionValidation
	err := r.db.Where("session_id = ?", sessionID).First(&validation).Error
	if err == gorm.ErrRecordNotFound {
		validation = models.SessionValidation{
			SessionID:          sessionID,
			InvalidAnswerCount: 0,
			IsTerminated:       false,
		}
		if err := r.db.Create(&validation).Error; err != nil {
			return nil, err
		}
		return &validation, nil
	}
	if err != nil {
		return nil, err
	}
	return &validation, nil
}

// IncrementInvalidCount 無効回答カウントをインクリメント
func (r *SessionValidationRepository) IncrementInvalidCount(sessionID string) (*models.SessionValidation, error) {
	validation, err := r.GetOrCreate(sessionID)
	if err != nil {
		return nil, err
	}

	validation.InvalidAnswerCount++
	now := time.Now()
	validation.LastInvalidAnswerTime = &now

	if err := r.db.Save(validation).Error; err != nil {
		return nil, err
	}

	return validation, nil
}

// ResetInvalidCount 無効回答カウントをリセット
func (r *SessionValidationRepository) ResetInvalidCount(sessionID string) error {
	validation, err := r.GetOrCreate(sessionID)
	if err != nil {
		return err
	}

	validation.InvalidAnswerCount = 0
	return r.db.Save(validation).Error
}

// TerminateSession セッションを強制終了
func (r *SessionValidationRepository) TerminateSession(sessionID string) error {
	validation, err := r.GetOrCreate(sessionID)
	if err != nil {
		return err
	}

	validation.IsTerminated = true
	return r.db.Save(validation).Error
}

// IsTerminated セッションが終了しているかチェック
func (r *SessionValidationRepository) IsTerminated(sessionID string) (bool, error) {
	validation, err := r.GetOrCreate(sessionID)
	if err != nil {
		return false, err
	}
	return validation.IsTerminated, nil
}
