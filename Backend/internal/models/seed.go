package models

import "gorm.io/gorm"

// SeedData データベースに初期データを投入
func SeedData(db *gorm.DB) error {
	// 業界データ
	if err := seedIndustries(db); err != nil {
		return err
	}

	// 職種データ
	if err := seedJobCategories(db); err != nil {
		return err
	}

	// 業界-職種の関連付け
	if err := seedIndustryJobCategories(db); err != nil {
		return err
	}

	// 重み付けルール
	if err := seedWeightRules(db); err != nil {
		return err
	}

	// AI質問テンプレート
	if err := seedAIQuestionTemplates(db); err != nil {
		return err
	}

	return nil
}

func seedIndustries(db *gorm.DB) error {
	var count int64
	db.Model(&Industry{}).Count(&count)
	if count > 0 {
		return nil // 既にデータがある場合はスキップ
	}

	industries := []Industry{
		{Code: "IT", Name: "情報通信業", NameEn: "Information Technology", Level: 0, DisplayOrder: 1, IsActive: true},
		{Code: "IT-SW", Name: "ソフトウェア開発", NameEn: "Software Development", Level: 1, DisplayOrder: 1, IsActive: true},
		{Code: "IT-WEB", Name: "Web サービス", NameEn: "Web Services", Level: 1, DisplayOrder: 2, IsActive: true},
		{Code: "MFG", Name: "製造業", NameEn: "Manufacturing", Level: 0, DisplayOrder: 2, IsActive: true},
		{Code: "MFG-AUTO", Name: "自動車製造", NameEn: "Automotive", Level: 1, DisplayOrder: 1, IsActive: true},
		{Code: "MFG-ELEC", Name: "電子機器製造", NameEn: "Electronics", Level: 1, DisplayOrder: 2, IsActive: true},
		{Code: "FIN", Name: "金融・保険業", NameEn: "Finance & Insurance", Level: 0, DisplayOrder: 3, IsActive: true},
		{Code: "FIN-BANK", Name: "銀行", NameEn: "Banking", Level: 1, DisplayOrder: 1, IsActive: true},
		{Code: "FIN-INS", Name: "保険", NameEn: "Insurance", Level: 1, DisplayOrder: 2, IsActive: true},
		{Code: "CONS", Name: "コンサルティング", NameEn: "Consulting", Level: 0, DisplayOrder: 4, IsActive: true},
		{Code: "EDU", Name: "教育・学習支援業", NameEn: "Education", Level: 0, DisplayOrder: 5, IsActive: true},
		{Code: "MED", Name: "医療・福祉", NameEn: "Healthcare", Level: 0, DisplayOrder: 6, IsActive: true},
	}

	return db.Create(&industries).Error
}

func seedJobCategories(db *gorm.DB) error {
	var count int64
	db.Model(&JobCategory{}).Count(&count)
	if count > 0 {
		return nil
	}

	jobCategories := []JobCategory{
		{Code: "ENG", Name: "エンジニア", NameEn: "Engineer", Level: 0, DisplayOrder: 1, IsActive: true},
		{Code: "ENG-SW", Name: "ソフトウェアエンジニア", NameEn: "Software Engineer", Level: 1, DisplayOrder: 1, IsActive: true},
		{Code: "ENG-WEB", Name: "Webエンジニア", NameEn: "Web Engineer", Level: 1, DisplayOrder: 2, IsActive: true},
		{Code: "ENG-DATA", Name: "データエンジニア", NameEn: "Data Engineer", Level: 1, DisplayOrder: 3, IsActive: true},
		{Code: "SALES", Name: "営業", NameEn: "Sales", Level: 0, DisplayOrder: 2, IsActive: true},
		{Code: "SALES-B2B", Name: "法人営業", NameEn: "B2B Sales", Level: 1, DisplayOrder: 1, IsActive: true},
		{Code: "SALES-B2C", Name: "個人営業", NameEn: "B2C Sales", Level: 1, DisplayOrder: 2, IsActive: true},
		{Code: "MKT", Name: "マーケティング", NameEn: "Marketing", Level: 0, DisplayOrder: 3, IsActive: true},
		{Code: "MKT-DIG", Name: "デジタルマーケティング", NameEn: "Digital Marketing", Level: 1, DisplayOrder: 1, IsActive: true},
		{Code: "HR", Name: "人事", NameEn: "Human Resources", Level: 0, DisplayOrder: 4, IsActive: true},
		{Code: "FIN-ACC", Name: "財務・経理", NameEn: "Finance & Accounting", Level: 0, DisplayOrder: 5, IsActive: true},
		{Code: "CONS-BIZ", Name: "経営コンサルタント", NameEn: "Business Consultant", Level: 0, DisplayOrder: 6, IsActive: true},
	}

	return db.Create(&jobCategories).Error
}

