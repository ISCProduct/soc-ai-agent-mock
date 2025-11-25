package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"os"

	"Backend/internal/models"
)

func main() {
	// .envファイルを読み込み
	// 実行ディレクトリがBackend/cmd/migrateの場合は../../.envを、Backendの場合は.envを読み込む
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
		log.Println("No .env file found, using environment variables")
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

	log.Println("✓ Database migration completed successfully")

	// 初期データ投入(オプション)
	if os.Getenv("SEED_DATA") == "true" {
		log.Println("Seeding initial data...")
		if err := seedData(db); err != nil {
			log.Fatal("Failed to seed data:", err)
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
