package models

import "gorm.io/gorm"

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
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
		&AnalysisPhase{},
		&UserAnalysisProgress{},
		&SessionValidation{},
		// 企業関連モデル
		&Company{},
		&CompanyJobPosition{},
		&CompanyWeightProfile{},
		&UserCompanyMatch{},
		&CompanyReview{},
		&CompanyBenefit{},
		&CompanyRelation{},
		&CompanyMarketInfo{},
	)
}
