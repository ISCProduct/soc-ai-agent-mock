package models

import "time"

// InterviewReport 面接後の要約・評価
type InterviewReport struct {
	SessionID         uint   `gorm:"primaryKey"`
	SummaryText       string `gorm:"type:text"`
	ScoresJSON        string `gorm:"type:json"`
	EvidenceJSON      string `gorm:"type:json"`
	StrengthsJSON     string `gorm:"type:json"`
	ImprovementsJSON  string `gorm:"type:json"`
	TeacherReportJSON string `gorm:"type:json"` // 教員用詳細レポート（指導コメント・エビデンス等）
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
