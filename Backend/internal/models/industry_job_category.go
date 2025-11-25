package models

import "time"

type IndustryJobCategory struct {
	ID            uint        `gorm:"primaryKey"`
	IndustryID    uint        `gorm:"not null;index"`
	JobCategoryID uint        `gorm:"not null;index"`
	Industry      Industry    `gorm:"foreignKey:IndustryID"`
	JobCategory   JobCategory `gorm:"foreignKey:JobCategoryID"`
	IsCommon      bool        `gorm:"default:false"`
	DisplayOrder  int         `gorm:"default:0"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
