package services

import (
	"Backend/internal/models"
	"Backend/internal/openai"
	"Backend/internal/repositories"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ChatService struct {
	aiClient                *openai.Client
	questionWeightRepo      *repositories.QuestionWeightRepository
	chatMessageRepo         *repositories.ChatMessageRepository
	userWeightScoreRepo     *repositories.UserWeightScoreRepository
	aiGeneratedQuestionRepo *repositories.AIGeneratedQuestionRepository
	userRepo                *repositories.UserRepository
	phaseRepo               *repositories.AnalysisPhaseRepository
	progressRepo            *repositories.UserAnalysisProgressRepository
	sessionValidationRepo   *repositories.SessionValidationRepository
}

func NewChatService(
	aiClient *openai.Client,
	questionWeightRepo *repositories.QuestionWeightRepository,
	chatMessageRepo *repositories.ChatMessageRepository,
	userWeightScoreRepo *repositories.UserWeightScoreRepository,
	aiGeneratedQuestionRepo *repositories.AIGeneratedQuestionRepository,
	userRepo *repositories.UserRepository,
	phaseRepo *repositories.AnalysisPhaseRepository,
	progressRepo *repositories.UserAnalysisProgressRepository,
	sessionValidationRepo *repositories.SessionValidationRepository,
) *ChatService {
	return &ChatService{
		aiClient:                aiClient,
		questionWeightRepo:      questionWeightRepo,
		chatMessageRepo:         chatMessageRepo,
		userWeightScoreRepo:     userWeightScoreRepo,
		aiGeneratedQuestionRepo: aiGeneratedQuestionRepo,
		userRepo:                userRepo,
		phaseRepo:               phaseRepo,
		progressRepo:            progressRepo,
		sessionValidationRepo:   sessionValidationRepo,
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

	// 2. 会話履歴を取得（全履歴を取得して重複チェックに使用）
	history, err := s.chatMessageRepo.FindRecentBySessionID(req.SessionID, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat history: %w", err)
	}

	// 2.5. 回答の妥当性チェック
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
		// 全フェーズ完了の場合は特別な応答を返す
		if err.Error() == "all phases completed" {
			completionMsg := "分析が完了しました！あなたに最適な企業をマッチングしました。「結果を見る」ボタンから詳細をご確認ください。"

			// 完了メッセージを保存
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
				AnsweredQuestions:   15,
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
	if err := s.analyzeAndUpdateWeights(ctx, req.UserID, req.SessionID, req.Message); err != nil {
		// ログに記録するが、処理は継続
		fmt.Printf("Warning: failed to update weights: %v\n", err)
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
		askedTexts[q.QuestionText] = true
	}

	// 4-2. チャット履歴からもアシスタントの質問を収集
	for _, msg := range history {
		if msg.Role == "assistant" {
			// 質問文を正規化して記録
			questionText := strings.TrimSpace(msg.Content)
			// 💡マークなどのヒント部分を除去
			if idx := strings.Index(questionText, "\n\n💡"); idx > 0 {
				questionText = questionText[:idx]
			}
			askedTexts[questionText] = true
		}
	}

	fmt.Printf("Total asked questions for duplicate check: %d\n", len(askedTexts))

	// 5. 現在のスコアを分析して、次に評価すべきカテゴリを決定
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

	// 全カテゴリ
	allCategories := []string{
		"技術志向", "コミュニケーション能力", "リーダーシップ", "チームワーク",
		"問題解決力", "創造性・発想力", "計画性・実行力", "学習意欲・成長志向",
		"ストレス耐性・粘り強さ", "ビジネス思考・目標志向",
	}

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

	// 常にAIで戦略的に質問を生成
	var questionWeightID uint
	var aiResponse string

	// 質問生成には最新10件の履歴のみ使用（文脈を保ちつつ、プロンプトを短く）
	recentHistory := history
	if len(history) > 10 {
		recentHistory = history[len(history)-10:]
	}

	fmt.Printf("Generating strategic question with AI for category: %s (asked: %d questions)\n", targetCategory, len(askedTexts))
	aiResponse, _, err = s.generateStrategicQuestion(ctx, recentHistory, req.UserID, req.SessionID, scoreMap, allCategories, askedTexts, req.IndustryID, req.JobCategoryID, currentPhase)
	if err != nil {
		return nil, fmt.Errorf("failed to generate question: %w", err)
	}

	// 5. フェーズベースの完了判定
	// 全フェーズが完了しているかチェック
	allPhases, err := s.phaseRepo.FindAll()
	if err != nil {
		fmt.Printf("Warning: failed to get phases: %v\n", err)
	}
	completedProgresses, _ := s.progressRepo.FindByUserAndSession(req.UserID, req.SessionID)
	completedPhaseCount := 0
	for _, p := range completedProgresses {
		if p.IsCompleted {
			completedPhaseCount++
		}
	}

	isComplete := completedPhaseCount >= len(allPhases)

	// 質問数と評価カテゴリ数を計算（進捗表示用）
	answeredQuestions, _ := s.aiGeneratedQuestionRepo.FindByUserAndSession(req.UserID, req.SessionID)
	answeredCount := len(answeredQuestions)

	fmt.Printf("Diagnosis progress: %d phases completed out of %d, %d questions asked, %d/10 categories evaluated, complete: %v\n",
		completedPhaseCount, len(allPhases), answeredCount, len(evaluatedCategories), isComplete)

	// 診断完了時のメッセージを追加（保存前に）
	if isComplete {
		completionMessage := "\n\n✅ 全てのフェーズが完了しました！あなたの適性を分析し、最適な企業をマッチングします。"
		aiResponse = aiResponse + completionMessage
	}

	// 6. AIの応答を保存
	assistantMsg := &models.ChatMessage{
		SessionID:        req.SessionID,
		UserID:           req.UserID,
		Role:             "assistant",
		Content:          aiResponse,
		QuestionWeightID: questionWeightID,
	}
	if err := s.chatMessageRepo.Create(assistantMsg); err != nil {
		return nil, fmt.Errorf("failed to save assistant message: %w", err)
	}

	// 7. 現在のスコアを取得
	finalScores, err := s.userWeightScoreRepo.FindByUserAndSession(req.UserID, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scores: %w", err)
	}

	// フェーズ情報を構築
	allPhasesInfo, currentPhaseInfo, _ := s.buildPhaseProgressResponse(req.UserID, req.SessionID)

	return &ChatResponse{
		Response:            aiResponse,
		QuestionWeightID:    questionWeightID,
		CurrentScores:       finalScores,
		CurrentPhase:        currentPhaseInfo,
		AllPhases:           allPhasesInfo,
		IsComplete:          isComplete,
		TotalQuestions:      len(allPhases) * 3, // 各フェーズ平均3問と想定
		AnsweredQuestions:   answeredCount,
		EvaluatedCategories: len(evaluatedCategories),
		TotalCategories:     10,
	}, nil
}

// analyzeAndUpdateWeights ユーザーの回答を分析し重み係数を更新
func (s *ChatService) analyzeAndUpdateWeights(ctx context.Context, userID uint, sessionID, message string) error {
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

	// 会話履歴を取得して文脈を理解（最新5件のみ）
	history, err := s.chatMessageRepo.FindRecentBySessionID(sessionID, 5)
	if err != nil {
		fmt.Printf("Warning: failed to get history for analysis: %v\n", err)
		history = []models.ChatMessage{}
	}

	// 会話履歴から質問と回答のペアを抽出
	conversationContext := ""
	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		if msg.Role == "assistant" || msg.Role == "user" {
			conversationContext += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
		}
	}

	// 簡潔な分析プロンプト
	prompt := fmt.Sprintf(`あなたは就職活動の適性診断専門家です。以下の回答を分析し、スコアリングしてください。

## 会話
%s

## 最新回答
%s

## 評価カテゴリ（-10〜+10で評価）

## 評価カテゴリ（-10〜+10で評価）

1. 技術志向: プログラミング・技術への興味
2. コミュニケーション能力: 対話力・説明力
3. リーダーシップ: 主導性・意思決定力
4. チームワーク: 協働・協調性
5. 問題解決力: 論理思考・分析力
6. 創造性・発想力: 独創性・革新性
7. 計画性・実行力: 目標設定・タスク管理
8. 学習意欲・成長志向: 継続学習・成長意識
9. ストレス耐性・粘り強さ: 困難対処・プレッシャー対応
10. ビジネス思考・目標志向: ビジネス価値理解・成果志向

## 重要
- 判断材料がない場合は0点
- 必ずJSON形式で返す
- 短く簡潔な理由を記載

## 出力形式（この形式を厳守）
{
  "技術志向": {"score": 0, "reason": "理由"},
  "コミュニケーション能力": {"score": 0, "reason": "理由"},
  "リーダーシップ": {"score": 0, "reason": "理由"},
  "チームワーク": {"score": 0, "reason": "理由"},
  "問題解決力": {"score": 0, "reason": "理由"},
  "創造性・発想力": {"score": 0, "reason": "理由"},
  "計画性・実行力": {"score": 0, "reason": "理由"},
  "学習意欲・成長志向": {"score": 0, "reason": "理由"},
  "ストレス耐性・粘り強さ": {"score": 0, "reason": "理由"},
  "ビジネス思考・目標志向": {"score": 0, "reason": "理由"}
}`, conversationContext, message)

	response, err := s.aiClient.Responses(ctx, prompt)
	if err != nil {
		return err
	}

	// JSONパース
	type ScoreDetail struct {
		Score  int    `json:"score"`
		Reason string `json:"reason"`
	}
	var scores map[string]ScoreDetail

	// JSONブロックを抽出
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart == -1 || jsonEnd == -1 {
		fmt.Printf("Warning: No JSON found in AI response, skipping score update\n")
		return nil // JSONが見つからない場合はスキップ（エラーにしない）
	}
	jsonStr := response[jsonStart : jsonEnd+1]

	if err := json.Unmarshal([]byte(jsonStr), &scores); err != nil {
		fmt.Printf("Warning: failed to parse AI response JSON: %v\nResponse: %s\n", err, jsonStr)
		return nil // 解析失敗してもスキップ（エラーにしない）
	}

	// スコアを更新（スコアが0でないもののみ）
	for category, detail := range scores {
		if detail.Score != 0 {
			if err := s.userWeightScoreRepo.UpdateScore(userID, sessionID, category, detail.Score); err != nil {
				fmt.Printf("Warning: failed to update score for %s: %v\n", category, err)
			} else {
				fmt.Printf("Updated score: %s = %d (%s)\n", category, detail.Score, detail.Reason)
			}
		}
	}

	return nil
}

// generateStrategicQuestion AIが戦略的に次の質問を生成
func (s *ChatService) generateStrategicQuestion(ctx context.Context, history []models.ChatMessage, userID uint, sessionID string, scoreMap map[string]int, allCategories []string, askedTexts map[string]bool, industryID, jobCategoryID uint, currentPhase *models.UserAnalysisProgress) (string, uint, error) {
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
		"技術志向":        "プログラミング、新技術への興味、技術的深掘り → 技術主導企業かサポート企業か",
		"コミュニケーション能力": "対話力、説明力、協調性 → チーム重視企業か個人裁量企業か",
		"リーダーシップ":     "主導性、意思決定、メンバー育成 → マネジメント志向かスペシャリスト志向か",
		"チームワーク":      "協働、役割認識、チーム貢献 → 大規模チーム企業か少数精鋭企業か",
		"問題解決力":       "論理思考、課題分析、解決策創出 → コンサル系か開発系か",
		"創造性・発想力":     "独創性、革新性、新アプローチ → スタートアップか大企業か",
		"計画性・実行力":     "目標設定、タスク管理、完遂力 → プロジェクト型企業か運用型企業か",
		"学習意欲・成長志向":   "継続学習、成長意識、フィードバック受容 → 教育重視企業か実践重視企業か",
		"ストレス耐性・粘り強さ": "困難対処、プレッシャー対応 → 高負荷環境かワークライフバランス重視か",
		"ビジネス思考・目標志向": "ビジネス価値理解、成果志向 → 事業会社か受託開発か",
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

	prompt := fmt.Sprintf(`あなたは就職活動の適性診断と企業マッチングの専門家です。
これまでの会話と評価状況を分析し、**企業選定に直結する戦略的な質問**を1つ生成してください。
%s
## これまでの会話
%s

%s

%s

## 質問の目的
%s

## 対象カテゴリ: %s
%s

## 企業選定との関連性を重視した質問作成ガイドライン

### 1. **企業タイプの絞り込みに直結**
質問への回答が、以下のような企業選定の判断材料になること：
- スタートアップ vs 大企業
- 自社開発 vs 受託開発
- 技術特化 vs ビジネス重視
- グローバル vs 国内
- チーム型 vs 個人裁量型

### 2. **具体的な状況設定**
抽象的な質問ではなく、実際の業務シーンを想定：
- 「新しいプロジェクトが始まるとき、あなたは...」
- 「チームで意見が分かれたとき、あなたは...」
- 「締め切りが迫っているとき、あなたは...」

### 3. **段階的な選択肢の提示**
完全なオープン質問より、選択肢や具体例を示す：
- 「A、B、Cのような状況で、どのアプローチを取りますか？」
- 「1〜5のうち、どれに近いですか？」

### 4. **深掘りと文脈理解**
これまでの回答を踏まえた自然な流れ：
- 前の回答で触れた内容を掘り下げる
- 矛盾や曖昧な点を明確にする

### 5. **企業文化との適合性を判定**
- 失敗への向き合い方 → 挑戦を推奨する文化 vs 安定志向
- 意思決定のスタイル → トップダウン vs ボトムアップ
- 働き方の優先順位 → 成果重視 vs プロセス重視

## 質問の例（良い例）

**技術志向を評価する場合:**
「新しい技術やツールを学ぶとき、どのようなアプローチを取りますか？
A) 公式ドキュメントを読み込んで体系的に理解する
B) まず実際に手を動かしてみて、必要に応じて調べる
C) チュートリアルや解説記事を参考に学ぶ
D) 経験者に教えてもらいながら学ぶ」

**チームワークを評価する場合:**
「チームメンバーが困っているとき、あなたはどのように行動しますか？具体的なエピソードがあれば教えてください。」

**ビジネス思考を評価する場合:**
「作ったシステムやプロダクトについて、どのような点を最も重視しますか？
- 技術的な完成度
- ユーザーの使いやすさ
- ビジネスへの貢献
- 保守性や拡張性」

## 【重要】質問生成の制約
1. **重複厳禁**: 既出質問と同じ内容や類似する質問は絶対に生成しないこと
2. **簡潔明瞭**: 質問は1つのみ、説明や前置きは不要
3. **回答可能性**: 学生が具体的に答えられる質問
4. **目的の明確化**: 何を評価したいかを明確に
5. **文脈の活用**: これまでの会話の流れを自然に継続
6. **進捗表示禁止**: 質問に進捗状況（例: 📊 進捗: X/10カテゴリ評価済み）を含めないこと

## 質問の例（良い例）

**技術志向を評価する場合:**
「プログラミングを学ぶとき、あなたはどのようなアプローチを取ることが多いですか？具体的な経験があれば教えてください。」

**チームワークを評価する場合:**
「これまでのプロジェクトや活動で、チームメンバーと協力して成果を出した経験について教えてください。あなたはどのような役割を果たしましたか？」

**問題解決力を評価する場合:**
「困難な課題に直面したとき、あなたはどのように解決策を見つけますか？最近の具体例があれば教えてください。」

**業界ID: %d, 職種ID: %d を考慮して質問を生成してください。**

質問のみを返してください。説明や補足は一切不要です。`,
		phaseContext,
		historyText,
		scoreAnalysis,
		askedQuestionsText,
		questionPurpose,
		targetCategory,
		categoryDescriptions[targetCategory],
		industryID,
		jobCategoryID)

	questionText, err := s.aiClient.Responses(ctx, prompt)
	if err != nil {
		return "", 0, err
	}

	// 質問文をクリーンアップ
	questionText = strings.TrimSpace(questionText)
	questionText = strings.Trim(questionText, `"「」`)

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

		questionText, err = s.aiClient.Responses(ctx, retryPrompt)
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

	// AI生成質問をデータベースに保存
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

	// 初回メッセージを生成
	initialPrompt := fmt.Sprintf(`あなたは「ソフィア」という名前のIT業界専門キャリアエージェントです。
これから就職活動中の学生と会話を始めます。

## ユーザー情報
- ユーザー名: %s

## 最初のメッセージの方針
- 簡潔に自己紹介する（「初めまして、ソフィアです」程度）
- IT業界のどの分野に興味があるか聞く
- シンプルで答えやすい質問にする

**挨拶と質問を簡潔に生成してください。**`, userName)

	response, err := s.aiClient.Responses(ctx, initialPrompt)
	if err != nil {
		// AIエラー時のフォールバック
		response = fmt.Sprintf("初めまして、ソフィアです。IT業界のどの分野に興味がありますか？", userName)
	}

	response = strings.TrimSpace(response)
	response = strings.Trim(response, `"「」`)

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
				"わからない", "分からない", "わかりません", "分かりません",
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

	// まだ評価されていないカテゴリを特定
	allCategories := []string{
		"技術志向", "コミュニケーション能力", "リーダーシップ", "チームワーク",
		"問題解決力", "創造性・発想力", "計画性・実行力", "学習意欲・成長志向",
		"ストレス耐性・粘り強さ", "ビジネス思考・目標志向",
	}

	unevaluatedCategories := []string{}
	for _, cat := range allCategories {
		if _, exists := scoreMap[cat]; !exists {
			unevaluatedCategories = append(unevaluatedCategories, cat)
		}
	}

	var prompt string
	if hasLowConfidenceAnswer {
		// わからない回答の場合は、同じカテゴリで別の角度から質問
		prompt = fmt.Sprintf(`あなたは就活適性診断のための優秀なインタビュアーです。

## これまでの会話
%s

## 状況
ユーザーが前の質問「%s」に答えられなかったようです。
同じカテゴリで、より答えやすい質問を生成してください。

## 質問作成のガイドライン
1. **具体的な状況設定**: 抽象的な質問ではなく、具体的なシーンを想定した質問
2. **経験ベース**: 「もし〜だったら」より「今までに〜したことは」という形式
3. **段階的アプローチ**: いきなり難しい質問ではなく、小さな経験から聞く
4. **選択肢を提示**: 完全にオープンな質問ではなく、いくつかの例を示す
5. **日常的な例**: 特別な経験でなくても答えられる質問

## 例
悪い例: 「あなたのリーダーシップについて教えてください」
良い例: 「グループワークや部活動で、自分から提案したり、メンバーをまとめたりした経験はありますか？どんな小さなことでも構いません」

業界ID: %d, 職種ID: %d

**質問のみ**を1つ返してください。`, historyText, lastQuestion, industryID, jobCategoryID)
	} else if len(unevaluatedCategories) > 0 {
		// 未評価のカテゴリがある場合は、それを重点的に評価
		targetCategory := unevaluatedCategories[0]

		categoryDescriptions := map[string]string{
			"技術志向":        "プログラミング、技術学習、技術的課題への興味",
			"コミュニケーション能力": "他者との対話、説明力、協調性",
			"リーダーシップ":     "チームを率いる、意思決定、メンバーのサポート",
			"チームワーク":      "協力、役割分担、チーム目標への貢献",
			"問題解決力":       "論理的思考、課題分析、解決策の創出",
			"創造性・発想力":     "アイデア創出、新しいアプローチ、革新的思考",
			"計画性・実行力":     "目標設定、計画立案、タスク管理、完遂力",
			"学習意欲・成長志向":   "継続学習、フィードバック受容、成長への意識",
			"ストレス耐性・粘り強さ": "困難への対処、プレッシャー対応、粘り強さ",
			"ビジネス思考・目標志向": "ビジネス価値理解、成果志向、戦略的思考",
		}

		description := categoryDescriptions[targetCategory]

		prompt = fmt.Sprintf(`あなたは就活適性診断のための優秀なインタビュアーです。

## これまでの会話
%s

## 次に評価すべきカテゴリ
**%s** (%s)

## 質問作成のガイドライン
1. **自然な流れ**: これまでの会話の流れを踏まえ、唐突でない質問
2. **具体性**: 抽象的ではなく、具体的な経験や行動を引き出す
3. **深掘り**: 表面的でなく、本質的な適性を見極められる質問
4. **答えやすさ**: 学生が具体的なエピソードで答えられる質問
5. **複数の観点**: 1つの質問で複数の側面を評価できるように工夫

## 良い質問の例
- 「プロジェクトで予期せぬ問題が発生したとき、どのように対処しましたか？具体的なエピソードを教えてください」
- 「チームメンバーと意見が対立したとき、どのように解決しましたか？」
- 「最近、自分から進んで学んだことは何ですか？それを学ぼうと思ったきっかけは？」

業界ID: %d, 職種ID: %d

**質問のみ**を1つ返してください。`, historyText, targetCategory, description, industryID, jobCategoryID)
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

		prompt = fmt.Sprintf(`あなたは就活適性診断のための優秀なインタビュアーです。

## これまでの会話
%s

## 現在の評価状況
ユーザーの強みとして「%s」が見えてきました（スコア: %d）。
この強みをさらに深掘りし、具体的なエピソードや行動特性を引き出す質問を作成してください。

## 質問作成のガイドライン
1. **深い洞察**: 表面的でなく、本質的な能力や価値観を探る
2. **具体的エピソード**: 実際の経験に基づいた詳細を引き出す
3. **行動特性**: どのように考え、行動したかを明確にする
4. **強みの確認**: その強みが本物かを検証できる質問
5. **キャリア適合**: その強みがキャリアでどう活きるか考えさせる

業界ID: %d, 職種ID: %d

**質問のみ**を1つ返してください。`, historyText, highestCategory, highestScore, industryID, jobCategoryID)
	}

	questionText, err := s.aiClient.Responses(ctx, prompt)
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

// validateAnswerRelevance: AIを使って回答が質問に沿っているかを判定
// moderateでスタブ化（実運用ではモデレーションAPIを呼ぶ）
// temperature=0で厳格に判定し、JSONのみを返す
func (s *ChatService) validateAnswerRelevance(ctx context.Context, question, answer string) (bool, error) {
	systemPrompt := `あなたは回答の妥当性を判定する厳格な審査AIです。

## 重要な制約
- 必ずJSON形式のみで応答してください
- 他の説明文やコメントは一切含めないでください
- 無関係な発言は絶対に禁止です

## 出力形式（厳守）
{"valid": true} または {"valid": false}

この形式以外の応答は絶対に行わないでください。`

	userPrompt := fmt.Sprintf(`以下の質問に対するユーザーの回答が適切かどうかを判定してください。

## 質問
%s

## ユーザーの回答
%s

## 判定基準（厳格）
1. IT業界・職種に関する具体的な内容であること
2. 質問の内容に直接関連していること
3. 挨拶のみ（「こんにちは」「よろしく」など）は無効
4. 無関係な話題（天気、日常会話など）は無効
5. 「わからない」「特にない」のみは無効
6. 最低5文字以上の意味のある回答であること

## 判定
上記基準に基づき、JSON形式で回答の妥当性を判定してください。
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

// isLikelyAnswer: ユーザーの入力が質問に対する「回答らしい」かを判定する簡易ロジック（フォールバック用）
// AI判定が失敗した場合の厳格なフォールバック
func isLikelyAnswer(answer, question string) bool {
	a := strings.TrimSpace(answer)

	// 5文字未満は無効（厳格化）
	if len([]rune(a)) < 5 {
		fmt.Printf("[Validation] Fallback: Too short (< 5 chars): %s\n", a)
		return false
	}

	// 挨拶・感謝などの雑談パターンは無効
	if containsGreeting(a) {
		fmt.Printf("[Validation] Fallback: Contains greeting: %s\n", a)
		return false
	}

	// 明らかな無回答パターンをチェック
	noAnswerPatterns := []string{
		"わからない", "分からない", "わかりません", "分かりません",
		"知らない", "知りません", "思いつかない", "思いつきません",
		"特にない", "特になし", "ありません", "ないです",
	}
	answerLower := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(a, " ", ""), "　", ""))
	for _, pattern := range noAnswerPatterns {
		if answerLower == pattern || answerLower == pattern+"。" {
			fmt.Printf("[Validation] Fallback: No-answer pattern detected: %s\n", a)
			return false
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

	// IT関連キーワードを含み、かつ10文字以上なら有効
	if hasITKeyword && len([]rune(a)) >= 10 {
		fmt.Printf("[Validation] Fallback: Has IT keyword and >= 10 chars: %s\n", a)
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

	// 共通キーワードが2つ以上あれば回答とみなす（厳格化）
	if common >= 2 {
		fmt.Printf("[Validation] Fallback: Common keywords >= 2: %s\n", a)
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
	// 現在進行中のフェーズを取得
	currentProgress, err := s.progressRepo.GetCurrentPhase(userID, sessionID)
	if err == nil {
		return currentProgress, nil
	}

	// 進行中のフェーズがない場合、次のフェーズを開始
	allPhases, err := s.phaseRepo.FindAll()
	if err != nil {
		return nil, err
	}

	// 既に完了したフェーズを確認
	completedProgresses, _ := s.progressRepo.FindByUserAndSession(userID, sessionID)
	completedMap := make(map[uint]bool)
	for _, p := range completedProgresses {
		if p.IsCompleted {
			completedMap[p.PhaseID] = true
		}
	}

	// 次の未完了フェーズを見つける
	for _, phase := range allPhases {
		if !completedMap[phase.ID] {
			// 新しいフェーズを開始
			return s.progressRepo.FindOrCreate(userID, sessionID, phase.ID)
		}
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

	// 完了スコアを計算（有効回答率 × 100）
	if progress.QuestionsAsked > 0 {
		progress.CompletionScore = (float64(progress.ValidAnswers) / float64(progress.QuestionsAsked)) * 100
	}

	// フェーズ完了条件をチェック
	// 最小質問数に達し、かつ完了スコアが70%以上、または最大質問数に達した場合
	if (progress.QuestionsAsked >= progress.Phase.MinQuestions && progress.CompletionScore >= 70) ||
		progress.QuestionsAsked >= progress.Phase.MaxQuestions {
		progress.IsCompleted = true
		now := new(time.Time)
		*now = time.Now()
		progress.CompletedAt = now
	}

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
			pp.QuestionsAsked = progress.QuestionsAsked
			pp.ValidAnswers = progress.ValidAnswers
			pp.CompletionScore = progress.CompletionScore
			pp.IsCompleted = progress.IsCompleted

			if !progress.IsCompleted && current == nil {
				current = &pp
			}
		}

		result = append(result, pp)
	}

	return result, current, nil
}
