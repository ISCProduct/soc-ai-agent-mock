package services

import (
	"Backend/internal/models"
	"Backend/internal/openai"
	"Backend/internal/repositories"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type ChatService struct {
	aiClient                *openai.Client
	questionWeightRepo      *repositories.QuestionWeightRepository
	chatMessageRepo         *repositories.ChatMessageRepository
	userWeightScoreRepo     *repositories.UserWeightScoreRepository
	aiGeneratedQuestionRepo *repositories.AIGeneratedQuestionRepository
	predefinedQuestionRepo  *repositories.PredefinedQuestionRepository
	jobCategoryRepo         *repositories.JobCategoryRepository
	userRepo                *repositories.UserRepository
	phaseRepo               *repositories.AnalysisPhaseRepository
	progressRepo            *repositories.UserAnalysisProgressRepository
	sessionValidationRepo   *repositories.SessionValidationRepository
	conversationContextRepo *repositories.ConversationContextRepository
	answerEvaluator         *AnswerEvaluator
	jobValidator            *JobCategoryValidator
}

func NewChatService(
	aiClient *openai.Client,
	questionWeightRepo *repositories.QuestionWeightRepository,
	chatMessageRepo *repositories.ChatMessageRepository,
	userWeightScoreRepo *repositories.UserWeightScoreRepository,
	aiGeneratedQuestionRepo *repositories.AIGeneratedQuestionRepository,
	predefinedQuestionRepo *repositories.PredefinedQuestionRepository,
	jobCategoryRepo *repositories.JobCategoryRepository,
	userRepo *repositories.UserRepository,
	phaseRepo *repositories.AnalysisPhaseRepository,
	progressRepo *repositories.UserAnalysisProgressRepository,
	sessionValidationRepo *repositories.SessionValidationRepository,
	conversationContextRepo *repositories.ConversationContextRepository,
) *ChatService {
	return &ChatService{
		aiClient:                aiClient,
		questionWeightRepo:      questionWeightRepo,
		chatMessageRepo:         chatMessageRepo,
		userWeightScoreRepo:     userWeightScoreRepo,
		aiGeneratedQuestionRepo: aiGeneratedQuestionRepo,
		predefinedQuestionRepo:  predefinedQuestionRepo,
		jobCategoryRepo:         jobCategoryRepo,
		userRepo:                userRepo,
		phaseRepo:               phaseRepo,
		progressRepo:            progressRepo,
		sessionValidationRepo:   sessionValidationRepo,
		conversationContextRepo: conversationContextRepo,
		answerEvaluator:         NewAnswerEvaluator(),
		jobValidator:            NewJobCategoryValidator(aiClient, jobCategoryRepo),
	}
}

// ChatRequest チャットリクエスト
type ChatRequest struct {
	UserID        uint   `json:"user_id"`
	SessionID     string `json:"session_id"`
	Message       string `json:"message"`
	IndustryID    uint   `json:"industry_id"`
	JobCategoryID uint   `json:"job_category_id"`
}

// ChatResponse チャットレスポンス
type ChatResponse struct {
	Response            string                   `json:"response"`
	QuestionWeightID    uint                     `json:"question_weight_id,omitempty"`
	CurrentScores       []models.UserWeightScore `json:"current_scores,omitempty"`
	CurrentPhase        *PhaseProgress           `json:"current_phase,omitempty"`
	AllPhases           []PhaseProgress          `json:"all_phases,omitempty"`
	IsComplete          bool                     `json:"is_complete"`
	IsTerminated        bool                     `json:"is_terminated,omitempty"`
	InvalidAnswerCount  int                      `json:"invalid_answer_count,omitempty"`
	TotalQuestions      int                      `json:"total_questions"`
	AnsweredQuestions   int                      `json:"answered_questions"`
	EvaluatedCategories int                      `json:"evaluated_categories"`
	TotalCategories     int                      `json:"total_categories"`
}

// PhaseProgress フェーズ進捗情報
type PhaseProgress struct {
	PhaseID         uint    `json:"phase_id"`
	PhaseName       string  `json:"phase_name"`
	DisplayName     string  `json:"display_name"`
	QuestionsAsked  int     `json:"questions_asked"`
	ValidAnswers    int     `json:"valid_answers"`
	CompletionScore float64 `json:"completion_score"`
	IsCompleted     bool    `json:"is_completed"`
	MinQuestions    int     `json:"min_questions"`
	MaxQuestions    int     `json:"max_questions"`
}

// ProcessChat チャット処理のメインロジック
func (s *ChatService) ProcessChat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// セッション開始の特殊処理
	if req.Message == "START_SESSION" {
		return s.handleSessionStart(ctx, req)
	}

	// セッション終了チェック
	isTerminated, err := s.sessionValidationRepo.IsTerminated(req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to check session status: %w", err)
	}
	if isTerminated {
		terminationMsg := "このセッションは終了しています。不適切な回答が3回続いたため、チャットを終了しました。新しいセッションを開始してください。"
		assistantMsg := &models.ChatMessage{
			SessionID: req.SessionID,
			UserID:    req.UserID,
			Role:      "assistant",
			Content:   terminationMsg,
		}
		if err := s.chatMessageRepo.Create(assistantMsg); err != nil {
			fmt.Printf("Warning: failed to save termination message: %v\n", err)
		}
		return &ChatResponse{
			Response:     terminationMsg,
			IsComplete:   true,
			IsTerminated: true,
		}, nil
	}

	// 1. ユーザーのメッセージを保存
	userMsg := &models.ChatMessage{
		SessionID: req.SessionID,
		UserID:    req.UserID,
		Role:      "user",
		Content:   req.Message,
	}
	if err := s.chatMessageRepo.Create(userMsg); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// 2. 会話履歴を取得（ユーザーメッセージ保存後に取得）
	history, err := s.chatMessageRepo.FindRecentBySessionID(req.SessionID, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat history: %w", err)
	}

	// 2-1. 職種の解決（未設定なら判定し、セッションに保存）
	jobCategoryID := req.JobCategoryID
	storedJobCategoryID := uint(0)
	if s.conversationContextRepo != nil {
		if id, err := s.conversationContextRepo.GetJobCategoryID(req.SessionID); err == nil {
			storedJobCategoryID = id
		}
	}
	if jobCategoryID == 0 {
		jobCategoryID = storedJobCategoryID
	}
	if jobCategoryID != 0 && s.conversationContextRepo != nil && storedJobCategoryID != jobCategoryID {
		if err := s.conversationContextRepo.SetJobCategoryID(req.UserID, req.SessionID, jobCategoryID); err != nil {
			fmt.Printf("Warning: failed to store job category: %v\n", err)
		}
	}

	jobJustResolved := false
	if jobCategoryID == 0 && s.shouldValidateJobCategory(history) {
		fmt.Printf("[JobValidation] Validating job category answer: %s\n", req.Message)
		jobValidation, err := s.jobValidator.ValidateJobCategory(ctx, req.Message)
		if err != nil {
			fmt.Printf("[JobValidation] Error: %v\n", err)
			// エラーでも続行
		} else if jobValidation != nil {
			if jobValidation.IsValid && len(jobValidation.MatchedCategories) > 0 {
				// 明確に職種が特定できた場合
				fmt.Printf("[JobValidation] Valid job category matched: %d categories\n", len(jobValidation.MatchedCategories))
				jobCategoryID = jobValidation.MatchedCategories[0].ID
				jobJustResolved = true
				if s.conversationContextRepo != nil {
					if err := s.conversationContextRepo.SetJobCategoryID(req.UserID, req.SessionID, jobCategoryID); err != nil {
						fmt.Printf("Warning: failed to store job category: %v\n", err)
					}
				}
			} else if jobValidation.NeedsClarification && jobValidation.SuggestedQuestion != "" {
				// 職種が曖昧な場合は選択肢を提示
				fmt.Printf("[JobValidation] Needs clarification, presenting options\n")

				assistantMsg := &models.ChatMessage{
					SessionID: req.SessionID,
					UserID:    req.UserID,
					Role:      "assistant",
					Content:   jobValidation.SuggestedQuestion,
				}
				if err := s.chatMessageRepo.Create(assistantMsg); err != nil {
					fmt.Printf("Warning: failed to save assistant message: %v\n", err)
				}

				return &ChatResponse{
					Response:          jobValidation.SuggestedQuestion,
					IsComplete:        false,
					TotalQuestions:    15,
					AnsweredQuestions: 0,
				}, nil
			}
		}
	}

	// 2.5. 回答の妥当性チェック（保存後のhistoryを使用）
	handled, response, err := s.checkAnswerValidity(ctx, history, req.Message, req.UserID, req.SessionID)
	if err != nil {
		return nil, err
	}

	// 無効な回答の場合は、ここで処理を終了
	if handled {
		validation, err := s.sessionValidationRepo.GetOrCreate(req.SessionID)
		if err != nil {
			fmt.Printf("Warning: failed to get validation: %v\n", err)
		}

		allPhases, currentPhaseInfo, _ := s.buildPhaseProgressResponse(req.UserID, req.SessionID)

		chatResponse := &ChatResponse{
			Response:          response,
			IsComplete:        false,
			TotalQuestions:    15,
			AnsweredQuestions: len(history) / 2,
			AllPhases:         allPhases,
			CurrentPhase:      currentPhaseInfo,
		}

		if validation != nil {
			chatResponse.InvalidAnswerCount = validation.InvalidAnswerCount
			chatResponse.IsTerminated = validation.IsTerminated

			// 3回目の無効回答の場合は完了フラグを立てる
			if validation.IsTerminated {
				chatResponse.IsComplete = true
			}
		}

		return chatResponse, nil
	}

	// 有効な回答の場合のみ、以降の処理を実行
	// 2.6. 現在のフェーズを取得または開始
	currentPhase, err := s.getCurrentOrNextPhase(ctx, req.UserID, req.SessionID)
	if err != nil {
		// 全フェーズ完了の場合は完了応答を返す
		if err.Error() == "all phases completed" {
			completionMsg := "分析が完了しました！あなたに最適な企業をマッチングしました。「結果を見る」ボタンから詳細をご確認ください。"

			assistantMsg := &models.ChatMessage{
				SessionID: req.SessionID,
				UserID:    req.UserID,
				Role:      "assistant",
				Content:   completionMsg,
			}
			if err := s.chatMessageRepo.Create(assistantMsg); err != nil {
				fmt.Printf("Warning: failed to save completion message: %v\n", err)
			}

			allPhases, currentPhaseInfo, _ := s.buildPhaseProgressResponse(req.UserID, req.SessionID)
			return &ChatResponse{
				Response:            completionMsg,
				IsComplete:          true,
				TotalQuestions:      15,
				AnsweredQuestions:   countUserAnswers(history),
				EvaluatedCategories: 10,
				TotalCategories:     10,
				AllPhases:           allPhases,
				CurrentPhase:        currentPhaseInfo,
			}, nil
		}
		return nil, fmt.Errorf("failed to get current phase: %w", err)
	}

	// 2.7. フェーズ進捗を更新（有効な回答のみ）
	if err := s.updatePhaseProgress(currentPhase, true); err != nil {
		fmt.Printf("Warning: failed to update phase progress: %v\n", err)
	}

	// 3. ユーザーの回答から重み係数を判定・更新
	// 選択肢の回答かどうかをチェック
	trimmedAnswer := strings.TrimSpace(req.Message)
	fmt.Printf("[ProcessChat] Checking if choice answer: '%s' (len=%d)\n", trimmedAnswer, len(trimmedAnswer))
	if len(trimmedAnswer) <= 3 && s.isChoiceAnswer(trimmedAnswer) {
		fmt.Printf("[ProcessChat] Processing as choice answer\n")
		// 選択肢回答の場合は直接スコアを計算
		if err := s.processChoiceAnswer(ctx, req.UserID, req.SessionID, trimmedAnswer, history, jobCategoryID); err != nil {
			fmt.Printf("Warning: failed to process choice answer: %v\n", err)
		}
	} else {
		fmt.Printf("[ProcessChat] Processing as text answer\n")
		// 通常の回答分析
		if err := s.analyzeAndUpdateWeights(ctx, req.UserID, req.SessionID, req.Message, jobCategoryID); err != nil {
			// ログに記録するが、処理は継続
			fmt.Printf("Warning: failed to update weights: %v\n", err)
		}
	}

	// 4. 既に聞いた質問を全て収集（重複防止を徹底）
	askedTexts := make(map[string]bool)

	// 4-1. AI生成質問テーブルから取得
	askedQuestions, err := s.aiGeneratedQuestionRepo.FindByUserAndSession(req.UserID, req.SessionID)
	if err != nil {
		fmt.Printf("Warning: failed to get asked questions: %v\n", err)
		askedQuestions = []models.AIGeneratedQuestion{}
	}
	for _, q := range askedQuestions {
		questionText := normalizeQuestionText(q.QuestionText)
		if questionText == "" {
			questionText = strings.TrimSpace(q.QuestionText)
		}
		if questionText != "" {
			askedTexts[questionText] = true
		}
	}

	// 4-2. チャット履歴からもアシスタントの質問を収集
	for _, msg := range history {
		if msg.Role == "assistant" {
			questionText := normalizeQuestionText(msg.Content)
			if questionText != "" {
				askedTexts[questionText] = true
			}
		}
	}

	fmt.Printf("Total asked questions for duplicate check: %d\n", len(askedTexts))

	// 5. 現在のスコアを分析して、次に評価すべきカテゴリを決定
	targetLevel := s.getUserTargetLevel(req.UserID)
	scores, err := s.userWeightScoreRepo.FindByUserAndSession(req.UserID, req.SessionID)
	if err != nil {
		fmt.Printf("Warning: failed to get scores for question selection: %v\n", err)
	}

	// スコア分布を分析
	scoreMap := make(map[string]int)
	evaluatedCategories := make(map[string]bool)
	for _, score := range scores {
		scoreMap[score.WeightCategory] = score.Score
		if score.Score != 0 {
			evaluatedCategories[score.WeightCategory] = true
		}
	}

	// 全カテゴリ（職種に応じて並び順を調整）
	allCategories := s.getCategoryOrder(jobCategoryID)

	// 未評価カテゴリを優先的に選択
	var targetCategory string
	unevaluatedCategories := []string{}
	weaklyEvaluatedCategories := []string{}

	for _, cat := range allCategories {
		score, exists := scoreMap[cat]
		if !exists || score == 0 {
			unevaluatedCategories = append(unevaluatedCategories, cat)
		} else if score > -3 && score < 3 {
			// スコアが-3〜3の範囲は評価が曖昧
			weaklyEvaluatedCategories = append(weaklyEvaluatedCategories, cat)
		}
	}

	if len(unevaluatedCategories) > 0 {
		targetCategory = unevaluatedCategories[0]
		fmt.Printf("Targeting unevaluated category: %s\n", targetCategory)
	} else if len(weaklyEvaluatedCategories) > 0 {
		targetCategory = weaklyEvaluatedCategories[0]
		fmt.Printf("Targeting weakly evaluated category: %s (score: %d)\n", targetCategory, scoreMap[targetCategory])
	} else {
		// 全カテゴリ評価済みなら、最もスコアが極端なものを深掘り
		maxAbsScore := 0
		for cat, score := range scoreMap {
			absScore := score
			if absScore < 0 {
				absScore = -absScore
			}
			if absScore > maxAbsScore {
				maxAbsScore = absScore
				targetCategory = cat
			}
		}
		fmt.Printf("All categories evaluated, deepening strongest: %s (score: %d)\n", targetCategory, scoreMap[targetCategory])
	}

	// 常にまずルールベース質問を試し、なければAIで生成
	var questionWeightID uint
	var aiResponse string

	// 質問生成には最新10件の履歴のみ使用（文脈を保ちつつ、プロンプトを短く）
	recentHistory := history
	if len(history) > 10 {
		recentHistory = history[len(history)-10:]
	}

	// まず、ルールベース質問から選択を試みる
	fmt.Printf("[RuleBased] Attempting to get predefined question for category: %s\n", targetCategory)
	currentPhaseName := ""
	if currentPhase != nil && currentPhase.Phase != nil {
		currentPhaseName = currentPhase.Phase.PhaseName
	}
	predefinedQ, err := s.tryGetPredefinedQuestion(req.UserID, req.SessionID, targetCategory, req.IndustryID, jobCategoryID, targetLevel, askedTexts, currentPhaseName)

	if err == nil && predefinedQ != nil {
		fmt.Printf("[RuleBased] Using predefined question (ID: %d) for category: %s\n", predefinedQ.ID, predefinedQ.Category)
		aiResponse = predefinedQ.QuestionText
		questionWeightID = predefinedQ.ID
	} else {
		// ルールベース質問がない場合、AIで生成
		fmt.Printf("[AI] No predefined question available, generating with AI for category: %s (asked: %d questions)\n", targetCategory, len(askedTexts))
		aiResponse, _, err = s.generateStrategicQuestion(ctx, recentHistory, req.UserID, req.SessionID, scoreMap, allCategories, askedTexts, req.IndustryID, jobCategoryID, targetLevel, currentPhase)
		if err != nil {
			// エラーは致命的にせずフォールバック質問を設定
			fmt.Printf("Warning: failed to generate question via AI: %v\n", err)
			fallbackQuestion := s.selectFallbackQuestion(targetCategory, jobCategoryID, targetLevel, askedTexts)
			if fallbackQuestion != "" {
				aiResponse = fallbackQuestion
			} else {
				aiResponse = "すみません、質問を生成できませんでした。少し時間をおいてからもう一度お試しください。"
			}
		}
	}
	if currentPhaseName != "" && isTextBasedQuestion(aiResponse) && !shouldForceTextQuestion(recentHistory, currentPhase) {
		if currentPhaseName == "job_analysis" || currentPhaseName == "interest_analysis" || currentPhaseName == "aptitude_analysis" || currentPhaseName == "future_analysis" {
			aiResponse = buildChoiceFallback(aiResponse, currentPhaseName)
		}
	}

	// 5. フェーズベースの完了判定
	// 全フェーズが完了しているかチェック
	allPhases, err := s.phaseRepo.FindAll()
	if err != nil {
		fmt.Printf("Warning: failed to get phases: %v\n", err)
	}
	completedProgresses, _ := s.progressRepo.FindByUserAndSession(req.UserID, req.SessionID)
	completedPhaseCount := 0
	phaseByID := make(map[uint]*models.AnalysisPhase, len(allPhases))
	for i := range allPhases {
		phaseByID[allPhases[i].ID] = &allPhases[i]
	}
	for _, p := range completedProgresses {
		phase := p.Phase
		if phase == nil {
			phase = phaseByID[p.PhaseID]
		}
		if isPhaseComplete(p.QuestionsAsked, phase) {
			completedPhaseCount++
		}
	}

	// 質問数を計算（進捗表示用）
	answeredCount := countUserAnswers(history)
	_ = allPhasesReachedMax(completedProgresses, allPhases)

	// 完了判定: 全フェーズが完了していれば終了
	isComplete := completedPhaseCount == len(allPhases)

	fmt.Printf("Diagnosis progress: %d phases completed out of %d, %d questions asked, %d/10 categories evaluated, complete: %v\n",
		completedPhaseCount, len(allPhases), answeredCount, len(evaluatedCategories), isComplete)

	// 診断完了時のメッセージは追加しない（次の回答時に完了判定する）

	// 6. AIの応答を保存
	// Guard: do not save empty assistant messages
	if strings.TrimSpace(aiResponse) != "" {
		if jobJustResolved {
			aiResponse = "ありがとうございます！それでは、適性診断を始めますね。\n\n" + aiResponse
		}
		if targetLevel == "新卒" && isVerboseQuestion(aiResponse) && isTextBasedQuestion(aiResponse) {
			simple, err := s.simplifyQuestionWithAI(ctx, aiResponse)
			if err != nil || strings.TrimSpace(simple) == "" {
				simple = s.selectFallbackQuestion(targetCategory, jobCategoryID, targetLevel, askedTexts)
			}
			if strings.TrimSpace(simple) == "" {
				simple = simplifyNewGradQuestion(aiResponse)
			}
			aiResponse = simple
		}
		// 新卒向けに表現を調整（全フェーズ共通）
		if targetLevel == "新卒" {
			aiResponse = sanitizeForNewGrad(aiResponse)
		}

		assistantMsg := &models.ChatMessage{
			SessionID:        req.SessionID,
			UserID:           req.UserID,
			Role:             "assistant",
			Content:          aiResponse,
			QuestionWeightID: questionWeightID,
		}
		if err := s.chatMessageRepo.Create(assistantMsg); err != nil {
			fmt.Printf("Warning: failed to save assistant message: %v\n", err)
			// 続行は可能にする
		}
	} else {
		// フォールバック: 空のAI応答の場合は簡易質問を返す
		fmt.Printf("Warning: skipped saving empty assistant message for session %s user %d\n", req.SessionID, req.UserID)
		aiResponse = "すみません、質問を生成できませんでした。少し時間をおいてからもう一度お試しください。"
	}

	// 7. 現在のスコアを取得
	finalScores, err := s.userWeightScoreRepo.FindByUserAndSession(req.UserID, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scores: %w", err)
	}

	// フェーズ情報を構築
	allPhasesInfo, currentPhaseInfo, _ := s.buildPhaseProgressResponse(req.UserID, req.SessionID)

	// フェーズの最大質問数の合計を計算
	totalMaxQuestions := 0
	for _, phase := range allPhases {
		totalMaxQuestions += phase.MaxQuestions
	}

	return &ChatResponse{
		Response:            aiResponse,
		QuestionWeightID:    questionWeightID,
		CurrentScores:       finalScores,
		CurrentPhase:        currentPhaseInfo,
		AllPhases:           allPhasesInfo,
		IsComplete:          isComplete,
		TotalQuestions:      totalMaxQuestions, // 全フェーズの最大質問数の合計
		AnsweredQuestions:   answeredCount,
		EvaluatedCategories: len(evaluatedCategories),
		TotalCategories:     10,
	}, nil
}

