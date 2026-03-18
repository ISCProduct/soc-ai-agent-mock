package repository

import "Backend/internal/models"

// UserEmbeddingRepository はユーザーベクトル埋め込みの永続化インターフェース。
type UserEmbeddingRepository interface {
	FindByUserAndSession(userID uint, sessionID string) (*models.UserEmbedding, error)
	Upsert(userID uint, sessionID, profileText, embedding string) error
}

// JobCategoryEmbeddingRepository は職種カテゴリのベクトル埋め込みの永続化インターフェース。
type JobCategoryEmbeddingRepository interface {
	FindByJobCategoryID(jobCategoryID uint) (*models.JobCategoryEmbedding, error)
	Upsert(jobCategoryID uint, sourceText, embedding string) error
}
