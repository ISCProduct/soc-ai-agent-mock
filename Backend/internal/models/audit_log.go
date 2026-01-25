package models

import "time"

// AuditLog stores admin actions for traceability.
type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ActorEmail string    `gorm:"type:varchar(255)" json:"actor_email"`
	Action     string    `gorm:"type:varchar(100);not null" json:"action"`
	TargetType string    `gorm:"type:varchar(50);not null" json:"target_type"`
	TargetID   uint      `json:"target_id"`
	Metadata   string    `gorm:"type:text" json:"metadata"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
