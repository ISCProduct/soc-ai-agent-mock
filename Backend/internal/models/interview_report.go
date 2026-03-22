package models

import "time"

// InterviewReport 面接後の要約・評価
type InterviewReport struct {
	SessionID       uint   `gorm:"primaryKey"`
	SummaryText     string `gorm:"type:text"`
	ScoresJSON      string `gorm:"type:json"`
	EvidenceJSON    string `gorm:"type:json"`
	StrengthsJSON   string `gorm:"type:json"`
	ImprovementsJSON string `gorm:"type:json"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