// analyzeAndUpdateWeights ユーザーの回答を分析し重み係数を更新
func (s *ChatService) analyzeAndUpdateWeights(ctx context.Context, userID uint, sessionID, message string, jobCategoryID uint) error {
	// 回答の妥当性を事前チェック
	messageTrimmed := strings.TrimSpace(message)

	// 1. 空または極端に短い回答（5文字未満は無視）
	if len([]rune(messageTrimmed)) < 5 {
		fmt.Printf("Answer too short (%d chars), skipping analysis\n", len([]rune(messageTrimmed)))
		return nil
	}

	// 2. 「わからない」などの回答パターンを検出（より厳格に）
	lowConfidencePatterns := []string{
		"わからない", "分からない", "わかりません", "分かりません",
		"よくわからない", "よく分からない", "不明", "知らない", "しらない",
		"特にない", "思いつかない", "特に無い", "ありません", "特になし", "なし",
		"無い", "ない", "いいえ", "とくにない", "とくになし",
	}

	isLowConfidence := false
	messageNormalized := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(messageTrimmed), " ", ""), "　", "")

	// 短い回答で否定的な内容の場合
	if len([]rune(messageTrimmed)) < 15 {
		for _, pattern := range lowConfidencePatterns {
			if strings.Contains(messageNormalized, pattern) {
				isLowConfidence = true
				fmt.Printf("Low confidence answer detected: '%s' contains '%s'\n", messageTrimmed, pattern)
				break
			}
		}
	}

	// わからない回答の場合は、スキップ
	if isLowConfidence {
		fmt.Printf("Skipping analysis for low confidence answer\n")
		return nil
	}

	// 3. 10文字以上の回答のみAI分析を実行
	if len([]rune(messageTrimmed)) < 10 {
		fmt.Printf("Answer too short for meaningful analysis (%d chars): %s\n", len([]rune(messageTrimmed)), messageTrimmed)
		return nil
	}

	// 会話履歴から直近の質問を取得
	history, err := s.chatMessageRepo.FindRecentBySessionID(sessionID, 5)
	if err != nil {
		fmt.Printf("Warning: failed to get history for analysis: %v\n", err)
		history = []models.ChatMessage{}
	}

	lastQuestion := ""
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			lastQuestion = history[i].Content
			break
		}
	}

	if strings.TrimSpace(lastQuestion) == "" {
		fmt.Printf("Warning: no previous question found for scoring\n")
		return nil
	}
	if s.isJobSelectionQuestion(lastQuestion) {
		fmt.Printf("Skipping analysis for job selection question\n")
		return nil
	}

	if jobCategoryID == 0 {
		fmt.Printf("Skipping job-fit scoring because job category is not set\n")
		return nil
	}

	targetCategory := s.inferCategoryFromQuestion(lastQuestion)
	isChoice := !isTextBasedQuestion(lastQuestion)

	evaluation, err := s.evaluateJobFitScoreWithAI(ctx, jobCategoryID, lastQuestion, message, isChoice)
	if err != nil {
		fmt.Printf("Warning: failed to evaluate job fit: %v\n", err)
		return nil
	}

	if evaluation.Score <= 0 {
		fmt.Printf("No job-fit score applied (score=%d)\n", evaluation.Score)
		return nil
	}

	return s.updateCategoryScore(userID, sessionID, targetCategory, evaluation.Score)
}

