package main

import (
	"Backend/internal/config"
	"Backend/internal/models"
	"log"
	"math/rand"
	"time"
)

func main() {
	// 乱数のシードを設定
	rand.Seed(time.Now().UnixNano())

	// 設定を読み込む（環境変数の読み込みはconfig.LoadConfig内で実施）
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// データベース接続
	db, err := config.ConnectDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Starting large data seed process...")
	startTime := time.Now()

	// マイグレーション実行
	if err := models.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database migration completed")

	// ステップ1: 大規模企業データのシード（40,000社）
	log.Println("[Step 1/4] Seeding 40,000 companies...")
	if err := models.SeedLargeCompanyData(db); err != nil {
		log.Fatalf("Failed to seed large company data: %v", err)
	}

	// ステップ2: 市場情報のシード
	log.Println("[Step 2/4] Seeding market information...")
	if err := models.SeedLargeCompanyMarketInfo(db); err != nil {
		log.Fatalf("Failed to seed market info: %v", err)
	}

	// ステップ3: 企業プロファイルのシード
	log.Println("[Step 3/4] Seeding company profiles...")
	if err := models.SeedLargeCompanyProfiles(db); err != nil {
		log.Fatalf("Failed to seed company profiles: %v", err)
	}

	// ステップ4: 企業関係（資本+ビジネス）のシード
	log.Println("[Step 4/4] Seeding company relations (capital & business)...")
	if err := models.SeedLargeCompanyRelations(db); err != nil {
		log.Fatalf("Failed to seed company relations: %v", err)
	}

	// 最終確認
	var companyCount, relationCount, profileCount int64
	db.Model(&models.Company{}).Count(&companyCount)
	db.Model(&models.CompanyRelation{}).Count(&relationCount)
	db.Model(&models.CompanyWeightProfile{}).Count(&profileCount)

	log.Printf("\n==============================================")
	log.Printf("Large data seed completed successfully!")
	log.Printf("Total time: %v", time.Since(startTime))
	log.Printf("==============================================")
	log.Printf("Final counts:")
	log.Printf("  - Companies: %d", companyCount)
	log.Printf("  - Relations: %d", relationCount)
	log.Printf("  - Profiles: %d", profileCount)
	log.Printf("==============================================\n")
}
