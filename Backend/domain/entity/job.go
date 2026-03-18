package entity

import "time"

// JobCategory 職種カテゴリエンティティ
type JobCategory struct {
	ID           uint
	ParentID     *uint
	Code         string
	Name         string
	NameEn       string
	Level        int
	Path         string
	Description  string
	DisplayOrder int
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsRoot ルートカテゴリかどうか
func (j *JobCategory) IsRoot() bool {
	return j.ParentID == nil
}
