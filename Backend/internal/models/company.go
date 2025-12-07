package models

import "time"

// Company 企業情報
type Company struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	Name          string `gorm:"type:varchar(255);not null" json:"name"`
	Description   string `gorm:"type:text" json:"description"`
	Industry      string `gorm:"type:varchar(100)" json:"industry"`
	EmployeeCount int    `gorm:"default:0" json:"employee_count"`
	FoundedYear   int    `json:"founded_year"`
	Location      string `gorm:"type:varchar(255)" json:"location"`
	WebsiteURL    string `gorm:"type:varchar(500)" json:"website_url"`
	LogoURL       string `gorm:"type:varchar(500)" json:"logo_url"`

	// 企業の特徴・文化
	Culture        string `gorm:"type:text" json:"culture"`            // 企業文化の説明
	WorkStyle      string `gorm:"type:varchar(100)" json:"work_style"` // リモート、ハイブリッド、オフィス
	WelfareDetails string `gorm:"type:text" json:"welfare_details"`    // 福利厚生の詳細

	// 技術情報
	TechStack        string `gorm:"type:text" json:"tech_stack"`                // 使用技術スタック（JSON形式）
	DevelopmentStyle string `gorm:"type:varchar(100)" json:"development_style"` // アジャイル、ウォーターフォールなど

	// ビジネス情報
	MainBusiness string  `gorm:"type:text" json:"main_business"` // 主要事業内容
	AverageAge   float64 `json:"average_age"`                    // 平均年齢
	FemaleRatio  float64 `json:"female_ratio"`                   // 女性比率（%）

	// 評価・ステータス
	IsActive   bool `gorm:"default:true" json:"is_active"`
	IsVerified bool `gorm:"default:false" json:"is_verified"` // 認証済み企業フラグ

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index"`
}

// CompanyJobPosition 企業の募集職種
type CompanyJobPosition struct {
	ID        uint    `gorm:"primaryKey"`
	CompanyID uint    `gorm:"not null;index"`
	Company   Company `gorm:"foreignKey:CompanyID"`

	Title         string      `gorm:"type:varchar(255);not null"`
	Description   string      `gorm:"type:text"`
	JobCategoryID uint        `gorm:"index"`
	JobCategory   JobCategory `gorm:"foreignKey:JobCategoryID"`

	// 給与情報
	MinSalary int // 最低年収（万円）
	MaxSalary int // 最高年収（万円）

	// 勤務条件
	EmploymentType string `gorm:"type:varchar(50)"` // 正社員、契約社員など
	WorkLocation   string `gorm:"type:varchar(255)"`
	RemoteOption   bool   `gorm:"default:false"`

	// 必須スキル・歓迎スキル
	RequiredSkills  string `gorm:"type:text"` // JSON形式
	PreferredSkills string `gorm:"type:text"` // JSON形式

	// 募集ステータス
	IsActive bool `gorm:"default:true"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

// CompanyWeightProfile 企業の適性プロファイル（10カテゴリの重視度）
type CompanyWeightProfile struct {
	ID            uint                `gorm:"primaryKey"`
	CompanyID     uint                `gorm:"not null;uniqueIndex:idx_company_job"`
	Company       Company             `gorm:"foreignKey:CompanyID"`
	JobPositionID *uint               `gorm:"uniqueIndex:idx_company_job"` // 職種別に設定可能
	JobPosition   *CompanyJobPosition `gorm:"foreignKey:JobPositionID"`

	// 10カテゴリの重視度（0-100のスコア）
	TechnicalOrientation  int `gorm:"default:50"` // 技術志向
	TeamworkOrientation   int `gorm:"default:50"` // チームワーク志向
	LeadershipOrientation int `gorm:"default:50"` // リーダーシップ志向
	CreativityOrientation int `gorm:"default:50"` // 創造性志向
	StabilityOrientation  int `gorm:"default:50"` // 安定志向
	GrowthOrientation     int `gorm:"default:50"` // 成長志向
	WorkLifeBalance       int `gorm:"default:50"` // ワークライフバランス
	ChallengeSeeking      int `gorm:"default:50"` // チャレンジ志向
	DetailOrientation     int `gorm:"default:50"` // 細部志向
	CommunicationSkill    int `gorm:"default:50"` // コミュニケーション力

	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserCompanyMatch ユーザーと企業のマッチング結果
type UserCompanyMatch struct {
	ID            uint                `gorm:"primaryKey"`
	UserID        uint                `gorm:"not null;index:idx_user_session"`
	User          User                `gorm:"foreignKey:UserID"`
	SessionID     string              `gorm:"type:varchar(255);index:idx_user_session"`
	CompanyID     uint                `gorm:"not null;index"`
	Company       Company             `gorm:"foreignKey:CompanyID"`
	JobPositionID *uint               `gorm:"index"`
	JobPosition   *CompanyJobPosition `gorm:"foreignKey:JobPositionID"`

	// マッチングスコア
	MatchScore         float64 `gorm:"not null"` // 総合マッチ度（0-100）
	TechnicalMatch     float64 // 技術志向マッチ度
	TeamworkMatch      float64 // チームワーク志向マッチ度
	LeadershipMatch    float64 // リーダーシップ志向マッチ度
	CreativityMatch    float64 // 創造性志向マッチ度
	StabilityMatch     float64 // 安定志向マッチ度
	GrowthMatch        float64 // 成長志向マッチ度
	WorkLifeMatch      float64 // ワークライフバランスマッチ度
	ChallengeMatch     float64 // チャレンジ志向マッチ度
	DetailMatch        float64 // 細部志向マッチ度
	CommunicationMatch float64 // コミュニケーション力マッチ度

	// マッチング理由・推薦文
	MatchReason string `gorm:"type:text"` // AIが生成したマッチング理由

	// ステータス
	IsViewed    bool `gorm:"default:false"` // ユーザーが閲覧したか
	IsFavorited bool `gorm:"default:false"` // お気に入り登録
	IsApplied   bool `gorm:"default:false"` // 応募済み

	CreatedAt time.Time
	UpdatedAt time.Time
}

// CompanyReview 企業レビュー（オプション機能）
type CompanyReview struct {
	ID        uint    `gorm:"primaryKey"`
	CompanyID uint    `gorm:"not null;index"`
	Company   Company `gorm:"foreignKey:CompanyID"`
	UserID    uint    `gorm:"not null;index"`
	User      User    `gorm:"foreignKey:UserID"`

	// レビュー内容
	Rating  int    `gorm:"not null"` // 1-5の評価
	Title   string `gorm:"type:varchar(255)"`
	Comment string `gorm:"type:text"`

	// レビューカテゴリ別評価
	WorkLifeRating     int // ワークライフバランス評価
	CultureRating      int // 企業文化評価
	GrowthRating       int // 成長機会評価
	CompensationRating int // 報酬評価

	// ステータス
	IsVerified bool `gorm:"default:false"` // 認証済みレビュー
	IsPublic   bool `gorm:"default:true"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

// CompanyBenefit 企業の福利厚生
type CompanyBenefit struct {
	ID        uint    `gorm:"primaryKey"`
	CompanyID uint    `gorm:"not null;index"`
	Company   Company `gorm:"foreignKey:CompanyID"`

	BenefitType string `gorm:"type:varchar(100);not null"` // 健康保険、退職金、リモートワークなど
	Description string `gorm:"type:text"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