// generateStrategicQuestion AIが戦略的に次の質問を生成
func (s *ChatService) generateStrategicQuestion(ctx context.Context, history []models.ChatMessage, userID uint, sessionID string, scoreMap map[string]int, allCategories []string, askedTexts map[string]bool, industryID, jobCategoryID uint, targetLevel string, currentPhase *models.UserAnalysisProgress) (string, uint, error) {
	// 会話履歴を構築
	historyText := ""
	for _, msg := range history {
		historyText += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	// 既に聞いた質問のリスト（重複防止を徹底）
	askedQuestionsText := "\n## 【重要】既に聞いた質問（絶対に重複させないこと）\n"
	if len(askedTexts) == 0 {
		askedQuestionsText += "（まだ質問していません）\n"
	} else {
		questionCount := 0
		for text := range askedTexts {
			questionCount++
			askedQuestionsText += fmt.Sprintf("%d. %s\n", questionCount, text)
		}
		askedQuestionsText += fmt.Sprintf("\n**上記%d個の質問と類似・重複する質問は絶対に生成しないでください**\n", questionCount)
	}

	// スコア状況の分析
	scoreAnalysis := "## 現在の評価状況\n"
	evaluatedCategories := []string{}
	unevaluatedCategories := []string{}

	for _, cat := range allCategories {
		score, exists := scoreMap[cat]
		if exists && score != 0 {
			scoreAnalysis += fmt.Sprintf("- %s: %d点\n", cat, score)
			evaluatedCategories = append(evaluatedCategories, cat)
		} else {
			unevaluatedCategories = append(unevaluatedCategories, cat)
		}
	}

	// 職種名と業界名を取得
	jobCategoryName := "指定なし"
	if jobCategoryID != 0 {
		if jc, err := s.jobCategoryRepo.FindByID(jobCategoryID); err == nil && jc != nil {
			jobCategoryName = jc.Name
		}
	}

	// 企業選定に必要な情報を特定
	var targetCategory string
	var questionPurpose string

	if len(unevaluatedCategories) > 0 {
		// 未評価カテゴリがあれば優先
		targetCategory = unevaluatedCategories[0]
		questionPurpose = fmt.Sprintf("まだ評価できていない「%s」を評価するため", targetCategory)
	} else {
		// 全カテゴリ評価済みなら、スコアが中途半端なものを深掘り
		targetCategory = ""
		for cat, score := range scoreMap {
			if score > -3 && score < 3 {
				targetCategory = cat
				questionPurpose = fmt.Sprintf("評価が曖昧な「%s」をより明確に判定するため", cat)
				break
			}
		}

		if targetCategory == "" {
			// 最もスコアが高いカテゴリを深掘り
			highestScore := -100
			for cat, score := range scoreMap {
				if score > highestScore {
					highestScore = score
					targetCategory = cat
				}
			}
			questionPurpose = fmt.Sprintf("強みである「%s」をさらに深く評価し、最適な企業を絞り込むため", targetCategory)
		}
	}

	categoryDescriptions := map[string]string{
		"技術志向":       "技術やデジタル活用への興味、学習経験（授業、趣味、独学）→ 技術主導企業か事業主導企業か",
		"コミュニケーション力": "対話力、説明力、プレゼン経験（授業発表、サークル）→ チーム重視企業か個人裁量企業か",
		"リーダーシップ志向":  "主導性、提案力、まとめ役経験（グループワーク、サークル）→ マネジメント志向かスペシャリスト志向か",
		"チームワーク志向":   "協力、役割認識、グループ活動経験（授業、サークル、バイト）→ 大規模チーム企業か少数精鋭企業か",
		"創造性志向":      "独創性、アイデア発想、工夫した経験（課題、趣味）→ スタートアップか大企業か",
		"安定志向":       "長期的キャリア観、安定性重視 → 大手企業かベンチャーか",
		"成長志向":       "学習意欲、自己成長、新しい挑戦（資格、自主学習）→ 教育重視企業か実践重視企業か",
		"チャレンジ志向":    "困難への挑戦、失敗を恐れない姿勢 → 挑戦推奨文化か安定志向文化か",
		"細部志向":       "丁寧さ、正確性、品質へのこだわり → 品質重視企業かスピード重視企業か",
		"ワークライフバランス": "仕事と私生活のバランス観 → ワークライフバランス重視企業か成果主義企業か",
	}
	categoryDescriptionsMid := map[string]string{
		"技術志向":       "技術への興味、業務での技術活用や改善経験 → 技術主導企業か事業主導企業か",
		"コミュニケーション力": "関係者との調整、説明力、合意形成の経験 → チーム重視企業か個人裁量企業か",
		"リーダーシップ志向":  "意思決定、主導性、チームや案件の推進経験 → マネジメント志向かスペシャリスト志向か",
		"チームワーク志向":   "協力、役割認識、チームでの成果創出経験 → 大規模チーム企業か少数精鋭企業か",
		"創造性志向":      "改善提案、業務の工夫、新しいアプローチ → スタートアップか大企業か",
		"安定志向":       "長期的キャリア観、安定性重視 → 大手企業かベンチャーか",
		"成長志向":       "学習意欲、自己成長、新しい挑戦 → 教育重視企業か実践重視企業か",
		"チャレンジ志向":    "困難への挑戦、失敗を恐れない姿勢 → 挑戦推奨文化か安定志向文化か",
		"細部志向":       "丁寧さ、正確性、品質へのこだわり → 品質重視企業かスピード重視企業か",
		"ワークライフバランス": "仕事と私生活のバランス観 → ワークライフバランス重視企業か成果主義企業か",
	}

	// フェーズ情報を追加
	phaseContext := ""
	if currentPhase != nil && currentPhase.Phase != nil {
		phaseContext = fmt.Sprintf(`
## 現在の分析フェーズ: %s
%s
このフェーズでは%dつ〜%dつの質問を行います。現在%d個目の質問です。
フェーズの目的に沿った質問を生成してください。
`, currentPhase.Phase.DisplayName, currentPhase.Phase.Description,
			currentPhase.Phase.MinQuestions, currentPhase.Phase.MaxQuestions,
			currentPhase.QuestionsAsked+1)
	}
	choiceGuidance := ""
	phaseName := ""
	if currentPhase != nil && currentPhase.Phase != nil {
		phaseName = currentPhase.Phase.PhaseName
	}
	forceTextQuestion := shouldForceTextQuestion(history, currentPhase)
	if phaseName != "" {
		switch phaseName {
		case "job_analysis":
			choiceGuidance = "- 職種分析では選択肢中心で質問を構成する\n- 4〜5択で興味や方向性を選ばせ、最後に「その他（自由記述）」を用意する\n- 選択肢は必ず「A)」「B)」または「1)」「2)」形式で改行区切りで列挙する\n- 出力は『質問文 + 選択肢列挙』の形式とし、選択肢がない質問は不可\n- 文章でないと判定できない場合のみ自由記述にする（その場合も「その他（自由記述）」として選択肢に含める）"
		case "interest_analysis":
			choiceGuidance = "- 興味分析では選択肢中心で質問を構成する\n- 可能な限り4〜5択で提示し、最後に「その他（自由記述）」を用意する\n- 選択肢は必ず「A)」「B)」または「1)」「2)」形式で改行区切りで列挙する\n- 出力は『質問文 + 選択肢列挙』の形式とし、選択肢がない質問は不可\n- 文章必須の深掘りが必要な場合のみ自由記述にする（その場合も「その他（自由記述）」として選択肢に含める）"
		case "aptitude_analysis":
			choiceGuidance = "- 適性分析では選択肢中心で質問を構成する\n- 4〜5択で具体的な行動や傾向を選ばせる\n- 選択肢は必ず「A)」「B)」または「1)」「2)」形式で改行区切りで列挙する\n- 出力は『質問文 + 選択肢列挙』の形式とし、選択肢がない質問は不可\n- 文章でないと判定できない場合のみ自由記述にする（その場合も「その他（自由記述）」として選択肢に含める）"
		case "future_analysis":
			choiceGuidance = "- 将来分析（待遇・働き方の希望を含む）では選択肢中心で質問を構成する\n- 4〜5択で希望や優先順位を選ばせ、最後に「その他（自由記述）」を用意する\n- 選択肢は必ず「A)」「B)」または「1)」「2)」形式で改行区切りで列挙する\n- 出力は『質問文 + 選択肢列挙』の形式とし、選択肢がない質問は不可\n- 理由や背景が必要な場合のみ自由記述にする（その場合も「その他（自由記述）」として選択肢に含める）"
		}
	}
	if forceTextQuestion {
		choiceGuidance = "- このフェーズでは最低限の自由記述質問が必要です\n- 今回は必ず自由記述で質問を作成する\n- 選択肢は出さない"
	}
	if choiceGuidance != "" {
		choiceGuidance = fmt.Sprintf("## 質問形式の方針\n%s\n", choiceGuidance)
	}

	if strings.TrimSpace(targetLevel) == "" {
		targetLevel = "新卒"
	}

	requiresChoice := currentPhase != nil && !forceTextQuestion && (phaseName == "" || phaseName == "job_analysis" || phaseName == "interest_analysis" || phaseName == "aptitude_analysis" || phaseName == "future_analysis")

	description := categoryDescriptions[targetCategory]
	if targetLevel == "中途" {
		description = categoryDescriptionsMid[targetCategory]
	}

	var prompt string
	if targetLevel == "中途" {
		prompt = fmt.Sprintf(`あなたは中途向けの就職適性診断の専門家です。
これまでの会話と評価状況を分析し、**実務経験を引き出しやすく、企業選定に役立つ質問**を1つ生成してください。
%s
%s
## これまでの会話
%s

%s

%s

## 質問の目的
%s

## 対象カテゴリ: %s
%s

## 【重要】中途向け質問ガイドライン
- 実務経験・業務・プロジェクト・成果・数値に触れる
- 役割・判断・工夫・関係者との調整を具体的に聞く
- 抽象的ではなく、具体的なシーンを想定して聞く
- 質問は1つのみ、説明や前置きは不要
- 既出質問と重複しない

**志望職種: %s, 業界ID: %d, 職種ID: %d を考慮して、この職種に相応しい文脈で質問を生成してください。**

質問のみを返してください。説明や補足は一切不要です。`,
			phaseContext,
			choiceGuidance,
			historyText,
			scoreAnalysis,
			askedQuestionsText,
			questionPurpose,
			targetCategory,
			description,
			jobCategoryName,
			industryID,
			jobCategoryID)
	} else {
		prompt = fmt.Sprintf(`あなたは新卒学生向けの就職適性診断の専門家です。
これまでの会話と評価状況を分析し、**学生が答えやすく、企業選定に役立つ質問**を1つ生成してください。
%s
%s
## これまでの会話
%s

%s

%s

## 質問の目的
%s

## 対象カテゴリ: %s
%s

## 【重要】新卒学生向け質問作成ガイドライン

### 1. **実務経験を前提としない**
❌ 悪い例: 「プロジェクトリーダーとしての経験は？」
✅ 良い例: 「グループ活動で、自分から提案したことはありますか？」

❌ 悪い例: 「業務での課題解決経験は？」
✅ 良い例: 「授業やサークルで困ったとき、どのように対処しましたか？」

### 2. **学生生活で答えられる質問**
以下のような場面を想定：
- 授業、ゼミ、グループワーク
- サークル、部活動
- アルバイト
- 趣味、個人の活動
- 資格勉強、自主学習

### 3. **具体的で答えやすい**
抽象的な質問より、具体的なシーンを想定：
✅ 「グループワークで意見が分かれたとき、どうしましたか？」
✅ 「新しい技術やツールに触れ始めたきっかけは何ですか？」
✅ 「サークルやバイトで、どんな役割が多かったですか？」

### 4. **小さな経験も評価**
「どんな小さなことでも構いません」と添える：
✅ 「リーダー経験がなくても、自分から提案したことはありますか？」
✅ 「技術に触れた経験が少なくても、興味はありますか？」

### 5. **選択肢や例を示す**
完全にオープンではなく、具体例を示す：
✅ 「勉強するとき、A) 一人で集中する、B) 友人と一緒に、C) 先生に質問、どれが多いですか？」

## 質問の例（新卒向け・良い例）

**技術志向:**
「身近なITツールや新しい技術に触れることに興味はありますか？もし触れたことがあれば、授業、趣味、独学など、どんな形でも良いので教えてください。」

**チームワーク:**
「グループワークやサークル活動で、メンバーと協力したことはありますか？その時、あなたはどんな役割でしたか？」

**リーダーシップ:**
「グループで何かをするとき、自分から提案したり、まとめ役をしたことはありますか？どんな小さなことでも構いません。」

**問題解決:**
「課題やレポートで行き詰まったとき、どうやって解決しますか？最近の例があれば教えてください。」

**学習意欲:**
「新しいことを学ぶのは好きですか？最近、何か新しく始めたことや、挑戦したことはありますか？」

**コミュニケーション:**
「人と話すことや、自分の考えを伝えることは得意ですか？授業やサークルでの発表、アルバイトでの接客など、経験があれば教えてください。」

## 【重要】避けるべき表現

❌ 「プロジェクト」→ ✅ 「グループワーク」「課題」
❌ 「業務」→ ✅ 「活動」「勉強」
❌ 「クライアント」→ ✅ 「相手」「メンバー」
❌ 「マネジメント」→ ✅ 「まとめ役」「リーダー」
❌ 「実績」→ ✅ 「経験」「やったこと」
❌ 「スキル」→ ✅ 「できること」「学んだこと」

## 【重要】質問生成の制約
1. **重複厳禁**: 既出質問と同じ内容や類似する質問は絶対に生成しないこと
2. **簡潔明瞭**: 質問は1つのみ、説明や前置きは不要
3. **学生が答えられる**: 実務経験不要、学生生活で答えられる内容
4. **具体例を促す**: 「どんな小さなことでも」「例えば授業やサークルで」
5. **文脈の活用**: これまでの会話の流れを自然に継続
6. **進捗表示禁止**: 質問に進捗状況（例: 📊 進捗: X/10カテゴリ評価済み）を含めないこと
7. **親しみやすい言葉**: 堅苦しくなく、話しかけるような口調

**技術志向・専門性を評価する場合:**
「授業や個人制作などで取り組んだものづくりの経験があれば教えてください。使った技術やツール、担当したことがあれば教えてください。」

## 質問生成時の重要な指針
- **資格・認定について**: 適切なタイミングで、保有資格や勉強中の資格について尋ねることで、学習意欲や専門性を評価する
- **経験・実績について**: プロジェクト経験、インターン、アルバイト、課外活動などの具体的な経験を聞き出し、スキルレベルと適性を判断する
- **自然な文脈で**: 会話の流れに沿って、資格や経験について質問する（例: 技術の話題が出たら「その技術を使った経験はありますか？」）

**志望職種: %s, 業界ID: %d, 職種ID: %d を考慮して、この職種に相応しい文脈で質問を生成してください。特に「技術志向」を評価する場合は、職種がエンジニアであればプログラミングについて、非エンジニア職種ではITツール活用や効率化の関心について聞き、プログラミング経験を前提としないでください。**

質問のみを返してください。説明や補足は一切不要です。`,
		phaseContext,
		choiceGuidance,
		historyText,
		scoreAnalysis,
		askedQuestionsText,
		questionPurpose,
		targetCategory,
		description,
		jobCategoryName,
		industryID,
		jobCategoryID)
	}

	questionText, err := s.aiCallWithRetries(ctx, prompt)
	if err != nil {
		return "", 0, err
	}

	// 質問文をクリーンアップ
	questionText = strings.TrimSpace(questionText)
	questionText = strings.Trim(questionText, `"「」`)

	// フォールバック: AIが空を返した場合は簡易質問を使用する
	if questionText == "" {
		fallbackQuestion := s.selectFallbackQuestion(targetCategory, jobCategoryID, targetLevel, askedTexts)
		if fallbackQuestion != "" {
			questionText = fallbackQuestion
		} else {
			questionText = "すみません、質問を生成できませんでした。少し時間をおいてからもう一度お試しください。"
		}
	}

	// 選択肢必須フェーズで選択肢がない場合は再生成
	if requiresChoice && isTextBasedQuestion(questionText) {
		for attempt := 0; attempt < 2; attempt++ {
			choicePrompt := fmt.Sprintf(`以下の質問は選択肢が不足しています。
"%s"

必ず4〜5個の選択肢を「A)」「B)」「C)」「D)」「E)」または「1)」「2)」「3)」「4)」「5)」形式で改行区切りで列挙し、最後に「その他（自由記述）」を含めてください。

質問文は1つのみ。説明は不要です。質問文の後に選択肢を列挙してください。`, questionText)

			regenerated, err := s.aiCallWithRetries(ctx, choicePrompt)
			if err != nil {
				break
			}
			regenerated = strings.TrimSpace(regenerated)
			regenerated = strings.Trim(regenerated, `"「」`)
			if regenerated != "" {
				questionText = regenerated
			}
			if !isTextBasedQuestion(questionText) {
				break
			}
		}
		if isTextBasedQuestion(questionText) {
			questionText = buildChoiceFallback(questionText, phaseName)
		}
	}

	// 重複チェック（完全一致および類似度チェック）を最大3回まで試行
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		isDuplicate := false
		duplicateReason := ""

		// 完全一致チェック
		if askedTexts[questionText] {
			isDuplicate = true
			duplicateReason = fmt.Sprintf("完全一致: %s", questionText)
		} else {
			// 類似度チェック
			for askedQ := range askedTexts {
				similarity := calculateSimilarity(questionText, askedQ)
				if similarity > 0.6 { // 閾値を0.6に下げて、より厳格に
					isDuplicate = true
					duplicateReason = fmt.Sprintf("類似度%.2f: %s", similarity, askedQ)
					break
				}
			}
		}

		if !isDuplicate {
			break // 重複なし、使用可能
		}

		fmt.Printf("Retry %d: Duplicate detected (%s)\n", attempt+1, duplicateReason)

		// 再生成プロンプト
		retryPrompt := fmt.Sprintf(`以下の質問は既に聞いているか類似しています：
"%s"

既に聞いた全ての質問：
%s

これらと完全に異なる新しい質問を生成してください。
対象カテゴリ: %s
**質問のみ**を返してください。説明は不要です。`,
			questionText,
			func() string {
				var list string
				count := 0
				for q := range askedTexts {
					count++
					list += fmt.Sprintf("%d. %s\n", count, q)
				}
				return list
			}(),
			targetCategory)

		questionText, err = s.aiCallWithRetries(ctx, retryPrompt)
		if err != nil {
			return "", 0, err
		}
		questionText = strings.TrimSpace(questionText)
		questionText = strings.Trim(questionText, `"「」`)

		// 最後の試行で重複してもそのまま使用（無限ループ防止）
		if attempt == maxRetries-1 {
			fmt.Printf("Max retries reached, using question anyway: %s\n", questionText)
		}
	}

	// AI生成質問をデータベースに保存（空文字は保存しない）
	questionText = strings.TrimSpace(questionText)
	if questionText == "" {
		fmt.Printf("Warning: AI generated empty question even after fallback, not saving. user=%d session=%s\n", userID, sessionID)
		return "", 0, fmt.Errorf("ai returned empty question")
	}

	aiGenQuestion := &models.AIGeneratedQuestion{
		UserID:       userID,
		SessionID:    sessionID,
		TemplateID:   nil, // AI生成の場合はNULL
		QuestionText: questionText,
		Weight:       7, // 戦略的質問は重み高め
		IsAnswered:   false,
		ContextData:  fmt.Sprintf(`{"target_category": "%s", "purpose": "%s"}`, targetCategory, questionPurpose),
	}

	if err := s.aiGeneratedQuestionRepo.Create(aiGenQuestion); err != nil {
		return "", 0, fmt.Errorf("failed to save AI generated question: %w", err)
	}

	return questionText, aiGenQuestion.ID, nil
}

