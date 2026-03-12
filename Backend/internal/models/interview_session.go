package models

import "time"

// InterviewSession 面接セッション
type InterviewSession struct {
	ID               uint       `gorm:"primaryKey"`
	UserID           uint       `gorm:"index;not null"`
	Status           string     `gorm:"size:32;index;not null"`
	StartedAt        *time.Time `gorm:"index"`
	EndedAt          *time.Time `gorm:"index"`
	EstimatedCostUSD float64    `gorm:"type:decimal(10,4);default:0"`
	TemplateVersion  string     `gorm:"size:32;default:'v1'"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time `gorm:"index"`
}
