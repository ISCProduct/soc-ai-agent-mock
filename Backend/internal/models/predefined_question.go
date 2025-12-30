package models

import "time"

// PredefinedQuestion 事前定義された質問（ルールベース判定用）
type PredefinedQuestion struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	Category      string `gorm:"size:100;not null;index" json:"category"` // 評価カテゴリ
	QuestionText  string `gorm:"type:text;not null" json:"question_text"`
	TargetLevel   string `gorm:"size:20;default:'新卒'" json:"target_level"` // "新卒" | "中途" | "両方"
	IndustryID    *uint  `gorm:"index" json:"industry_id,omitempty"`
	JobCategoryID *uint  `gorm:"index" json:"job_category_id,omitempty"`
	Priority      int    `gorm:"default:10" json:"priority"` // 優先度（数値が大きいほど優先）
	IsActive      bool   `gorm:"default:true;index" json:"is_active"`

	// 判定用データ（JSON形式で保存）
	PositiveKeywords string `gorm:"type:json" json:"positive_keywords"` // ["プログラミング", "開発"...]
	NegativeKeywords string `gorm:"type:json" json:"negative_keywords"` // ["わからない", "苦手"...]
	ScoreRules       string `gorm:"type:json" json:"score_rules"`       // スコアリングルール

	// 追加質問設定
	FollowUpRules string `gorm:"type:json" json:"follow_up_rules"` // 追加質問のルール

	// フェーズ制御: JSON配列で保存（例: ["job_analysis","interest_analysis"]）。未設定は全フェーズ許可
	AllowedPhases string `gorm:"type:json" json:"allowed_phases"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ScoreRule スコアリングルール
type ScoreRule struct {
	Condition   string   `json:"condition"`    // "contains_any", "contains_all", "length", "regex"
	Keywords    []string `json:"keywords"`     // 判定キーワード
	ScoreChange int      `json:"score_change"` // スコア変動値
	Description string   `json:"description"`  // ルール説明
}

// FollowUpRule 追加質問ルール
type FollowUpRule struct {
	Trigger       string `json:"trigger"`        // "low_confidence", "high_score", "no_keywords"
	UseAI         bool   `json:"use_ai"`         // AIで質問生成するか
	AIPrompt      string `json:"ai_prompt"`      // AI用のプロンプト
	FixedQuestion string `json:"fixed_question"` // 固定の追加質問（AIを使わない場合）
	Purpose       string `json:"purpose"`        // 追加質問の目的
}

// TableName テーブル名を指定
func (PredefinedQuestion) TableName() string {
	return "predefined_questions"
}