// calculateSimilarity 2つの文字列の類似度を計算（簡易版）
func calculateSimilarity(s1, s2 string) float64 {
	// 正規化
	s1 = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s1, " ", ""), "　", ""))
	s2 = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s2, " ", ""), "　", ""))

	// 完全一致
	if s1 == s2 {
		return 1.0
	}

	// 一方が他方を含む場合
	if strings.Contains(s1, s2) || strings.Contains(s2, s1) {
		return 0.9
	}

	// 共通の単語数をカウント
	words1 := extractKeywords(s1)
	words2 := extractKeywords(s2)

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	commonCount := 0
	for w1 := range words1 {
		if words2[w1] {
			commonCount++
		}
	}

	// Jaccard係数
	totalWords := len(words1) + len(words2) - commonCount
	if totalWords == 0 {
		return 0.0
	}

	return float64(commonCount) / float64(totalWords)
}

// extractKeywords 文字列から重要なキーワードを抽出
func extractKeywords(s string) map[string]bool {
	// ストップワードを除外
	stopWords := map[string]bool{
		"あなた": true, "ます": true, "です": true, "ですか": true, "ください": true,
		"について": true, "として": true, "という": true, "どのよう": true,
		"何": true, "どう": true, "いつ": true, "どこ": true, "誰": true,
		"か": true, "の": true, "に": true, "を": true, "は": true, "が": true,
		"で": true, "と": true, "や": true, "から": true, "まで": true,
	}

	keywords := make(map[string]bool)

	// 3文字以上の単語を抽出（簡易版）
	runes := []rune(s)
	for i := 0; i < len(runes)-2; i++ {
		word := string(runes[i : i+3])
		if !stopWords[word] {
			keywords[word] = true
		}

		// 4文字以上も試す
		if i < len(runes)-3 {
			word4 := string(runes[i : i+4])
			if !stopWords[word4] {
				keywords[word4] = true
			}
		}
	}

	return keywords
}

// handleSessionStart セッション開始時の初回質問を生成
func (s *ChatService) handleSessionStart(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	fmt.Printf("Starting new session: %s\n", req.SessionID)

	// ユーザー情報を取得
	user, err := s.userRepo.GetUserByID(req.UserID)
	userName := "あなた"
	if err == nil && user != nil && user.Name != "" {
		userName = user.Name
	}

	// 職種選択の質問を生成
	jobQuestion, err := s.jobValidator.GenerateJobSelectionQuestion(ctx)
	if err != nil {
		// エラー時のフォールバック
		jobQuestion = `初めまして！あなたの適性診断をサポートします。

まず、どの職種に興味がありますか？以下から選んでください：

1. エンジニア（プログラミング、開発）
2. 営業（顧客対応、提案）
3. マーケティング（企画、分析）
4. 人事（採用、育成）
5. その他・まだ決めていない

番号で答えても、職種名で答えても構いません。`
	} else {
		jobQuestion = fmt.Sprintf("初めまして、%sさん！あなたの適性診断をサポートします。\n\n%s", userName, jobQuestion)
	}

	response := jobQuestion

	// 初回メッセージを保存
	assistantMsg := &models.ChatMessage{
		SessionID: req.SessionID,
		UserID:    req.UserID,
		Role:      "assistant",
		Content:   response,
	}
	if err := s.chatMessageRepo.Create(assistantMsg); err != nil {
		return nil, fmt.Errorf("failed to save initial message: %w", err)
	}

	return &ChatResponse{
		Response:            response,
		IsComplete:          false,
		TotalQuestions:      15,
		AnsweredQuestions:   0,
		EvaluatedCategories: 0,
		TotalCategories:     10,
	}, nil
}

func (s *ChatService) generateQuestionWithAI(ctx context.Context, history []models.ChatMessage, userID uint, sessionID string, industryID, jobCategoryID uint) (string, uint, error) {
	// 会話履歴を構築
	historyText := ""
	hasLowConfidenceAnswer := false
	lastQuestion := ""

	for i, msg := range history {
		historyText += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)

		if msg.Role == "assistant" {
			lastQuestion = msg.Content
		}

		// 最後のユーザー回答が「わからない」系かチェック
		if i == len(history)-1 && msg.Role == "user" {
			lowConfidencePatterns := []string{
				"わからない", "わからない", "わかりません", "分かりません",
				"よくわからない", "特にない", "思いつかない", "ありません",
			}
			for _, pattern := range lowConfidencePatterns {
				if strings.Contains(strings.ToLower(msg.Content), pattern) {
					hasLowConfidenceAnswer = true
					break
				}
			}
		}
	}

	// 現在のスコアを取得して、まだ評価が不十分な領域を特定
	scores, err := s.userWeightScoreRepo.FindByUserAndSession(userID, sessionID)
	if err != nil {
		fmt.Printf("Warning: failed to get scores for question generation: %v\n", err)
	}

	// スコア分布を分析
	scoreMap := make(map[string]int)
	for _, score := range scores {
		scoreMap[score.WeightCategory] = score.Score
	}

	// まだ評価されていないカテゴリを特定（職種に応じて並び順を調整）
	allCategories := s.getCategoryOrder(jobCategoryID)

	unevaluatedCategories := []string{}
	for _, cat := range allCategories {
		if _, exists := scoreMap[cat]; !exists {
			unevaluatedCategories = append(unevaluatedCategories, cat)
		}
	}

	var prompt string
	if hasLowConfidenceAnswer {
		// わからない回答の場合は、同じカテゴリで別の角度から質問
		prompt = fmt.Sprintf(`あなたは新卒学生向けの適性診断インタビュアーです。

## これまでの会話
%s

## 状況
学生が前の質問「%s」に答えられなかったようです。
同じカテゴリで、**より答えやすい質問**を生成してください。

## 【重要】新卒学生向け質問ガイドライン

### 1. 実務経験を前提としない
❌ 「プロジェクトでの経験は？」
✅ 「授業やサークルでの経験は？」

❌ 悪い例: 「業務での課題解決経験は？」
✅ 良い例: 「授業やサークルで困ったとき、どのように対処しましたか？」

### 2. より具体的なシーンを提示
❌ 「リーダーシップについて教えて」
✅ 「グループワークで、自分から提案したことはありますか？」

### 3. 小さな経験も評価
「どんな小さなことでも構いません」と添える

### 4. 身近な例を挙げる
「例えば、授業、サークル、アルバイト、趣味など」

### 5. 選択肢や例を示す
完全にオープンではなく、具体例を示す

## 質問の例（答えやすい良い例）

**技術志向:**
「身近なITツールや新しい技術に興味はありますか？授業で触れた程度でも、使ったことがあれば教えてください。」

**チームワーク:**
「グループで作業するとき、どんな役割が多いですか？例えば、まとめ役、アイデアを出す人、サポート役など。」

**リーダーシップ:**
「友達と遊ぶ計画を立てるとき、自分から提案することはありますか？」

**コミュニケーション:**
「授業で発表したり、アルバイトで接客したりする経験はありますか？」

**避けるべき言葉:**
- プロジェクト → グループワーク、課題
- 業務 → 活動、勉強
- マネジメント → まとめ役
- 実績 → 経験、やったこと

業界ID: %d, 職種ID: %d

**質問のみ**を1つ返してください。説明や補足は不要です。`, historyText, lastQuestion, industryID, jobCategoryID)
	} else if len(unevaluatedCategories) > 0 {
		// 未評価のカテゴリがある場合は、それを重点的に評価
		targetCategory := unevaluatedCategories[0]

		categoryDescriptions := map[string]string{
			"技術志向":       "技術やデジタル活用への興味（授業、趣味、独学）",
			"コミュニケーション力": "人と話すこと、説明すること、協力すること",
			"リーダーシップ志向":  "自分から提案、まとめ役、メンバーのサポート",
			"チームワーク志向":   "グループでの協力、役割分担、助け合い",
			"創造性志向":      "アイデア発想、工夫、新しいアプローチ",
			"安定志向":       "長期的キャリア観、安定性への考え方",
			"成長志向":       "学習意欲、自己成長、新しい挑戦",
			"チャレンジ志向":    "困難への挑戦、失敗を恐れない姿勢",
			"細部志向":       "丁寧さ、正確性、品質へのこだわり",
			"ワークライフバランス": "仕事と私生活のバランス観",
		}

		description := categoryDescriptions[targetCategory]

		prompt = fmt.Sprintf(`あなたは新卒学生向けの適性診断インタビュアーです。

## これまでの会話
%s

## 次に評価すべきカテゴリ
**%s** (%s)

## 【重要】新卒学生向け質問ガイドライン

### 1. 実務経験を前提としない
学生生活で答えられる質問：
- 授業、ゼミ、グループワーク
- サークル、部活動
- アルバイト
- 趣味、個人活動

### 2. 具体的で答えやすい
❌ 「プロジェクトでの問題解決経験は？」
✅ 「課題やレポートで行き詰まったとき、どうしましたか？」

### 3. 小さな経験も評価
「どんな小さなことでも構いません」と添える

### 4. 自然な会話の流れ
これまでの会話を踏まえた質問

## 良い質問の例（新卒向け）

**技術志向:**
「技術やツールに触れた経験で、楽しかったことや苦労したことはありますか？」

**チームワーク:**
「グループ活動で、メンバーと協力してうまくいったとき、どんな気持ちでしたか？」

**リーダーシップ:**
「自分から提案したとき、周りの反応はどうでしたか？やりがいを感じましたか？」

**成長志向:**
「新しいことを学ぶとき、どんなことに気をつけていますか？直近で学んだことはありますか？」

業界ID: %d, 職種ID: %d

**質問のみ**を1つ返してください。説明や補足は不要です。`, historyText, targetCategory, description, industryID, jobCategoryID)
	} else {
		// 全カテゴリ評価済みの場合は、深掘り質問
		// スコアが高いカテゴリをさらに深掘り
		var highestCategory string
		highestScore := -100
		for cat, score := range scoreMap {
			if score > highestScore {
				highestScore = score
				highestCategory = cat
			}
		}

		prompt = fmt.Sprintf(`あなたは新卒学生向けの適性診断インタビュアーです。

## これまでの会話
%s

## 現在の評価状況
学生の強みとして「%s」が見えてきました（スコア: %d）。
この強みを深掘りし、具体的なエピソードや考え方を引き出す質問を作成してください。

## 【重要】新卒学生向け深掘り質問ガイドライン

### 1. 実務経験を前提としない
学生生活で答えられる質問：
- 授業、ゼミ、グループワーク
- サークル、部活動
- アルバイト
- 趣味、個人活動

### 2. 具体的なエピソードを引き出す
「その中で、特に印象に残っている経験はありますか？」
「それをどう感じましたか？」

### 3. 考え方や価値観を探る
「なぜそう思ったのですか？」
「それがあなたにとって大切な理由は？」

### 4. 強みの本質を確認
表面的でなく、本質的な能力や価値観を探る

### 5. 小さな経験も大切に
「どんな小さなことでも構いません」と添える

## 良い深掘り質問の例

**技術志向が強い場合:**
「新しい技術やツールに触れる中で、一番楽しかった瞬間や達成感を感じたことはありますか？」

**チームワークが強い場合:**
「グループ活動で、メンバーと協力してうまくいったとき、どんな気持ちでしたか？」

**リーダーシップが強い場合:**
「自分から提案したとき、周りの反応はどうでしたか？やりがいを感じましたか？」

**成長志向が強い場合:**
「新しいことを学び続けるモチベーションは何ですか？」

業界ID: %d, 職種ID: %d

**質問のみ**を1つ返してください。説明や補足は不要です。`, historyText, highestCategory, highestScore, industryID, jobCategoryID)
	}

	questionText, err := s.aiCallWithRetries(ctx, prompt)
	if err != nil {
		return "", 0, err
	}

	// 質問文をクリーンアップ
	questionText = strings.TrimSpace(questionText)
	questionText = strings.Trim(questionText, `"「」`)

	// AI生成質問をデータベースに保存
	aiGenQuestion := &models.AIGeneratedQuestion{
		UserID:       userID,
		SessionID:    sessionID,
		TemplateID:   nil, // AI生成の場合はNULL
		QuestionText: questionText,
		Weight:       5, // デフォルト重み
		IsAnswered:   false,
	}

	if err := s.aiGeneratedQuestionRepo.Create(aiGenQuestion); err != nil {
		return "", 0, fmt.Errorf("failed to save AI generated question: %w", err)
	}

	return questionText, aiGenQuestion.ID, nil
}

