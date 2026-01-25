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
		&PredefinedQuestion{}, // 事前定義質問（ルールベース判定用）
		&ChatMessage{},
		&UserWeightScore{},
		&AnalysisPhase{},
		&UserAnalysisProgress{},
		&SessionValidation{},
		&UserEmbedding{},
		&JobCategoryEmbedding{},
		// 企業関連モデル
		&Company{},
		&CompanyJobPosition{},
		&CompanyWeightProfile{},
		&UserCompanyMatch{},
		&CompanyReview{},
		&CompanyBenefit{},
		&GBizCompanyProfile{},
		&GBizProcurement{},
		&GBizSubsidy{},
		&GBizFinance{},
		&GBizWorkplace{},
		&CompanyRelation{},
		&CompanyMarketInfo{},
		&CompanyPopularityRecord{},
		&GraduateEmployment{},
		&ResumeDocument{},
		&ResumeTextBlock{},
		&ResumeReview{},
		&ResumeReviewItem{},
		&CrawlSource{},
		&CrawlRun{},
		&AuditLog{},
	)
}
