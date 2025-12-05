package main

import (
	"Backend/internal/config"
	"Backend/internal/controllers"
	"Backend/internal/models"
	"Backend/internal/openai"
	"Backend/internal/repositories"
	"Backend/internal/routes"
	"Backend/internal/services"
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// .env ファイルを読み込む
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// 設定を読み込む
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// データベース接続
	db, err := config.ConnectDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// マイグレーション実行
	if err := models.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database migration completed")

	// 初期データ投入
	if err := models.SeedData(db); err != nil {
		log.Fatalf("Failed to seed data: %v", err)
	}
	log.Println("Database seeding completed")

	// OpenAI クライアント初期化
	aiClient, err := openai.NewFromEnv("")
	if err != nil {
		log.Fatalf("Failed to initialize OpenAI client: %v", err)
	}

	// OAuth設定読み込み
	oauthConfig := config.LoadOAuthConfig()

	// リポジトリ層の初期化
	userRepo := repositories.NewUserRepository(db)
	questionWeightRepo := repositories.NewQuestionWeightRepository(db)
	chatMessageRepo := repositories.NewChatMessageRepository(db)
	userWeightScoreRepo := repositories.NewUserWeightScoreRepository(db)
	aiGeneratedQuestionRepo := repositories.NewAIGeneratedQuestionRepository(db)
	phaseRepo := repositories.NewAnalysisPhaseRepository(db)
	progressRepo := repositories.NewUserAnalysisProgressRepository(db)
	sessionValidationRepo := repositories.NewSessionValidationRepository(db)

	// サービス層の初期化
	authService := services.NewAuthService(userRepo)
	oauthService := services.NewOAuthService(userRepo, oauthConfig)
	chatService := services.NewChatService(aiClient, questionWeightRepo, chatMessageRepo, userWeightScoreRepo, aiGeneratedQuestionRepo, userRepo, phaseRepo, progressRepo, sessionValidationRepo)
	questionService := services.NewQuestionGeneratorService(aiClient, questionWeightRepo)

	// コントローラー層の初期化
	authController := controllers.NewAuthController(authService)
	oauthController := controllers.NewOAuthController(oauthService)
	chatController := controllers.NewChatController(chatService)
	questionController := controllers.NewQuestionController(questionService)

	// ルーティング設定
	routes.SetupAuthRoutes(authController, oauthController)
	routes.SetupChatRoutes(chatController, questionController)

	// ヘルスチェックエンドポイント
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// サーバー起動
	port := cfg.ServerPort
	if port == "" {
		port = "80"
	}

	log.Printf("Starting server on port %s...", port)
	if err := http.ListenAndServe(":"+port, corsMiddleware(http.DefaultServeMux)); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