func (s *ChatService) getCategoryOrder(jobCategoryID uint) []string {
	defaultOrder := []string{
		"技術志向", "コミュニケーション能力", "リーダーシップ", "チームワーク",
		"問題解決力", "創造性・発想力", "計画性・実行力", "学習意欲・成長志向",
		"ストレス耐性・粘り強さ", "ビジネス思考・目標志向",
	}
	undecidedOrder := []string{
		"コミュニケーション能力", "学習意欲・成長志向", "問題解決力", "チームワーク",
		"ビジネス思考・目標志向", "計画性・実行力", "創造性・発想力", "ストレス耐性・粘り強さ",
		"リーダーシップ", "技術志向",
	}

	if jobCategoryID == 0 {
		return undecidedOrder
	}

	code := s.getJobCategoryCode(jobCategoryID)
	switch {
	case strings.HasPrefix(code, "ENG"):
		return []string{
			"技術志向", "問題解決力", "学習意欲・成長志向", "創造性・発想力",
			"計画性・実行力", "チームワーク", "コミュニケーション能力", "ストレス耐性・粘り強さ",
			"ビジネス思考・目標志向", "リーダーシップ",
		}
	case strings.HasPrefix(code, "SALES"):
		return []string{
			"コミュニケーション能力", "ビジネス思考・目標志向", "チームワーク", "ストレス耐性・粘り強さ",
			"計画性・実行力", "学習意欲・成長志向", "問題解決力", "リーダーシップ",
			"創造性・発想力", "技術志向",
		}
	case strings.HasPrefix(code, "MKT"):
		return []string{
			"創造性・発想力", "問題解決力", "コミュニケーション能力", "ビジネス思考・目標志向",
			"学習意欲・成長志向", "計画性・実行力", "チームワーク", "リーダーシップ",
			"ストレス耐性・粘り強さ", "技術志向",
		}
	case strings.HasPrefix(code, "HR"):
		return []string{
			"コミュニケーション能力", "チームワーク", "リーダーシップ", "学習意欲・成長志向",
			"計画性・実行力", "問題解決力", "ストレス耐性・粘り強さ", "ビジネス思考・目標志向",
			"創造性・発想力", "技術志向",
		}
	case strings.HasPrefix(code, "FIN"):
		return []string{
			"計画性・実行力", "問題解決力", "ビジネス思考・目標志向", "ストレス耐性・粘り強さ",
			"学習意欲・成長志向", "コミュニケーション能力", "チームワーク", "リーダーシップ",
			"創造性・発想力", "技術志向",
		}
	case strings.HasPrefix(code, "CONS"):
		return []string{
			"問題解決力", "コミュニケーション能力", "学習意欲・成長志向", "ビジネス思考・目標志向",
			"チームワーク", "リーダーシップ", "計画性・実行力", "ストレス耐性・粘り強さ",
			"創造性・発想力", "技術志向",
		}
	default:
		return defaultOrder
	}
}

func (s *ChatService) fallbackQuestionForCategory(category string, jobCategoryID uint, targetLevel string) string {
	switch category {
	case "技術志向":
		return s.techInterestQuestion(jobCategoryID, targetLevel)
	case "コミュニケーション能力":
		if targetLevel == "中途" {
			return "業務で関係者と調整した経験はありますか？どんな場面で、どのように進めましたか？"
		}
		return "グループワークであなたがよく担当する役割は何ですか？（例: アイデア出し、まとめ役、サポートなど）"
	case "リーダーシップ":
		if targetLevel == "中途" {
			return "業務でチームや案件をリードした経験はありますか？どのように進めましたか？"
		}
		return "グループで何かをまとめた経験はありますか？どんな場面でしたか？"
	case "チームワーク":
		if targetLevel == "中途" {
			return "チームで協力して成果を出した経験はありますか？あなたの役割も教えてください。"
		}
		return "サークルや授業で、チームで取り組んだ経験はありますか？どんな役割でしたか？"
	case "問題解決力":
		if targetLevel == "中途" {
			return "業務で課題が起きたとき、どのように解決しましたか？最近の例を教えてください。"
		}
		return "課題やレポートで困ったとき、どのように解決しましたか？最近の例を教えてください。"
	case "創造性・発想力":
		if targetLevel == "中途" {
			return "業務で改善や工夫を提案した経験はありますか？どんな内容でしたか？"
		}
		return "新しいアイデアを出した経験はありますか？どんな工夫をしましたか？"
	case "計画性・実行力":
		if targetLevel == "中途" {
			return "業務で計画を立てて実行した経験を教えてください。どのように進めましたか？"
		}
		return "何かを計画して実行した経験を教えてください。どのように進めましたか？"
	case "学習意欲・成長志向":
		if targetLevel == "中途" {
			return "業務に役立てるために学んだことはありますか？直近の例があれば教えてください。"
		}
		return "新しいことを学ぶとき、どうやって学習を進めますか？直近で学んだことはありますか？"
	case "ストレス耐性・粘り強さ":
		if targetLevel == "中途" {
			return "業務で困難に直面したとき、どのように乗り越えましたか？具体例があれば教えてください。"
		}
		return "困難に直面したとき、どのように乗り越えましたか？具体例があれば教えてください。"
	case "ビジネス思考・目標志向":
		if targetLevel == "中途" {
			return "業務で目標を立てて達成した経験はありますか？どんな目標でしたか？"
		}
		return "目標を立てて達成した経験はありますか？どんな目標でしたか？"
	default:
		return ""
	}
}

func (s *ChatService) fallbackQuestionsForCategory(category string, jobCategoryID uint, targetLevel string) []string {
	switch category {
	case "技術志向":
		return []string{
			s.techInterestQuestion(jobCategoryID, targetLevel),
			"最近触れた技術やツールはありますか？どんなことでも大丈夫です。",
		}
	case "コミュニケーション能力":
		if targetLevel == "中途" {
			return []string{
				"業務で相手に説明するとき、意識していることは何ですか？",
				"関係者とのやり取りで工夫したことはありますか？",
			}
		}
		return []string{
			"人に説明するとき、意識していることは何ですか？",
			"授業やサークルで発表した経験はありますか？",
		}
	case "リーダーシップ":
		if targetLevel == "中途" {
			return []string{
				"業務で主導したことはありますか？どんな場面でしたか？",
				"周りを巻き込んで進めた経験はありますか？",
			}
		}
		return []string{
			"自分から提案したりまとめ役をしたことはありますか？",
			"人をまとめた経験があれば教えてください。",
		}
	case "チームワーク":
		if targetLevel == "中途" {
			return []string{
				"チームで協力して進めた仕事はありますか？",
				"メンバーと連携する際に意識していることは？",
			}
		}
		return []string{
			"グループで協力した経験はありますか？",
			"チームで取り組んだときの役割を教えてください。",
		}
	case "問題解決力":
		if targetLevel == "中途" {
			return []string{
				"業務で困ったとき、どう解決しましたか？",
				"トラブル対応で工夫したことはありますか？",
			}
		}
		return []string{
			"困ったとき、どうやって解決しましたか？",
			"課題で行き詰まったときの対処法を教えてください。",
		}
	case "創造性・発想力":
		if targetLevel == "中途" {
			return []string{
				"業務で改善案を出したことはありますか？",
				"新しいアイデアを提案した経験はありますか？",
			}
		}
		return []string{
			"新しいアイデアを出した経験はありますか？",
			"いつもと違う工夫をしたことはありますか？",
		}
	case "計画性・実行力":
		if targetLevel == "中途" {
			return []string{
				"業務で計画を立てて進めた経験はありますか？",
				"期限に向けて進めた仕事はありますか？",
			}
		}
		return []string{
			"計画を立てて進めた経験はありますか？",
			"期限を意識して進めたことはありますか？",
		}
	case "学習意欲・成長志向":
		if targetLevel == "中途" {
			return []string{
				"最近学んだことはありますか？",
				"仕事のために学習したことはありますか？",
			}
		}
		return []string{
			"最近学んだことはありますか？",
			"新しく始めたことはありますか？",
		}
	case "ストレス耐性・粘り強さ":
		if targetLevel == "中途" {
			return []string{
				"大変だった仕事をどう乗り越えましたか？",
				"プレッシャーのある場面での対処を教えてください。",
			}
		}
		return []string{
			"大変なとき、どうやって乗り越えましたか？",
			"うまくいかない時の気持ちの切り替え方は？",
		}
	case "ビジネス思考・目標志向":
		if targetLevel == "中途" {
			return []string{
				"目標を立てて取り組んだ経験はありますか？",
				"成果を意識して進めた仕事はありますか？",
			}
		}
		return []string{
			"目標を立てて取り組んだ経験はありますか？",
			"目標達成のために工夫したことはありますか？",
		}
	default:
		return []string{s.fallbackQuestionForCategory(category, jobCategoryID, targetLevel)}
	}
}

func (s *ChatService) selectFallbackQuestion(category string, jobCategoryID uint, targetLevel string, askedTexts map[string]bool) string {
	options := s.fallbackQuestionsForCategory(category, jobCategoryID, targetLevel)
	for _, q := range options {
		if strings.TrimSpace(q) == "" {
			continue
		}
		if !askedTexts[q] {
			return q
		}
	}
	generic := []string{}
	if targetLevel == "中途" {
		generic = []string{
			"最近取り組んだ仕事やタスクはありますか？簡単に教えてください。",
			"仕事で工夫したことがあれば教えてください。",
		}
	} else {
		generic = []string{
			"最近頑張ったことはありますか？",
			"新しく挑戦したことはありますか？",
		}
	}
	for _, q := range generic {
		if strings.TrimSpace(q) == "" {
			continue
		}
		if !askedTexts[q] {
			return q
		}
	}
	return ""
}