func seedWeightRules(db *gorm.DB) error {
	var count int64
	db.Model(&WeightRule{}).Count(&count)
	if count > 0 {
		return nil
	}

	weightRules := []WeightRule{
		{
			Name:        "技術関連キーワード強化",
			Condition:   `{"keywords": ["プログラミング", "コード", "技術", "開発", "エンジニア"]}`,
			WeightBoost: 5,
			Description: "技術関連のキーワードが含まれる場合、技術志向のスコアを強化",
			Priority:    10,
			IsActive:    true,
		},
		{
			Name:        "チーム関連キーワード強化",
			Condition:   `{"keywords": ["チーム", "協力", "協働", "メンバー", "リーダー"]}`,
			WeightBoost: 3,
			Description: "チームワーク関連のキーワードが含まれる場合、チームワークのスコアを強化",
			Priority:    8,
			IsActive:    true,
		},
		{
			Name:        "コミュニケーション関連キーワード強化",
			Condition:   `{"keywords": ["コミュニケーション", "説明", "伝える", "話す", "対話"]}`,
			WeightBoost: 4,
			Description: "コミュニケーション関連のキーワードが含まれる場合、スコアを強化",
			Priority:    9,
			IsActive:    true,
		},
		{
			Name:        "分析思考キーワード強化",
			Condition:   `{"keywords": ["分析", "データ", "論理", "問題解決", "思考"]}`,
			WeightBoost: 4,
			Description: "分析思考関連のキーワードが含まれる場合、スコアを強化",
			Priority:    7,
			IsActive:    true,
		},
		{
			Name:        "創造性キーワード強化",
			Condition:   `{"keywords": ["創造", "アイデア", "革新", "新しい", "デザイン"]}`,
			WeightBoost: 3,
			Description: "創造性関連のキーワードが含まれる場合、スコアを強化",
			Priority:    6,
			IsActive:    true,
		},
		{
			Name:        "リーダーシップキーワード強化",
			Condition:   `{"keywords": ["リーダー", "リード", "指導", "マネジメント", "統率"]}`,
			WeightBoost: 5,
			Description: "リーダーシップ関連のキーワードが含まれる場合、スコアを強化",
			Priority:    8,
			IsActive:    true,
		},
	}

	return db.Create(&weightRules).Error
}

func seedAIQuestionTemplates(db *gorm.DB) error {
	var count int64
	db.Model(&AIQuestionTemplate{}).Count(&count)
	if count > 0 {
		return nil
	}

	templates := []AIQuestionTemplate{
		{
			Category:    "技術志向",
			Prompt:      "あなたの技術的な興味や経験について教えてください。特に最近取り組んでいることや学んでいることはありますか？",
			BaseWeight:  8,
			ContextKeys: `["技術", "開発", "プログラミング"]`,
			IsActive:    true,
		},
		{
			Category:    "コミュニケーション",
			Prompt:      "チームでの作業において、あなたはどのようにコミュニケーションを取りますか？具体的なエピソードがあれば教えてください。",
			BaseWeight:  7,
			ContextKeys: `["チーム", "コミュニケーション", "協力"]`,
			IsActive:    true,
		},
		{
			Category:    "チームワーク",
			Prompt:      "チームプロジェクトでの経験について教えてください。あなたはどのような役割を担当しましたか？",
			BaseWeight:  7,
			ContextKeys: `["チーム", "プロジェクト", "役割"]`,
			IsActive:    true,
		},
		{
			Category:    "リーダーシップ",
			Prompt:      "グループやチームを率いた経験はありますか？その時どのようなアプローチを取りましたか？",
			BaseWeight:  8,
			ContextKeys: `["リーダー", "リード", "指導"]`,
			IsActive:    true,
		},
		{
			Category:    "分析思考",
			Prompt:      "複雑な問題に直面したとき、あなたはどのように分析し解決しますか？具体例を教えてください。",
			BaseWeight:  8,
			ContextKeys: `["分析", "問題解決", "論理"]`,
			IsActive:    true,
		},
		{
			Category:    "創造性",
			Prompt:      "今までで最も創造的だと思うアイデアや解決策は何ですか？それはどのような場面で生まれましたか？",
			BaseWeight:  7,
			ContextKeys: `["創造", "アイデア", "革新"]`,
			IsActive:    true,
		},
	}

	return db.Create(&templates).Error
}

func seedIndustryJobCategories(db *gorm.DB) error {
	var count int64
	db.Model(&IndustryJobCategory{}).Count(&count)
	if count > 0 {
		return nil
	}

	// IT業界とエンジニア職の関連付け
	relations := []IndustryJobCategory{
		// IT業界
		{IndustryID: 1, JobCategoryID: 1, IsCommon: true, DisplayOrder: 1},  // IT - エンジニア
		{IndustryID: 1, JobCategoryID: 2, IsCommon: true, DisplayOrder: 2},  // IT - ソフトウェアエンジニア
		{IndustryID: 1, JobCategoryID: 3, IsCommon: true, DisplayOrder: 3},  // IT - Webエンジニア
		{IndustryID: 1, JobCategoryID: 4, IsCommon: true, DisplayOrder: 4},  // IT - データエンジニア
		{IndustryID: 1, JobCategoryID: 5, IsCommon: false, DisplayOrder: 5}, // IT - 営業
		{IndustryID: 1, JobCategoryID: 8, IsCommon: false, DisplayOrder: 6}, // IT - マーケティング

		// 製造業
		{IndustryID: 4, JobCategoryID: 1, IsCommon: true, DisplayOrder: 1},   // 製造業 - エンジニア
		{IndustryID: 4, JobCategoryID: 5, IsCommon: true, DisplayOrder: 2},   // 製造業 - 営業
		{IndustryID: 4, JobCategoryID: 10, IsCommon: false, DisplayOrder: 3}, // 製造業 - 人事

		// 金融・保険業
		{IndustryID: 7, JobCategoryID: 5, IsCommon: true, DisplayOrder: 1},   // 金融 - 営業
		{IndustryID: 7, JobCategoryID: 11, IsCommon: true, DisplayOrder: 2},  // 金融 - 財務・経理
		{IndustryID: 7, JobCategoryID: 12, IsCommon: false, DisplayOrder: 3}, // 金融 - コンサルタント

		// コンサルティング
		{IndustryID: 10, JobCategoryID: 12, IsCommon: true, DisplayOrder: 1}, // コンサルティング - コンサルタント
		{IndustryID: 10, JobCategoryID: 1, IsCommon: false, DisplayOrder: 2}, // コンサルティング - エンジニア
	}

	return db.Create(&relations).Error
}
