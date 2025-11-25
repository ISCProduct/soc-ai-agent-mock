package models

import "time"

// QuestionWeight 質問と重み係数を管理するテーブル
type QuestionWeight struct {
	ID             uint   `gorm:"primaryKey"`
	Question       string `gorm:"type:text;not null;uniqueIndex:idx_question_hash"`
	WeightCategory string `gorm:"size:100;not null;index"` // 例: "技術志向", "コミュニケーション", "リーダーシップ"
	WeightValue    int    `gorm:"not null"`                // 重み係数値
	IndustryID     uint   `gorm:"index"`                   // 関連する業界ID(オプション)
	JobCategoryID  uint   `gorm:"index"`                   // 関連する職種ID(オプション)
	Description    string `gorm:"type:text"`               // 質問の意図・説明
	IsActive       bool   `gorm:"default:true;index"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