func (s *ChatService) techInterestQuestion(jobCategoryID uint, targetLevel string) string {
	code := s.getJobCategoryCode(jobCategoryID)
	if targetLevel == "中途" {
		switch {
		case strings.HasPrefix(code, "ENG"):
			return "業務で使った技術や、最近取り組んだ開発について教えてください。"
		case strings.HasPrefix(code, "SALES"):
			return "営業活動でITツールや仕組みを活用した経験はありますか？どのように使いましたか？"
		case strings.HasPrefix(code, "MKT"):
			return "データやデジタルを使った施策の経験はありますか？内容を教えてください。"
		case strings.HasPrefix(code, "HR"):
			return "人事領域でITツールや仕組みを使った経験はありますか？具体例があれば教えてください。"
		case strings.HasPrefix(code, "FIN"):
			return "数値管理や分析で使ったツール・仕組みがあれば教えてください。"
		case strings.HasPrefix(code, "CONS"):
			return "業務でデータやツールを使って課題整理をした経験はありますか？"
		default:
			return "業務でITツールや仕組みを活用した経験はありますか？"
		}
	}
	switch {
	case strings.HasPrefix(code, "ENG"):
		return "プログラミングや技術に触れるのは好きですか？授業や趣味、独学で触れたことがあれば教えてください。"
	case strings.HasPrefix(code, "SALES"):
		return "営業で役立ちそうなITツールやアプリを使うことに興味はありますか？授業やアルバイトで使ったことがあれば教えてください。"
	case strings.HasPrefix(code, "MKT"):
		return "データやSNS分析など、デジタルを使って考えることに興味はありますか？授業や趣味で触れたことがあれば教えてください。"
	case strings.HasPrefix(code, "HR"):
		return "人事の仕事で役立ちそうなITツールや仕組みに興味はありますか？授業やアルバイトで使ったことがあれば教えてください。"
	case strings.HasPrefix(code, "FIN"):
		return "数字を扱う作業や表計算などのツールを使うのは好きですか？授業やアルバイトで使ったことがあれば教えてください。"
	case strings.HasPrefix(code, "CONS"):
		return "調べた情報をまとめるためにITツールやデータを使うことに興味はありますか？授業や課題での経験があれば教えてください。"
	default:
		return "身近なITツールやアプリを使って作業を効率化することに興味はありますか？授業やアルバイトで使った例があれば教えてください。"
	}
}

func (s *ChatService) getJobCategoryCode(jobCategoryID uint) string {
	if jobCategoryID == 0 {
		return ""
	}
	category, err := s.jobCategoryRepo.FindByID(jobCategoryID)
	if err != nil || category == nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(category.Code))
}

func (s *ChatService) getUserTargetLevel(userID uint) string {
	user, err := s.userRepo.GetUserByID(userID)
	if err == nil && user != nil && strings.TrimSpace(user.TargetLevel) == "中途" {
		return "中途"
	}
	return "新卒"
}

type jobFitEvaluation struct {
	Score           int      `json:"score"`
	Reason          string   `json:"reason"`
	MatchedKeywords []string `json:"matched_keywords"`
}

func (s *ChatService) getJobCategoryName(jobCategoryID uint) string {
	if jobCategoryID == 0 {
		return "未指定"
	}
	category, err := s.jobCategoryRepo.FindByID(jobCategoryID)
	if err != nil || category == nil {
		return "未指定"
	}
	return strings.TrimSpace(category.Name)
}

func (s *ChatService) getLastAssistantMessage(history []models.ChatMessage) string {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			return history[i].Content
		}
	}
	return ""
}

func (s *ChatService) isJobSelectionQuestion(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	keywords := []string{
		"職種", "どの職種", "IT職種", "興味がありますか", "選んでください",
		"まだ決めていない", "番号で答えても",
	}
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func (s *ChatService) shouldValidateJobCategory(history []models.ChatMessage) bool {
	lastAssistant := s.getLastAssistantMessage(history)
	if strings.TrimSpace(lastAssistant) == "" {
		return true
	}
	return s.isJobSelectionQuestion(lastAssistant)
}

func (s *ChatService) getJobFitKeywords(jobCategoryID uint) ([]string, []string) {
	code := s.getJobCategoryCode(jobCategoryID)
	switch {
	case strings.HasPrefix(code, "ENG"):
		return []string{"プログラミング", "開発", "コード", "設計", "デバッグ"},
			[]string{"アルゴリズム", "API", "テスト", "Git", "サーバー", "データベース"}
	case strings.HasPrefix(code, "SALES"):
		return []string{"提案", "顧客", "ヒアリング", "関係構築", "課題"},
			[]string{"ニーズ", "交渉", "フォロー", "目標", "商談"}
	case strings.HasPrefix(code, "MKT"):
		return []string{"分析", "企画", "データ", "広告", "改善"},
			[]string{"SNS", "市場", "ターゲット", "施策", "検証"}
	case strings.HasPrefix(code, "HR"):
		return []string{"採用", "面接", "人材", "育成", "評価"},
			[]string{"研修", "面談", "制度", "組織", "コミュニケーション"}
	case strings.HasPrefix(code, "FIN"):
		return []string{"会計", "財務", "数値", "予算", "分析"},
			[]string{"収支", "コスト", "利益", "報告", "精算"}
	case strings.HasPrefix(code, "CONS"):
		return []string{"課題", "分析", "提案", "改善", "戦略"},
			[]string{"ヒアリング", "資料", "仮説", "整理", "意思決定"}
	default:
		return []string{}, []string{}
	}
}

func (s *ChatService) evaluateJobFitScoreWithAI(ctx context.Context, jobCategoryID uint, question, answer string, isChoice bool) (*jobFitEvaluation, error) {
	jobName := s.getJobCategoryName(jobCategoryID)
	jobCode := s.getJobCategoryCode(jobCategoryID)
	coreKeywords, relatedKeywords := s.getJobFitKeywords(jobCategoryID)

	questionType := "文章"
	if isChoice {
		questionType = "選択肢"
	}

	prompt := fmt.Sprintf(`あなたは就職適性診断の採点者です。以下のルールに従って採点してください。

## 職種
%s (%s)

## 質問（%s）
%s

## 回答
%s

## 職種理解キーワード
- 必須キーワード: %s
- 関連キーワード: %s

## 採点ルール
### 選択肢問題
- 回答が職種に最も適している場合: 90〜100点
- 適しても不適切でもない場合: 40〜70点
- 全く適していない場合: 0〜20点

### 文章問題
- 必須キーワードがすべて含まれる場合: 90〜100点
- 1語以上含まれる場合: 含まれた語数に応じて加点（1語=10点、最大80点）
- 1語も含まれない場合: 0点

## 出力形式（JSONのみ）
{"score": 0, "reason": "理由", "matched_keywords": ["キーワード"]}`,
		jobName,
		jobCode,
		questionType,
		question,
		answer,
		strings.Join(coreKeywords, ", "),
		strings.Join(relatedKeywords, ", "),
	)

	response, err := s.aiCallWithRetries(ctx, prompt)
	if err != nil {
		return nil, err
	}

	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("invalid JSON response for job fit evaluation")
	}

	var evaluation jobFitEvaluation
	if err := json.Unmarshal([]byte(response[jsonStart:jsonEnd+1]), &evaluation); err != nil {
		return nil, fmt.Errorf("failed to parse job fit evaluation: %w", err)
	}

	return &evaluation, nil
}

// sanitizeForNewGrad 新卒向けに質問文を個人志向に書き換える
func sanitizeForNewGrad(q string) string {
	if strings.TrimSpace(q) == "" {
		return q
	}
	// 一般的な置換ルール（軽量）
	q = strings.ReplaceAll(q, "この会社", "あなた")
	q = strings.ReplaceAll(q, "会社で", "学ぶ場で")
	q = strings.ReplaceAll(q, "採用する", "学ぶ")
	q = strings.ReplaceAll(q, "採用しますか", "学びたいですか")
	q = strings.ReplaceAll(q, "導入", "学ぶこと")
	q = strings.ReplaceAll(q, "導入しますか", "学びますか")
	q = strings.ReplaceAll(q, "業務", "活動")
	q = strings.ReplaceAll(q, "プロジェクト", "グループワーク")
	q = strings.ReplaceAll(q, "クライアント", "相手")
	q = strings.ReplaceAll(q, "マネジメント", "まとめ役")
	q = strings.ReplaceAll(q, "KPI", "目標")
	q = strings.ReplaceAll(q, "売上", "成果")
	q = strings.ReplaceAll(q, "実績", "経験")
	q = strings.ReplaceAll(q, "現場", "活動の場")

	// パターン置換: 「新しい技術 .* 採用」-> 「新しい技術を学ぶことに興味がありますか」
	re := regexp.MustCompile(`(?i)新しい技術[\s\S]{0,30}採用`)
	if re.MatchString(q) {
		q = re.ReplaceAllString(q, "新しい技術を学ぶことに興味はありますか")
	}

	// 不自然な表現の微修正
	q = strings.ReplaceAll(q, "あなたは学ぶ", "あなたは学ぶことに興味がありますか")

	// 最後にトリム
	q = strings.TrimSpace(q)
	return q
}

func isVerboseQuestion(q string) bool {
	if strings.TrimSpace(q) == "" {
		return false
	}
	if len([]rune(q)) > 120 {
		return true
	}
	if strings.Contains(q, "（") || strings.Contains(q, "例：") || strings.Contains(q, "例:") || strings.Contains(q, "例えば") {
		return true
	}
	if strings.Count(q, "？")+strings.Count(q, "?") > 1 {
		return true
	}
	if strings.Count(q, "\n") > 1 {
		return true
	}
	return false
}

func simplifyNewGradQuestion(q string) string {
	s := strings.TrimSpace(q)
	if s == "" {
		return s
	}
	if idx := strings.Index(s, "（"); idx > 0 {
		s = strings.TrimSpace(s[:idx])
	}
	if idx := strings.Index(s, "例"); idx > 0 {
		s = strings.TrimSpace(s[:idx])
	}
	s = strings.ReplaceAll(s, "\n", " ")
	if len([]rune(s)) > 120 {
		s = string([]rune(s)[:120])
	}
	if !strings.HasSuffix(s, "？") && !strings.HasSuffix(s, "?") {
		s += "？"
	}
	return s
}

func countUserAnswers(history []models.ChatMessage) int {
	count := 0
	for _, msg := range history {
		if msg.Role == "user" {
			count++
		}
	}
	return count
}

func (s *ChatService) getLastPhaseProgress(userID uint, sessionID string) (*models.UserAnalysisProgress, error) {
	progresses, err := s.progressRepo.FindByUserAndSession(userID, sessionID)
	if err != nil {
		return nil, err
	}
	if len(progresses) == 0 {
		return nil, fmt.Errorf("no phase progress found")
	}
	return &progresses[len(progresses)-1], nil
}

func allPhasesReachedMax(progresses []models.UserAnalysisProgress, phases []models.AnalysisPhase) bool {
	if len(phases) == 0 {
		return false
	}
	progressMap := make(map[uint]models.UserAnalysisProgress, len(progresses))
	for _, p := range progresses {
		progressMap[p.PhaseID] = p
	}
	for _, phase := range phases {
		p, ok := progressMap[phase.ID]
		if !ok {
			return false
		}
		if phase.MaxQuestions > 0 && p.QuestionsAsked < phase.MaxQuestions {
			return false
		}
	}
	return true
}

func (s *ChatService) simplifyQuestionWithAI(ctx context.Context, question string) (string, error) {
	prompt := fmt.Sprintf(`次の質問を、新卒でも答えやすい短い質問に言い換えてください。

制約:
- 1文で、40〜80文字程度
- 例示やカッコ補足は入れない
- 同じ意味を保つ
- 質問文のみを返す

質問:
%s`, question)
	return s.aiCallWithRetries(ctx, prompt)
}

// GetChatHistory チャット履歴を取得
func (s *ChatService) GetChatHistory(sessionID string) ([]models.ChatMessage, error) {
	return s.chatMessageRepo.FindBySessionID(sessionID)
}

// GetUserScores ユーザーのスコアを取得
func (s *ChatService) GetUserScores(userID uint, sessionID string) ([]models.UserWeightScore, error) {
	return s.userWeightScoreRepo.FindByUserAndSession(userID, sessionID)
}

// GetTopRecommendations トップNの適性カテゴリを取得
func (s *ChatService) GetTopRecommendations(userID uint, sessionID string, limit int) ([]models.UserWeightScore, error) {
	return s.userWeightScoreRepo.FindTopCategories(userID, sessionID, limit)
}

// GetUserChatSessions ユーザーのチャットセッション一覧を取得
func (s *ChatService) GetUserChatSessions(userID uint) ([]models.ChatSession, error) {
	return s.chatMessageRepo.GetUserSessions(userID)
}

