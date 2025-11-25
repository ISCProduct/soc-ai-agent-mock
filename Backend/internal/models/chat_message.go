package models

import "time"

// ChatMessage チャット履歴を保存
type ChatMessage struct {
	ID               uint   `gorm:"primaryKey"`
	SessionID        string `gorm:"size:100;not null;index"`
	UserID           uint   `gorm:"not null;index"`
	Role             string `gorm:"size:20;not null"` // "user" or "assistant"
	Content          string `gorm:"type:text;not null"`
	QuestionWeightID uint   `gorm:"index"` // 質問に対応するQuestionWeightのID
	CreatedAt        time.Time
}
