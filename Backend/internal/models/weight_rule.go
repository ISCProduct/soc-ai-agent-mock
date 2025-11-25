package models

import "time"

type WeightRule struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"size:255;not null"`
	Condition   string `gorm:"type:json;not null"`
	WeightBoost int    `gorm:"not null"`
	Description string `gorm:"type:text"`
	Priority    int    `gorm:"default:0"`
	IsActive    bool   `gorm:"default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
