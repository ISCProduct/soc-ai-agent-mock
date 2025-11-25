package models

import "gorm.io/gorm"

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Industry{},
		&JobCategory{},
		&IndustryJobCategory{},
		&AIQuestionTemplate{},
		&ConversationContext{},
		&AIGeneratedQuestion{},
		&WeightRule{},
		&QuestionWeight{},
		&ChatMessage{},
		&UserWeightScore{},
	)
}