// checkAnswerValidity: 直近の assistant メッセージが質問かを判定し、ユーザー入力がその質問に対する有効な回答かを判定する。
// 無効な場合はアシスタントの「書かれた内容にはお答えできません」メッセージを保存して true を返す。
// 3回連続で無効な場合はセッションを強制終了する。
// 戻り値: handled(bool) - true の場合は処理を終了してよい、response(string) - 保存したアシスタント応答（ある場合）、error
func (s *ChatService) checkAnswerValidity(ctx context.Context, history []models.ChatMessage, userMessage string, userID uint, sessionID string) (bool, string, error) {
	// 直近の assistant メッセージを探す
	var lastAssistant *models.ChatMessage
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			lastAssistant = &history[i]
			break
		}
	}

	// アシスタントメッセージがない場合、またはそれが質問でない場合
	// → これは初回や説明メッセージの直後なので、職種に関する回答を期待する
	var questionText string
	if lastAssistant == nil {
		// 履歴がない場合は、初回の職種選択を期待
		questionText = "どのようなIT職種に興味がありますか？"
	} else if !isQuestion(lastAssistant.Content) {
		// 質問ではない場合（説明文など）も、職種に関する回答を期待
		questionText = "IT業界のどのような職種に興味がありますか？"
	} else {
		// 通常の質問の場合
		questionText = lastAssistant.Content
	}

	// ユーザー回答が質問に対する答えかどうか判定
	isValid, err := s.validateAnswerRelevance(ctx, questionText, userMessage)
	if err != nil {
		// AI判定エラー時は基本的な検証のみ
		fmt.Printf("[Validation] AI validation failed: %v, using basic validation\n", err)
		isValid = isLikelyAnswer(userMessage, questionText)
		fmt.Printf("[Validation] Basic validation result: %v for message: %s\n", isValid, userMessage)
	} else {
		fmt.Printf("[Validation] AI validation result: %v for message: %s\n", isValid, userMessage)
	}

	if isValid {
		// 有効な回答と判断 -> カウントをリセットして既存の処理に進める
		fmt.Printf("[Validation] Valid answer detected, resetting invalid count for session: %s\n", sessionID)
		if err := s.sessionValidationRepo.ResetInvalidCount(sessionID); err != nil {
			fmt.Printf("Warning: failed to reset invalid count: %v\n", err)
		}
		return false, "", nil
	}

	// 無効な回答と判断 -> カウントをインクリメント
	fmt.Printf("[Validation] Invalid answer detected for message: %s\n", userMessage)
	validation, err := s.sessionValidationRepo.IncrementInvalidCount(sessionID)
	if err != nil {
		return true, "", fmt.Errorf("failed to increment invalid count: %w", err)
	}
	fmt.Printf("[Validation] Invalid count incremented to: %d/3\n", validation.InvalidAnswerCount)

	var assistantText string
	if validation.InvalidAnswerCount >= 3 {
		// 3回目の無効回答 -> セッションを強制終了
		if err := s.sessionValidationRepo.TerminateSession(sessionID); err != nil {
			fmt.Printf("Warning: failed to terminate session: %v\n", err)
		}
		assistantText = "申し訳ございませんが、質問と関係のない内容が3回続いたため、チャットを終了させていただきます。新しいセッションで最初からやり直してください。"
	} else {
		// 1-2回目の無効回答 -> 警告メッセージ
		assistantText = fmt.Sprintf("書かれた内容にはお答えできません。質問に回答してください。（%d/3回目の警告）", validation.InvalidAnswerCount)
	}

	assistantMsg := &models.ChatMessage{
		SessionID: sessionID,
		UserID:    userID,
		Role:      "assistant",
		Content:   assistantText,
	}
	if err := s.chatMessageRepo.Create(assistantMsg); err != nil {
		return true, "", fmt.Errorf("failed to save assistant message for invalid answer: %w", err)
	}
	return true, assistantText, nil
}

// isQuestion: アシスタントのメッセージが「質問」であるか粗く判定する
func isQuestion(text string) bool {
	txt := strings.TrimSpace(text)
	if txt == "" {
		return false
	}
	// 疑問符があれば質問とみなす
	if strings.ContainsAny(txt, "？?") {
		return true
	}
	// 日本語の疑問語が含まれるか確認
	questionWords := []string{"どのよう", "どの", "どう", "なぜ", "なに", "何", "いつ", "どれ", "どこ", "どなた", "どんな", "〜ますか", "ますか", "でしょうか"}
	for _, w := range questionWords {
		if strings.Contains(txt, w) {
			return true
		}
	}
	return false
}

func normalizeQuestionText(text string) string {
	questionText := strings.TrimSpace(text)
	if questionText == "" {
		return ""
	}
	// 💡マークなどのヒント部分を除去
	if idx := strings.Index(questionText, "\n\n💡"); idx > 0 {
		questionText = questionText[:idx]
	}
	// 段落末尾の質問文を優先して抽出（前置きが付くケースを考慮）
	parts := strings.Split(questionText, "\n\n")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}
		if isQuestion(part) {
			return part
		}
	}
	if isQuestion(questionText) {
		return questionText
	}
	return ""
}

// validateAnswerRelevance: 回答が質問に沿っているかを判定（文章系はキーワードベースで柔軟に判定）
func (s *ChatService) validateAnswerRelevance(ctx context.Context, question, answer string) (bool, error) {
	// 文章系の質問かどうかを判定
	isTextQuestion := isTextBasedQuestion(question)

	if isTextQuestion {
		// 文章系の質問: キーワードベースで柔軟に判定
		fmt.Printf("[Validation] Text-based question detected, using keyword-based validation\n")
		return isLikelyAnswer(answer, question), nil
	}

	// 選択肢型の質問: AI判定を使用
	fmt.Printf("[Validation] Choice-based question detected, using AI validation\n")

	systemPrompt := `あなたは回答の妥当性を判定する審査AIです。

## 重要な制約
- 必ずJSON形式のみで応答してください
- 他の説明文やコメントは一切含めないでください

## 出力形式（厳守）
{"valid": true} または {"valid": false}`

	userPrompt := fmt.Sprintf(`以下の質問に対するユーザーの回答が適切かどうかを判定してください。

## 質問
%s

## ユーザーの回答
%s

## 判定基準
以下のいずれかに該当する場合は有効な回答とみなす：
1. 選択肢記号（A、B、C、1、2、3など）が含まれている
2. 質問に対する明確な選択や意思表示がある
3. 「はい」「いいえ」などの意思表示

以下の場合のみ無効とする：
- 挨拶のみ
- 完全に無関係な話題
- 質問を完全に無視した内容

## 判定
{"valid": true} または {"valid": false}`, question, answer)

	// temperature=0で安定した判定を行う
	response, err := s.aiClient.ResponsesWithTemperature(ctx, systemPrompt, userPrompt, 0.0)
	if err != nil {
		return false, fmt.Errorf("AI validation error: %w", err)
	}

	// コードフェンスを除去してJSON抽出
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// JSON構造体で検証
	type ValidationResult struct {
		Valid bool `json:"valid"`
	}

	var result ValidationResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// JSONパースに失敗した場合は無効とみなす
		fmt.Printf("Warning: Failed to parse AI validation response: %v, response: %s\n", err, response)
		return false, nil
	}

	return result.Valid, nil
}

// isTextBasedQuestion: 質問が文章系（具体的なエピソードを求める）かどうかを判定
func isTextBasedQuestion(question string) bool {
	// 選択肢型の質問のパターン
	choicePatterns := []string{
		"A)", "B)", "C)", "D)", "E)",
		"A：", "B：", "C：", "D：", "E：",
		"A、", "B、", "C、", "D、", "E、",
		"1)", "2)", "3)", "4)", "5)",
		"①", "②", "③", "④", "⑤",
		"1〜5", "1～5", "1-5",
		// numbered dot formats (1. , 1．) を選択肢として扱う
		"1.", "2.", "3.", "4.", "5.", "1．", "2．", "3．", "4．", "5．",
	}

	for _, pattern := range choicePatterns {
		if strings.Contains(question, pattern) {
			return false // 選択肢型
		}
	}

	// 文章系の質問のキーワード
	textPatterns := []string{
		"具体的", "エピソード", "経験", "体験",
		"教えてください", "教えて下さい",
		"について話して", "について教えて",
		"どのように", "どんな",
	}

	for _, pattern := range textPatterns {
		if strings.Contains(question, pattern) {
			return true // 文章系
		}
	}

	// デフォルトは文章系として扱う（柔軟に判定）
	return true
}

// isLikelyAnswer: ユーザーの入力が質問に対する「回答らしい」かを判定する簡易ロジック（フォールバック用）
// AI判定が失敗した場合の適度に柔軟なフォールバック
func isLikelyAnswer(answer, question string) bool {
	a := strings.TrimSpace(answer)

	// 記号のみの回答（A, B, 1, 2など）は有効
	if len([]rune(a)) <= 3 && strings.ContainsAny(a, "ABCDEabcde12345①②③④⑤") {
		fmt.Printf("[Validation] Fallback: Valid choice symbol: %s\n", a)
		return true
	}

	// 3文字未満は無効（ただし上で選択肢判定済み）
	if len([]rune(a)) < 3 {
		fmt.Printf("[Validation] Fallback: Too short (< 3 chars): %s\n", a)
		return false
	}

	// 挨拶・感謝などの雑談パターンは無効
	if containsGreeting(a) {
		fmt.Printf("[Validation] Fallback: Contains greeting: %s\n", a)
		return false
	}

	// 明らかな無回答パターンをチェック（「わからない」単体のみ無効）
	noAnswerPatterns := []string{
		"わからない", "分からない", "わかりません", "分かりません",
	}
	answerLower := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(a, " ", ""), "　", ""))
	for _, pattern := range noAnswerPatterns {
		// 「わからない」だけの回答のみ無効（他の文章が続く場合は有効）
		if answerLower == pattern || answerLower == pattern+"。" {
			fmt.Printf("[Validation] Fallback: No-answer pattern detected: %s\n", a)
			return false
		}
	}

	// 「はい」「いいえ」「好き」「嫌い」などの短い回答も有効
	shortValidAnswers := []string{
		"はい", "いいえ", "yes", "no", "好き", "嫌い", "得意", "苦手",
		"できる", "できない", "ある", "ない", "する", "しない",
	}
	for _, valid := range shortValidAnswers {
		if strings.Contains(strings.ToLower(a), valid) {
			fmt.Printf("[Validation] Fallback: Valid short answer: %s\n", a)
			return true
		}
	}

	// IT職種関連のキーワードを含むかチェック
	itKeywords := []string{
		"エンジニア", "プログラマ", "開発", "インフラ", "セキュリティ",
		"データ", "サイエンティスト", "アプリ", "Web", "モバイル",
		"フロントエンド", "バックエンド", "フルスタック", "DevOps",
		"クラウド", "ネットワーク", "システム", "プロジェクト",
		"技術", "スキル", "経験", "プログラミング", "コード",
	}

	hasITKeyword := false
	for _, keyword := range itKeywords {
		if strings.Contains(a, keyword) {
			hasITKeyword = true
			break
		}
	}

	// 質問文に選択肢や具体例が含まれている場合、回答側に数字や選択肢文字があれば回答とみなす
	if strings.Contains(question, "A)") || strings.Contains(question, "A：") || strings.Contains(question, "A、") {
		if strings.ContainsAny(a, "ABCDabcd1-5①②③④") {
			fmt.Printf("[Validation] Fallback: Contains choice character: %s\n", a)
			return true
		}
	}

	// IT関連キーワードを含む、または5文字以上なら有効（緩和）
	if hasITKeyword || len([]rune(a)) >= 5 {
		fmt.Printf("[Validation] Fallback: Valid answer (IT keyword or >= 5 chars): %s\n", a)
		return true
	}

	// 質問文から抽出したキーワードと回答に共通語があるかを確認する（簡易）
	qk := extractKeywords(question)
	ak := extractKeywords(a)
	common := 0
	for w := range qk {
		if ak[w] {
			common++
		}
	}

	// 共通キーワードが1つ以上あれば回答とみなす（緩和）
	if common >= 1 {
		fmt.Printf("[Validation] Fallback: Common keywords >= 1: %s\n", a)
		return true
	}

	// デフォルトは無効（厳格に判断）
	fmt.Printf("[Validation] Fallback: Default INVALID for: %s\n", a)
	return false
}

// containsGreeting: 簡易的な雑談フラグ（挨拶・感謝・了承など）
func containsGreeting(s string) bool {
	l := strings.ToLower(s)
	greetings := []string{
		"こんにちは", "こんばんは", "おはよう", "ありがとう", "ありがとうございます",
		"了解", "わかった", "わかりました", "よろしく", "ありがとうござい",
		"はい", "いいえ", "ok", "オッケー",
	}
	for _, g := range greetings {
		if strings.Contains(l, g) {
			return true
		}
	}
	return false
}

// getCurrentOrNextPhase 現在のフェーズを取得または次のフェーズを開始
func (s *ChatService) getCurrentOrNextPhase(ctx context.Context, userID uint, sessionID string) (*models.UserAnalysisProgress, error) {
	allPhases, err := s.phaseRepo.FindAll()
	if err != nil {
		return nil, err
	}

	progresses, _ := s.progressRepo.FindByUserAndSession(userID, sessionID)
	progressMap := make(map[uint]*models.UserAnalysisProgress, len(progresses))
	for i := range progresses {
		progressMap[progresses[i].PhaseID] = &progresses[i]
	}

	// 次の未完了フェーズを見つける
	for _, phase := range allPhases {
		if progress, exists := progressMap[phase.ID]; exists {
			if progress.Phase == nil {
				phaseCopy := phase
				progress.Phase = &phaseCopy
			}
			if isPhaseComplete(progress.QuestionsAsked, progress.Phase) {
				continue
			}
			return progress, nil
		}
		return s.progressRepo.FindOrCreate(userID, sessionID, phase.ID)
	}

	// 全フェーズ完了
	return nil, fmt.Errorf("all phases completed")
}

// updatePhaseProgress フェーズの進捗を更新
func (s *ChatService) updatePhaseProgress(progress *models.UserAnalysisProgress, isValidAnswer bool) error {
	progress.QuestionsAsked++
	if isValidAnswer {
		progress.ValidAnswers++
	} else {
		progress.InvalidAnswers++
	}

	progress.CompletionScore = phaseCompletionScore(progress.QuestionsAsked, progress.Phase)
	newIsCompleted := isPhaseComplete(progress.QuestionsAsked, progress.Phase)
	if newIsCompleted {
		if !progress.IsCompleted {
			now := time.Now()
			progress.CompletedAt = &now
		}
	} else {
		progress.CompletedAt = nil
	}
	progress.IsCompleted = newIsCompleted

	return s.progressRepo.Update(progress)
}

