package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"Backend/internal/models"
)

func main() {
	env := os.Getenv("APP_ENV")

	if env != "production" {
		// ローカル開発環境では .env ファイルを読み込む
		envPaths := []string{
			".env",
			"../.env",
			"../../.env",
		}

		envLoaded := false
		for _, envPath := range envPaths {
			if err := godotenv.Load(envPath); err == nil {
				log.Printf("Loaded .env file from: %s", envPath)
				envLoaded = true
				break
			}
		}

		if !envLoaded {
			log.Println("Warning: .env file not found. Skipping.")
		}
	}

	// データベース接続
	db, err := connectDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// マイグレーション実行
	log.Println("Starting database migration...")

	if err := models.AutoMigrate(db); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Ensure allowed_phases column exists for predefined_questions (backfill for older DBs)
	if err := ensureAllowedPhasesColumn(db); err != nil {
		log.Fatalf("Failed to ensure allowed_phases column: %v", err)
	}

	log.Println("✓ Database migration completed successfully")

	// 初期データ投入(オプション)
	if os.Getenv("SEED_DATA") == "true" {
		log.Println("Seeding initial data...")
		if err := seedData(db); err != nil {
			log.Fatal("Failed to seed data:", err)
		}
		// 追加: SeedData も呼び出す（その中で SeedPredefinedQuestions も呼ばれる）
		if err := models.SeedData(db); err != nil {
			log.Fatal("Failed to seed full data:", err)
		}
		log.Println("✓ Data seeding completed successfully")
	}
}

func connectDB() (*gorm.DB, error) {
	// デフォルト値を設定
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "3306"
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		log.Fatal("DB_USER is required")
	}

	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		log.Fatal("DB_NAME is required")
	}

	log.Printf("Connecting to database: %s@tcp(%s:%s)/%s", user, host, port, dbname)

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user,
		password,
		host,
		port,
		dbname,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Println("✓ Database connected successfully")
	return db, nil
}

func seedData(db *gorm.DB) error {
	// AIテンプレートの初期データ
	templates := []models.AIQuestionTemplate{
		{
			Category:    "basic",
			Prompt:      "ユーザーの基本的な興味や価値観を探る質問を生成してください",
			BaseWeight:  8,
			ContextKeys: `["industry_ids", "job_category_ids"]`,
			IsActive:    true,
		},
		{
			Category:    "skill",
			Prompt:      "ユーザーの具体的なスキルや経験を深掘りする質問を生成してください",
			BaseWeight:  6,
			ContextKeys: `["answer_history"]`,
			IsActive:    true,
		},
		{
			Category:    "preference",
			Prompt:      "ユーザーの働き方や環境の希望を確認する質問を生成してください",
			BaseWeight:  5,
			ContextKeys: `["industry_ids", "job_category_ids", "answer_history"]`,
			IsActive:    true,
		},
	}

	for _, template := range templates {
		if err := db.FirstOrCreate(&template, models.AIQuestionTemplate{Category: template.Category}).Error; err != nil {
			return err
		}
	}

	// 重みルールの初期データ
	rules := []models.WeightRule{
		{
			Name:        "深掘りフェーズでの重み増加",
			Condition:   `{"phase": "deep"}`,
			WeightBoost: 2,
			Priority:    10,
			IsActive:    true,
		},
		{
			Name:        "初回の業界質問",
			Condition:   `{"industry_count": 0}`,
			WeightBoost: 3,
			Priority:    20,
			IsActive:    true,
		},
	}

	for _, rule := range rules {
		if err := db.FirstOrCreate(&rule, models.WeightRule{Name: rule.Name}).Error; err != nil {
			return err
		}
	}

	return nil
}

// ensureAllowedPhasesColumn checks if the predefined_questions.allowed_phases column exists; if not, it adds it.
func ensureAllowedPhasesColumn(db *gorm.DB) error {
	var count int64
	// Query information_schema for the column
	err := db.Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'predefined_questions' AND COLUMN_NAME = 'allowed_phases'").Scan(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		log.Println("allowed_phases column not found, adding column...")
		// Add JSON column (nullable)
		if err := db.Exec("ALTER TABLE predefined_questions ADD COLUMN allowed_phases JSON NULL").Error; err != nil {
			return err
		}
		log.Println("added allowed_phases column to predefined_questions")
	} else {
		log.Println("allowed_phases column already exists, skipping")
	}
	// Backfill default allowed_phases for existing rows where null or empty
	if err := backfillAllowedPhases(db); err != nil {
		return err
	}
	return nil
}

// backfillAllowedPhases sets a sensible default for existing rows where allowed_phases is NULL or empty
func backfillAllowedPhases(db *gorm.DB) error {
	// default phases: all four phases
	defaultJSON := "[\"job_analysis\",\"interest_analysis\",\"aptitude_analysis\",\"future_analysis\"]"

	var before int64
	err := db.Raw("SELECT COUNT(*) FROM predefined_questions WHERE allowed_phases IS NULL OR allowed_phases = ''").Scan(&before).Error
	if err != nil {
		return err
	}
	log.Printf("predefined_questions rows needing backfill: %d", before)

	if before > 0 {
		res := db.Exec("UPDATE predefined_questions SET allowed_phases = ? WHERE allowed_phases IS NULL OR allowed_phases = ''", defaultJSON)
		if res.Error != nil {
			return res.Error
		}
		log.Printf("backfilled allowed_phases for %d rows", res.RowsAffected)
	} else {
		log.Println("no backfill needed for predefined_questions")
	}

	var after int64
	err = db.Raw("SELECT COUNT(*) FROM predefined_questions WHERE allowed_phases IS NULL OR allowed_phases = ''").Scan(&after).Error
	if err != nil {
		return err
	}
	log.Printf("remaining rows with empty allowed_phases: %d", after)
	return nil
}
