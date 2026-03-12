package models

import "time"

// InterviewUtterance 面接中の発話ログ
type InterviewUtterance struct {
	ID        uint      `gorm:"primaryKey"`
	SessionID uint      `gorm:"index;not null"`
	Role      string    `gorm:"size:16;index;not null"` // user / ai
	Text      string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"index"`
}