// buildPhaseProgressResponse フェーズ進捗レスポンスを構築
func (s *ChatService) buildPhaseProgressResponse(userID uint, sessionID string) ([]PhaseProgress, *PhaseProgress, error) {
	progresses, _ := s.progressRepo.FindByUserAndSession(userID, sessionID)
	allPhases, err := s.phaseRepo.FindAll()
	if err != nil {
		return nil, nil, err
	}

	progressMap := make(map[uint]*models.UserAnalysisProgress)
	for i := range progresses {
		progressMap[progresses[i].PhaseID] = &progresses[i]
	}

	var result []PhaseProgress
	var current *PhaseProgress

	for _, phase := range allPhases {
		pp := PhaseProgress{
			PhaseID:      phase.ID,
			PhaseName:    phase.PhaseName,
			DisplayName:  phase.DisplayName,
			MinQuestions: phase.MinQuestions,
			MaxQuestions: phase.MaxQuestions,
		}

		if progress, exists := progressMap[phase.ID]; exists {
			completionScore := phaseCompletionScore(progress.QuestionsAsked, &phase)
			pp.QuestionsAsked = progress.QuestionsAsked
			pp.ValidAnswers = progress.ValidAnswers
			pp.CompletionScore = completionScore
			pp.IsCompleted = isPhaseComplete(progress.QuestionsAsked, &phase)

			if !pp.IsCompleted && current == nil {
				current = &pp
			}
		}

		result = append(result, pp)
	}

	return result, current, nil
}

func phaseCompletionScore(questionsAsked int, phase *models.AnalysisPhase) float64 {
	if phase == nil {
		return 0
	}
	required := phase.MaxQuestions
	if required <= 0 {
		required = phase.MinQuestions
	}
	if required <= 0 {
		return 0
	}
	score := (float64(questionsAsked) / float64(required)) * 100
	if score > 100 {
		return 100
	}
	if score < 0 {
		return 0
	}
	return score
}

func isPhaseComplete(questionsAsked int, phase *models.AnalysisPhase) bool {
	if phase == nil {
		return false
	}
	required := phase.MaxQuestions
	if required <= 0 {
		required = phase.MinQuestions
	}
	if required <= 0 {
		return false
	}
	return questionsAsked >= required
}

// isChoiceAnswer 選択肢回答かどうかを判定
func (s *ChatService) isChoiceAnswer(answer string) bool {
	answer = strings.ToUpper(strings.TrimSpace(answer))
	// A-E または 1-5 の形式
	return answer == "A" || answer == "B" || answer == "C" || answer == "D" || answer == "E" ||
		answer == "1" || answer == "2" || answer == "3" || answer == "4" || answer == "5"
}

// processChoiceAnswer 選択肢回答を処理してスコアを更新
func (s *ChatService) processChoiceAnswer(ctx context.Context, userID uint, sessionID, answer string, history []models.ChatMessage, jobCategoryID uint) error {
	// 最後のAIの質問を取得
	var lastQuestion string
	var targetCategory string

	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			lastQuestion = history[i].Content
			break
		}
	}

	if lastQuestion == "" {
		return fmt.Errorf("no previous question found")
	}
	if s.isJobSelectionQuestion(lastQuestion) {
		fmt.Printf("[Choice Answer] Skipping score update for job selection question\n")
		return nil
	}

	// AIが生成した質問から対象カテゴリを特定
	aiQuestions, err := s.aiGeneratedQuestionRepo.FindByUserAndSession(userID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get AI questions: %w", err)
	}

	for i := len(aiQuestions) - 1; i >= 0; i-- {
		if strings.Contains(lastQuestion, aiQuestions[i].QuestionText) ||
			strings.Contains(aiQuestions[i].QuestionText, strings.Split(lastQuestion, "\n")[0]) {
			if aiQuestions[i].Template != nil {
				targetCategory = aiQuestions[i].Template.Category
			}
			break
		}
	}

	if targetCategory == "" {
		// カテゴリが特定できない場合は、質問文から推測
		targetCategory = s.inferCategoryFromQuestion(lastQuestion)
	}

	fmt.Printf("[Choice Answer] Processing choice '%s' for category: %s\n", answer, targetCategory)

	score := 0
	if jobCategoryID != 0 {
		evaluation, err := s.evaluateJobFitScoreWithAI(ctx, jobCategoryID, lastQuestion, answer, true)
		if err != nil {
			fmt.Printf("Warning: failed to evaluate job fit for choice answer: %v\n", err)
		} else {
			score = evaluation.Score
		}
	}
	if score == 0 {
		// フォールバック: 選択肢をスコアに変換（A=100, B=67, C=33, D=0 のスケール）
		score = s.convertChoiceToScore(answer)
	}

	// スコアを保存または更新
	return s.updateCategoryScore(userID, sessionID, targetCategory, score)
}

// convertChoiceToScore 選択肢をスコアに変換
func (s *ChatService) convertChoiceToScore(choice string) int {
	choice = strings.ToUpper(strings.TrimSpace(choice))
	switch choice {
	case "A", "1":
		return 100 // 非常に高い/強く同意
	case "B", "2":
		return 75 // やや高い/やや同意
	case "C", "3":
		return 50 // 中立/どちらでもない
	case "D", "4":
		return 25 // やや低い/やや不同意
	case "E", "5":
		return 0 // 低い/不同意
	default:
		return 50 // デフォルト
	}
}

func shouldForceTextQuestion(history []models.ChatMessage, currentPhase *models.UserAnalysisProgress) bool {
	if currentPhase == nil || currentPhase.Phase == nil {
		return false
	}
	minText := minTextQuestionsForPhase(currentPhase.Phase.PhaseName)
	if minText == 0 {
		return false
	}
	if currentPhase.QuestionsAsked == 0 {
		return true
	}
	textCount := countTextQuestionsInPhase(history, currentPhase.QuestionsAsked)
	return textCount < minText
}

func minTextQuestionsForPhase(phaseName string) int {
	switch phaseName {
	case "job_analysis", "interest_analysis", "aptitude_analysis", "future_analysis":
		return 1
	default:
		return 0
	}
}

func countTextQuestionsInPhase(history []models.ChatMessage, phaseQuestionsAsked int) int {
	if phaseQuestionsAsked <= 0 {
		return 0
	}
	textCount := 0
	questionCount := 0
	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		if msg.Role != "assistant" {
			continue
		}
		questionText := normalizeQuestionText(msg.Content)
		if questionText == "" || !isQuestion(questionText) {
			continue
		}
		questionCount++
		if isTextBasedQuestion(questionText) {
			textCount++
		}
		if questionCount >= phaseQuestionsAsked {
			break
		}
	}
	return textCount
}

func buildChoiceFallback(questionText, phaseName string) string {
	choices := []string{}
	switch phaseName {
	case "job_analysis":
		choices = []string{
			"1) ものづくり・開発系（Web/アプリ/設計）",
			"2) データ・分析系（分析/企画/改善）",
			"3) インフラ・運用系（基盤/安定稼働）",
			"4) 対人・調整系（営業/人事/サポート）",
			"5) その他（自由記述）",
		}
	case "interest_analysis":
		choices = []string{
			"1) 新しい技術やツールに触れる",
			"2) 仕組みを考えたり設計する",
			"3) 人と関わりながら進める",
			"4) コツコツ改善・整理する",
			"5) その他（自由記述）",
		}
	case "aptitude_analysis":
		choices = []string{
			"1) 自分から主導して進める",
			"2) みんなで協力して進める",
			"3) 支える・サポート役に回る",
			"4) 一人で集中して進める",
			"5) その他（自由記述）",
		}
	case "future_analysis":
		choices = []string{
			"1) 安定や福利厚生を重視",
			"2) 成長や挑戦を重視",
			"3) ワークライフバランス重視",
			"4) 裁量や自由度重視",
			"5) その他（自由記述）",
		}
	default:
		choices = []string{
			"1) とても当てはまる",
			"2) まあ当てはまる",
			"3) あまり当てはまらない",
			"4) まったく当てはまらない",
			"5) その他（自由記述）",
		}
	}
	return fmt.Sprintf("%s\n\n%s", strings.TrimSpace(questionText), strings.Join(choices, "\n"))
}

// inferCategoryFromQuestion 質問文からカテゴリを推測
func (s *ChatService) inferCategoryFromQuestion(question string) string {
	categoryKeywords := map[string][]string{
		"技術志向":       {"技術", "プログラミング", "コーディング", "アルゴリズム", "システム設計", "新しい技術", "技術的"},
		"チームワーク":     {"チーム", "協力", "協働", "連携", "メンバー", "共同"},
		"リーダーシップ":    {"リーダー", "指導", "率いる", "マネジメント", "方向性", "意思決定"},
		"創造性":        {"創造", "アイデア", "発想", "革新", "イノベーション", "新しい"},
		"安定志向":       {"安定", "確実", "堅実", "リスク回避", "慎重"},
		"成長志向":       {"成長", "キャリア", "昇進", "スキルアップ", "学習"},
		"ワークライフバランス": {"ワークライフ", "残業", "休日", "プライベート", "働き方"},
		"チャレンジ志向":    {"チャレンジ", "挑戦", "困難", "新しいこと", "未経験"},
		"細部志向":       {"細部", "詳細", "正確", "精密", "丁寧"},
		"コミュニケーション力": {"コミュニケーション", "説明", "伝える", "対話", "話す", "プレゼン"},
	}

	questionLower := strings.ToLower(question)
	for category, keywords := range categoryKeywords {
		for _, keyword := range keywords {
			if strings.Contains(questionLower, strings.ToLower(keyword)) {
				return category
			}
		}
	}

	return "技術志向" // デフォルト
}

// updateCategoryScore カテゴリスコアを更新
func (s *ChatService) updateCategoryScore(userID uint, sessionID, category string, score int) error {
	// 既存のスコアを取得
	existingScore, err := s.userWeightScoreRepo.FindByUserSessionAndCategory(userID, sessionID, category)

	if err != nil || existingScore == nil {
		// 新規作成
		if err := s.userWeightScoreRepo.UpdateScore(userID, sessionID, category, score); err != nil {
			return fmt.Errorf("failed to create score: %w", err)
		}
		fmt.Printf("[Choice Answer] Created new score: %s = %d\n", category, score)
	} else {
		// 平均値で更新（複数回の回答を考慮）
		newScore := (existingScore.Score + score) / 2
		existingScore.Score = newScore
		if err := s.userWeightScoreRepo.UpdateScore(userID, sessionID, category, newScore-existingScore.Score); err != nil {
			return fmt.Errorf("failed to update score: %w", err)
		}
		fmt.Printf("[Choice Answer] Updated score: %s = %d (average)\n", category, newScore)
	}

	return nil
}

// tryGetPredefinedQuestion ルールベースの事前定義質問を取得
func (s *ChatService) tryGetPredefinedQuestion(userID uint, sessionID string, prioritizeCategory string, industryID, jobCategoryID uint, targetLevel string, askedTexts map[string]bool, currentPhase string) (*models.PredefinedQuestion, error) {
	if jobCategoryID == 0 {
		// 職種未決定の場合はAI質問に任せる
		return nil, nil
	}
	if strings.TrimSpace(targetLevel) == "" {
		targetLevel = "新卒"
	}

	// すべての事前定義質問を取得して、質問文でフィルタ
	allQuestions, err := s.predefinedQuestionRepo.FindActiveQuestions(targetLevel, &industryID, &jobCategoryID, currentPhase)
	if err != nil {
		return nil, err
	}

	// 職種に合う質問のみ残す（汎用質問はAIに任せる）
	jobSpecificQuestions := make([]*models.PredefinedQuestion, 0, len(allQuestions))
	for _, q := range allQuestions {
		if q.JobCategoryID == nil || *q.JobCategoryID != jobCategoryID {
			continue
		}
		jobSpecificQuestions = append(jobSpecificQuestions, q)
	}

	if len(jobSpecificQuestions) == 0 {
		return nil, nil
	}

	// 優先カテゴリで質問を検索（該当がなければAIに任せる）
	var selected *models.PredefinedQuestion
	for _, q := range jobSpecificQuestions {
		if _, asked := askedTexts[q.QuestionText]; asked {
			continue
		}
		if prioritizeCategory != "" && q.Category != prioritizeCategory {
			continue
		}
		if selected == nil || q.Priority > selected.Priority || (q.Priority == selected.Priority && q.ID < selected.ID) {
			selected = q
		}
	}

	if selected == nil {
		return nil, nil
	}

	return selected, nil
}

// aiCallWithRetries AI呼び出しをリトライして安定化させる（最大3回）
func (s *ChatService) aiCallWithRetries(ctx context.Context, prompt string) (string, error) {
	var resp string
	var err error
	backoffs := []time.Duration{500 * time.Millisecond, 1 * time.Second, 2 * time.Second}
	for i := 0; i < len(backoffs); i++ {
		resp, err = s.aiClient.Responses(ctx, prompt)
		if err == nil && strings.TrimSpace(resp) != "" {
			return resp, nil
		}
		if err == nil {
			err = errors.New("empty response")
		}
		// log and wait before retry
		fmt.Printf("Warning: AI call failed or empty response (attempt %d): %v\n", i+1, err)
		if i == len(backoffs)-1 {
			break
		}
		select {
		case <-time.After(backoffs[i]):
			// continue
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	// last attempt with final call (no extra wait)
	resp, err = s.aiClient.Responses(ctx, prompt)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(resp) == "" {
		return "", errors.New("empty response")
	}
	return resp, nil
}
