package models

import "time"

type AIQuestionTemplate struct {
	ID          uint   `gorm:"primaryKey"`
	Category    string `gorm:"size:50;not null;index"`
	Prompt      string `gorm:"type:text;not null"`
	BaseWeight  int    `gorm:"default:5"`
	ContextKeys string `gorm:"type:json"`
	IsActive    bool   `gorm:"default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
