package models

import "time"

type ConversationContext struct {
	ID             uint   `gorm:"primaryKey"`
	UserID         uint   `gorm:"not null;index"`
	SessionID      string `gorm:"size:100;not null;index"`
	IndustryIDs    string `gorm:"type:json"`
	JobCategoryIDs string `gorm:"type:json"`
	AnswerHistory  string `gorm:"type:text"`
	CurrentPhase   string `gorm:"size:50"`
	TotalScore     int    `gorm:"default:0"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
