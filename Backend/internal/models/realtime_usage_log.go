package models

import "time"

// RealtimeUsageLog records usage for OpenAI Realtime interview sessions.
type RealtimeUsageLog struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	UserID             uint       `gorm:"index;not null" json:"user_id"`
	InterviewSessionID uint       `gorm:"index;not null" json:"interview_session_id"`
	Status             string     `gorm:"size:16;index;not null;default:'active'" json:"status"`
	StartedAt          time.Time  `gorm:"not null;index" json:"started_at"`
	EndedAt            *time.Time `gorm:"index" json:"ended_at,omitempty"`
	DurationSeconds    int        `gorm:"not null;default:0" json:"duration_seconds"`
	CostUSD            float64    `gorm:"type:decimal(10,4);default:0" json:"cost_usd"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}
