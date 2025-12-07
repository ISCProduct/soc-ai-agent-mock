package models

import "time"

// SessionValidation セッションごとのバリデーション状態を管理
type SessionValidation struct {
	ID                    uint       `gorm:"primaryKey"`
	SessionID             string     `gorm:"type:varchar(255);uniqueIndex;not null"`
	InvalidAnswerCount    int        `gorm:"default:0"`
	IsTerminated          bool       `gorm:"default:false"`
	LastInvalidAnswerTime *time.Time `gorm:"type:datetime;default:null"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
