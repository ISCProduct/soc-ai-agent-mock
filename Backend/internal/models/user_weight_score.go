package models

import "time"

// UserWeightScore ユーザーごとの重みカテゴリ別スコア
type UserWeightScore struct {
	ID             uint   `gorm:"primaryKey"`
	UserID         uint   `gorm:"not null;index:idx_user_category"`
	SessionID      string `gorm:"size:100;not null;index"`
	WeightCategory string `gorm:"size:100;not null;index:idx_user_category"`
	Score          int    `gorm:"default:0"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
