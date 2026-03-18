package entity

import "time"

// ChatMessage チャット履歴エンティティ
type ChatMessage struct {
	ID               uint
	SessionID        string
	UserID           uint
	Role             string // "user" or "assistant"
	Content          string
	QuestionWeightID uint
	CreatedAt        time.Time
}

// IsUserMessage ユーザー発言かどうか
func (m *ChatMessage) IsUserMessage() bool {
	return m.Role == "user"
}

// IsAssistantMessage アシスタント発言かどうか
func (m *ChatMessage) IsAssistantMessage() bool {
	return m.Role == "assistant"
}

// ChatSession チャットセッションエンティティ
type ChatSession struct {
	SessionID     string
	UserID        uint
	StartedAt     time.Time
	LastMessageAt time.Time
	MessageCount  int
}
