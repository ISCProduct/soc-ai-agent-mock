package models

import "time"

// AnalysisPhase 分析フェーズの定義
type AnalysisPhase struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	PhaseName    string    `gorm:"size:50;not null;uniqueIndex" json:"phase_name"` // 職種分析、興味分析、適性分析、将来分析
	DisplayName  string    `gorm:"size:100;not null" json:"display_name"`
	PhaseOrder   int       `gorm:"not null" json:"phase_order"` // 実行順序
	Description  string    `gorm:"type:text" json:"description"`
	MinQuestions int       `gorm:"not null;default:3" json:"min_questions"` // 最小質問数
	MaxQuestions int       `gorm:"not null;default:5" json:"max_questions"` // 最大質問数
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserAnalysisProgress ユーザーごとの分析進捗
type UserAnalysisProgress struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	UserID          uint           `gorm:"not null;index:idx_user_session_phase" json:"user_id"`
	SessionID       string         `gorm:"size:100;not null;index:idx_user_session_phase" json:"session_id"`
	PhaseID         uint           `gorm:"not null;index:idx_user_session_phase" json:"phase_id"`
	Phase           *AnalysisPhase `gorm:"foreignKey:PhaseID" json:"phase,omitempty"`
	QuestionsAsked  int            `gorm:"not null;default:0" json:"questions_asked"`
	ValidAnswers    int            `gorm:"not null;default:0" json:"valid_answers"`    // 有効な回答数
	InvalidAnswers  int            `gorm:"not null;default:0" json:"invalid_answers"`  // 無効な回答数
	CompletionScore float64        `gorm:"not null;default:0" json:"completion_score"` // 0-100のスコア
	IsCompleted     bool           `gorm:"not null;default:false" json:"is_completed"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// TableName overrides
func (AnalysisPhase) TableName() string {
	return "analysis_phases"
}

func (UserAnalysisProgress) TableName() string {
	return "user_analysis_progress"
}
