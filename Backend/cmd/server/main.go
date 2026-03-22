package main

import (
	"Backend/internal/config"
	"Backend/internal/controllers"
	"Backend/internal/models"
	"Backend/internal/openai"
	"Backend/internal/repositories"
	"Backend/internal/routes"
	"Backend/internal/scraper"
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

	// ── インフラ層: リポジトリ (domain/repository インターフェースを実装) ──────────
	// ユーザー・認証
	userRepo := repositories.NewUserRepository(db)
	pendingRegistrationRepo := repositories.NewPendingRegistrationRepository(db)
	// チャット・分析
	questionWeightRepo := repositories.NewQuestionWeightRepository(db)
	chatMessageRepo := repositories.NewChatMessageRepository(db)
	userWeightScoreRepo := repositories.NewUserWeightScoreRepository(db)
	aiGeneratedQuestionRepo := repositories.NewAIGeneratedQuestionRepository(db)
	predefinedQuestionRepo := repositories.NewPredefinedQuestionRepository(db)
	phaseRepo := repositories.NewAnalysisPhaseRepository(db)
	progressRepo := repositories.NewUserAnalysisProgressRepository(db)
	sessionValidationRepo := repositories.NewSessionValidationRepository(db)
	conversationContextRepo := repositories.NewConversationContextRepository(db)
	// 職種・企業
	jobCategoryRepo := repositories.NewJobCategoryRepository(db)
	companyRepo := repositories.NewCompanyRepository(db)
	crawlRepo := repositories.NewCrawlRepository(db)
	popularityRepo := repositories.NewCompanyPopularityRepository(db)
	graduateRepo := repositories.NewGraduateEmploymentRepository(db)
	companyRelationRepo := repositories.NewCompanyRelationRepository(db)
	companyQueryRepo := repositories.NewCompanyQueryRepository(db)
	matchRepo := repositories.NewUserCompanyMatchRepository(db)
	// 埋め込み・マッチング
	userEmbeddingRepo := repositories.NewUserEmbeddingRepository(db)
	jobEmbeddingRepo := repositories.NewJobCategoryEmbeddingRepository(db)
	// 面接
	interviewSessionRepo := repositories.NewInterviewSessionRepository(db)
	interviewUtteranceRepo := repositories.NewInterviewUtteranceRepository(db)
	interviewReportRepo := repositories.NewInterviewReportRepository(db)
	videoRepo := repositories.NewInterviewVideoRepository(db)
	// その他
	resumeRepo := repositories.NewResumeRepository(db)
	auditLogRepo := repositories.NewAuditLogRepository(db)
	// GitHub連携
	githubRepo := repositories.NewGitHubRepository(db)
	skillScoreRepo := repositories.NewSkillScoreRepository(db)

	// サービス層の初期化
	emailService := services.NewEmailService()
	authService := services.NewAuthService(userRepo, pendingRegistrationRepo, emailService)
	skillScoreService := services.NewSkillScoreService(skillScoreRepo)
	githubService := services.NewGitHubService(githubRepo, skillScoreService, aiClient)
	oauthService := services.NewOAuthService(userRepo, oauthConfig, githubService)
	chatService := services.NewChatService(aiClient, questionWeightRepo, chatMessageRepo, userWeightScoreRepo, aiGeneratedQuestionRepo, predefinedQuestionRepo, jobCategoryRepo, userRepo, userEmbeddingRepo, jobEmbeddingRepo, phaseRepo, progressRepo, sessionValidationRepo, conversationContextRepo)
	questionService := services.NewQuestionGeneratorService(aiClient, questionWeightRepo)
	matchingService := services.NewMatchingService(userWeightScoreRepo, companyRepo, matchRepo)
	resumeService := services.NewResumeService(resumeRepo, "storage/resumes", aiClient)
	crawlService := services.NewCrawlService(crawlRepo, companyRepo, popularityRepo, aiClient)
	auditLogService := services.NewAuditLogService(auditLogRepo)
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
	interviewService := services.NewInterviewService(interviewSessionRepo, interviewUtteranceRepo, interviewReportRepo, userRepo, emailService, aiClient)
	interviewService.StartWorker()

	// コントローラー層の初期化
	authController := controllers.NewAuthController(authService)
	oauthController := controllers.NewOAuthController(oauthService)
	chatController := controllers.NewChatController(chatService, matchingService, analysisService, userRepo, emailService)
	questionController := controllers.NewQuestionController(questionService)
	relationController := controllers.NewCompanyRelationController(companyQueryRepo, aiClient)
	adminCompanyController := controllers.NewAdminCompanyController(companyRepo, auditLogService, nil, aiClient)
	adminCrawlController := controllers.NewAdminCrawlController(crawlService, auditLogService)
	adminJobController := controllers.NewAdminJobController(companyRepo, jobCategoryRepo, graduateRepo, auditLogService)
	adminUserController := controllers.NewAdminUserController(userRepo, auditLogService)
	adminAuditController := controllers.NewAdminAuditController(auditLogService)
	companyGraphPipeline := &scraper.Pipeline{
		Mynavi:     scraper.NewMynaviScraper(""),
		Rikunabi:   scraper.NewRikunabiScraper(),
		CareerTasu: scraper.NewCareerTasuScraper(),
		Threshold:  0.75,
	}
	adminCompanyGraphController := controllers.NewAdminCompanyGraphController(companyGraphPipeline, companyRepo, companyRelationRepo, auditLogService)
	resumeController := controllers.NewResumeController(resumeService)

	// S3 upload service for interview videos (optional — skipped if env vars are not set)
	s3UploadService, s3Err := services.NewS3UploadService()
	if s3Err != nil {
		log.Printf("S3 upload service not available: %v", s3Err)
		s3UploadService = nil
	}
	interviewController := controllers.NewInterviewController(interviewService, videoRepo, s3UploadService)
	realtimeController := controllers.NewRealtimeController(interviewService)
	adminInterviewController := controllers.NewAdminInterviewController(interviewService, videoRepo, s3UploadService)
	companyEntryController := controllers.NewCompanyEntryController(companyRepo, graduateRepo)
	githubController := controllers.NewGitHubController(githubService, skillScoreService)

	// ルーティング設定
	routes.SetupAuthRoutes(authController, oauthController)
	routes.SetupChatRoutes(chatController, questionController)
	routes.SetupCompanyRoutes(relationController)
	routes.SetupAdminRoutes(adminCompanyController, adminCrawlController, adminJobController, adminUserController, adminAuditController, adminCompanyGraphController, adminInterviewController, userRepo)
	routes.SetupResumeRoutes(resumeController)
	routes.SetupInterviewRoutes(interviewController, realtimeController)
	routes.SetupGitHubRoutes(githubController)
	http.HandleFunc("/api/company-entry", companyEntryController.Submit)

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
