package entity

import "time"

// ResumeDocument 職務経歴書ドキュメントエンティティ
type ResumeDocument struct {
	ID               uint
	UserID           uint
	SessionID        string
	SourceType       string // pdf, docx, google_docs
	SourceURL        string
	OriginalFilename string
	StoredPath       string
	NormalizedPath   string
	AnnotatedPath    string
	Status           string // uploaded, processing, completed, failed
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// IsProcessed 処理済みかどうか
func (r *ResumeDocument) IsProcessed() bool {
	return r.Status == "completed"
}

// ResumeReview 職務経歴書レビュー結果エンティティ
type ResumeReview struct {
	ID         uint
	DocumentID uint
	Score      int
	Summary    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ResumeReviewItem 個別レビュー指摘事項エンティティ
type ResumeReviewItem struct {
	ID         uint
	ReviewID   uint
	PageNumber int
	BBox       string
	Severity   string // info, warning, critical
	Message    string
	Suggestion string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
