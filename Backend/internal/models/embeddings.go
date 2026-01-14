package models

import "time"

// UserEmbedding stores a user's profile embedding for analysis.
type UserEmbedding struct {
	ID          uint      `gorm:"primaryKey"`
	UserID      uint      `gorm:"not null;index:idx_user_session_embedding"`
	SessionID   string    `gorm:"size:100;not null;index:idx_user_session_embedding"`
	ProfileText string    `gorm:"type:text"` // Optional source text
	Embedding   string    `gorm:"type:json;not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// JobCategoryEmbedding stores an embedding for a job category.
type JobCategoryEmbedding struct {
	ID            uint      `gorm:"primaryKey"`
	JobCategoryID uint      `gorm:"not null;uniqueIndex"`
	SourceText    string    `gorm:"type:text"` // Optional source text
	Embedding     string    `gorm:"type:json;not null"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (UserEmbedding) TableName() string {
	return "user_embeddings"
}

func (JobCategoryEmbedding) TableName() string {
	return "job_category_embeddings"
}
