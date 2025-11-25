package models

import "time"

type Industry struct {
	ID           uint       `gorm:"primaryKey"`
	ParentID     *uint      `gorm:"index"`
	Parent       *Industry  `gorm:"foreignKey:ParentID"`
	Children     []Industry `gorm:"foreignKey:ParentID"`
	Code         string     `gorm:"uniqueIndex;size:100;not null"`
	Name         string     `gorm:"size:255;not null"`
	NameEn       string     `gorm:"size:255"`
	Level        int        `gorm:"default:0"`
	Path         string     `gorm:"size:1000"`
	Description  string     `gorm:"type:text"`
	DisplayOrder int        `gorm:"default:0"`
	IsActive     bool       `gorm:"default:true"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
