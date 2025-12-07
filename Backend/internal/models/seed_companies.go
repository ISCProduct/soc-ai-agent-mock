package models

import (
	"gorm.io/gorm"
)

// SeedCompanies 企業データのシード
func SeedCompanies(db *gorm.DB) error {
	// 既に企業データが存在する場合はスキップ
	var count int64
	db.Model(&Company{}).Count(&count)
	if count > 0 {
		return nil
	}

	// サンプル企業データ
	companies := []Company{
		{
			Name:             "株式会社テックイノベーション",
			Description:      "最先端のAI技術で社会課題を解決するスタートアップ企業。自由な社風と成長機会が魅力。",
			Industry:         "IT・ソフトウェア",
			EmployeeCount:    150,
			FoundedYear:      2015,
			Location:         "東京都渋谷区",
			WebsiteURL:       "https://tech-innovation.example.com",
			Culture:          "フラットな組織で自由な発想を尊重。失敗を恐れずチャレンジできる環境。",
			WorkStyle:        "ハイブリッド（週3日出社）",
			TechStack:        `["Go", "React", "TypeScript", "AWS", "Kubernetes", "Docker"]`,
			DevelopmentStyle: "アジャイル",
			MainBusiness:     "AIソリューション開発、データ分析プラットフォーム",
			AverageAge:       32.5,
			FemaleRatio:      35.0,
			IsActive:         true,
			IsVerified:       true,
		},
		{
			Name:             "エンタープライズシステムズ株式会社",
			Description:      "大手企業向けの安定したシステム開発を行う老舗企業。充実した研修制度と福利厚生。",
			Industry:         "IT・ソフトウェア",
			EmployeeCount:    800,
			FoundedYear:      1995,
			Location:         "東京都千代田区",
			WebsiteURL:       "https://enterprise-systems.example.com",
			Culture:          "安定した環境で長期的なキャリア形成が可能。チームワークを重視。",
			WorkStyle:        "オフィス勤務",
			TechStack:        `["Java", "Spring", "Oracle", "AWS"]`,
			DevelopmentStyle: "ウォーターフォール",
			MainBusiness:     "大規模基幹システム開発、保守運用",
			AverageAge:       38.5,
			FemaleRatio:      25.0,
			IsActive:         true,
			IsVerified:       true,
		},
		{
			Name:             "クリエイティブラボ株式会社",
			Description:      "デザインと技術の融合で新しい価値を創造。クリエイティブな環境で働きたい方に最適。",
			Industry:         "Web制作・デザイン",
			EmployeeCount:    80,
			FoundedYear:      2018,
			Location:         "東京都港区",
			WebsiteURL:       "https://creative-lab.example.com",
			Culture:          "クリエイティビティを最大限に発揮できる自由な環境。デザイナーとエンジニアの協業を重視。",
			WorkStyle:        "フルリモート可",
			TechStack:        `["React", "Next.js", "TypeScript", "Figma", "Firebase"]`,
			DevelopmentStyle: "アジャイル",
			MainBusiness:     "Webサービス開発、UI/UXデザイン",
			AverageAge:       29.5,
			FemaleRatio:      45.0,
			IsActive:         true,
			IsVerified:       true,
		},
	}

	for _, company := range companies {
		if err := db.Create(&company).Error; err != nil {
			return err
		}
	}

	// 企業プロファイルをシード
	profiles := []CompanyWeightProfile{
		{
			CompanyID:             1, // テックイノベーション
			TechnicalOrientation:  85,
			TeamworkOrientation:   75,
			LeadershipOrientation: 60,
			CreativityOrientation: 80,
			StabilityOrientation:  40,
			GrowthOrientation:     90,
			WorkLifeBalance:       70,
			ChallengeSeeking:      85,
			DetailOrientation:     65,
			CommunicationSkill:    70,
		},
		{
			CompanyID:             2, // エンタープライズシステムズ
			TechnicalOrientation:  70,
			TeamworkOrientation:   85,
			LeadershipOrientation: 75,
			CreativityOrientation: 50,
			StabilityOrientation:  90,
			GrowthOrientation:     60,
			WorkLifeBalance:       80,
			ChallengeSeeking:      50,
			DetailOrientation:     85,
			CommunicationSkill:    80,
		},
		{
			CompanyID:             3, // クリエイティブラボ
			TechnicalOrientation:  75,
			TeamworkOrientation:   80,
			LeadershipOrientation: 55,
			CreativityOrientation: 95,
			StabilityOrientation:  45,
			GrowthOrientation:     80,
			WorkLifeBalance:       75,
			ChallengeSeeking:      80,
			DetailOrientation:     70,
			CommunicationSkill:    85,
		},
	}

	for _, profile := range profiles {
		if err := db.Create(&profile).Error; err != nil {
			return err
		}
	}

	return nil
}
