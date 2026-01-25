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
	predefinedQuestionRepo := repositories.NewPredefinedQuestionRepository(db)
	jobCategoryRepo := repositories.NewJobCategoryRepository(db)
	phaseRepo := repositories.NewAnalysisPhaseRepository(db)
	progressRepo := repositories.NewUserAnalysisProgressRepository(db)
	sessionValidationRepo := repositories.NewSessionValidationRepository(db)
	conversationContextRepo := repositories.NewConversationContextRepository(db)
	companyRepo := repositories.NewCompanyRepository(db)
	gbizRepo := repositories.NewGBizInfoRepository(db)
	crawlRepo := repositories.NewCrawlRepository(db)
	popularityRepo := repositories.NewCompanyPopularityRepository(db)
	graduateRepo := repositories.NewGraduateEmploymentRepository(db)
	relationRepo := repositories.NewCompanyRelationRepository(db)
	auditLogRepo := repositories.NewAuditLogRepository(db)
	matchRepo := repositories.NewUserCompanyMatchRepository(db)
	resumeRepo := repositories.NewResumeRepository(db)
	userEmbeddingRepo := repositories.NewUserEmbeddingRepository(db)
	jobEmbeddingRepo := repositories.NewJobCategoryEmbeddingRepository(db)

	// サービス層の初期化
	authService := services.NewAuthService(userRepo)
	oauthService := services.NewOAuthService(userRepo, oauthConfig)
	chatService := services.NewChatService(aiClient, questionWeightRepo, chatMessageRepo, userWeightScoreRepo, aiGeneratedQuestionRepo, predefinedQuestionRepo, jobCategoryRepo, userRepo, userEmbeddingRepo, jobEmbeddingRepo, phaseRepo, progressRepo, sessionValidationRepo, conversationContextRepo)
	questionService := services.NewQuestionGeneratorService(aiClient, questionWeightRepo)
	matchingService := services.NewMatchingService(userWeightScoreRepo, companyRepo, matchRepo)
	resumeService := services.NewResumeService(resumeRepo, "storage/resumes", aiClient)
	crawlService := services.NewCrawlService(crawlRepo, companyRepo, popularityRepo, aiClient)
	auditLogService := services.NewAuditLogService(auditLogRepo)
	gbizService := services.NewGBizInfoService(cfg, gbizRepo, companyRepo, relationRepo)
	analysisService := services.NewAnalysisScoringService(
		userWeightScoreRepo,
		chatMessageRepo,
		progressRepo,
		conversationContextRepo,
		userEmbeddingRepo,
		jobEmbeddingRepo,
		matchRepo,
		nil,
	)

	// コントローラー層の初期化
	authController := controllers.NewAuthController(authService)
	oauthController := controllers.NewOAuthController(oauthService)
	chatController := controllers.NewChatController(chatService, matchingService, analysisService)
	questionController := controllers.NewQuestionController(questionService)
	relationController := &controllers.CompanyRelationController{DB: db}
	adminCompanyController := controllers.NewAdminCompanyController(companyRepo, auditLogService, gbizService)
	adminCrawlController := controllers.NewAdminCrawlController(crawlService, auditLogService)
	adminJobController := controllers.NewAdminJobController(companyRepo, jobCategoryRepo, graduateRepo, auditLogService)
	adminUserController := controllers.NewAdminUserController(userRepo, auditLogService)
	adminAuditController := controllers.NewAdminAuditController(auditLogService)
	resumeController := controllers.NewResumeController(resumeService)

	// ルーティング設定
	routes.SetupAuthRoutes(authController, oauthController)
	routes.SetupChatRoutes(chatController, questionController)
	routes.SetupCompanyRoutes(relationController)
	routes.SetupAdminRoutes(adminCompanyController, adminCrawlController, adminJobController, adminUserController, adminAuditController)
	routes.SetupResumeRoutes(resumeController)

	go crawlService.StartScheduler()

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
