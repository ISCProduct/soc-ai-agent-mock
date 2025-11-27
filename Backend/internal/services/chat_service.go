package services

import (
	"Backend/internal/models"
	"Backend/internal/openai"
	"Backend/internal/repositories"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type ChatService struct {
	aiClient                *openai.Client
	questionWeightRepo      *repositories.QuestionWeightRepository
	chatMessageRepo         *repositories.ChatMessageRepository
	userWeightScoreRepo     *repositories.UserWeightScoreRepository
	aiGeneratedQuestionRepo *repositories.AIGeneratedQuestionRepository
}

func NewChatService(
	aiClient *openai.Client,
	questionWeightRepo *repositories.QuestionWeightRepository,
	chatMessageRepo *repositories.ChatMessageRepository,
	userWeightScoreRepo *repositories.UserWeightScoreRepository,
	aiGeneratedQuestionRepo *repositories.AIGeneratedQuestionRepository,
) *ChatService {
	return &ChatService{
		aiClient:                aiClient,
		questionWeightRepo:      questionWeightRepo,
		chatMessageRepo:         chatMessageRepo,
		userWeightScoreRepo:     userWeightScoreRepo,
		aiGeneratedQuestionRepo: aiGeneratedQuestionRepo,
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
	Response          string                   `json:"response"`
	QuestionWeightID  uint                     `json:"question_weight_id,omitempty"`
	CurrentScores     []models.UserWeightScore `json:"current_scores,omitempty"`
	IsComplete        bool                     `json:"is_complete"`
	TotalQuestions    int                      `json:"total_questions"`
	AnsweredQuestions int                      `json:"answered_questions"`
}

// ProcessChat チャット処理のメインロジック
func (s *ChatService) ProcessChat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
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

	// 2. 会話履歴を取得（最新5件）
	history, err := s.chatMessageRepo.FindRecentBySessionID(req.SessionID, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat history: %w", err)
	}

	// 3. ユーザーの回答から重み係数を判定・更新
	if err := s.analyzeAndUpdateWeights(ctx, req.UserID, req.SessionID, req.Message); err != nil {
		// ログに記録するが、処理は継続
		fmt.Printf("Warning: failed to update weights: %v\n", err)
	}

	// 4. 既に聞いた質問のIDと質問文を取得
	askedQuestions, err := s.aiGeneratedQuestionRepo.FindByUserAndSession(req.UserID, req.SessionID)
	if err != nil {
		fmt.Printf("Warning: failed to get asked questions: %v\n", err)
		askedQuestions = []models.AIGeneratedQuestion{}
	}

	askedIDs := []uint{}
	askedTexts := make(map[string]bool)
	for _, q := range askedQuestions {
		if q.TemplateID > 0 {
			askedIDs = append(askedIDs, q.TemplateID)
		}
		// 質問文も記録（AIが生成した質問との重複防止）
		askedTexts[q.QuestionText] = true
	}

	// 5. 現在のスコアを分析して、次に評価すべきカテゴリを決定
	scores, err := s.userWeightScoreRepo.FindByUserAndSession(req.UserID, req.SessionID)
	if err != nil {
		fmt.Printf("Warning: failed to get scores for question selection: %v\n", err)
	}

	// スコア分布を分析
	scoreMap := make(map[string]int)
	for _, score := range scores {
		scoreMap[score.WeightCategory] = score.Score
	}

	// 全カテゴリ
	allCategories := []string{
		"技術志向", "コミュニケーション能力", "リーダーシップ", "チームワーク",
		"問題解決力", "創造性・発想力", "計画性・実行力", "学習意欲・成長志向",
		"ストレス耐性・粘り強さ", "ビジネス思考・目標志向",
	}

	// 未評価または評価が浅いカテゴリを優先
	var targetCategory string
	minScore := 1000
	for _, cat := range allCategories {
		score, exists := scoreMap[cat]
		if !exists {
			targetCategory = cat
			break
		}
		// 評価が浅いカテゴリを見つける
		if score < minScore && score > -5 && score < 5 {
			minScore = score
			targetCategory = cat
		}
	}

	// まだ評価が不十分なカテゴリがあれば、そのカテゴリの質問を優先
	var nextQuestion *models.QuestionWeight
	var err2 error
	if targetCategory != "" {
		// 特定カテゴリの質問を取得（既出を除外）
		nextQuestion, err2 = s.questionWeightRepo.GetRandomQuestionByCategory(targetCategory, askedIDs)
		if err2 != nil {
			fmt.Printf("No question found for category %s, falling back to general selection\n", targetCategory)
		}
	}

	// カテゴリ指定で見つからなければ、通常の選択
	if nextQuestion == nil {
		nextQuestion, err2 = s.questionWeightRepo.GetRandomQuestionExcluding(req.IndustryID, req.JobCategoryID, askedIDs)
	}

	var questionWeightID uint
	var aiResponse string

	if err2 != nil || nextQuestion == nil {
		// データベースに質問がない場合、AIに戦略的に生成させる
		fmt.Printf("No question found in DB, generating strategic question with AI\n")
		aiResponse, _, err = s.generateStrategicQuestion(ctx, history, req.UserID, req.SessionID, scoreMap, allCategories, askedTexts, req.IndustryID, req.JobCategoryID)
		if err != nil {
			return nil, fmt.Errorf("failed to generate question: %w", err)
		}
	} else {
		// 同じ質問文が既に出ていないか最終確認
		if askedTexts[nextQuestion.Question] {
			fmt.Printf("Question already asked (text match), generating new question\n")
			aiResponse, _, err = s.generateStrategicQuestion(ctx, history, req.UserID, req.SessionID, scoreMap, allCategories, askedTexts, req.IndustryID, req.JobCategoryID)
			if err != nil {
				return nil, fmt.Errorf("failed to generate question: %w", err)
			}
		} else {
			aiResponse = nextQuestion.Question
			questionWeightID = nextQuestion.ID

			// AI生成質問テーブルに記録
			aiGenQuestion := &models.AIGeneratedQuestion{
				UserID:       req.UserID,
				SessionID:    req.SessionID,
				TemplateID:   nextQuestion.ID,
				QuestionText: nextQuestion.Question,
				Weight:       nextQuestion.WeightValue,
				IsAnswered:   false,
			}
			if err := s.aiGeneratedQuestionRepo.Create(aiGenQuestion); err != nil {
				fmt.Printf("Warning: failed to save AI generated question: %v\n", err)
			}
		}
	}

	// 5. AIの応答を保存
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

	// 6. 現在のスコアを取得
	finalScores, err := s.userWeightScoreRepo.FindByUserAndSession(req.UserID, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scores: %w", err)
	}

	// 7. 質問数をカウントして終了判定
	answeredQuestions, err := s.aiGeneratedQuestionRepo.FindByUserAndSession(req.UserID, req.SessionID)
	if err != nil {
		fmt.Printf("Warning: failed to count answered questions: %v\n", err)
	}

	totalQuestions := 15 // 各カテゴリから最低1問ずつ + 深掘り質問
	answeredCount := len(answeredQuestions)
	isComplete := answeredCount >= totalQuestions

	return &ChatResponse{
		Response:          aiResponse,
		QuestionWeightID:  questionWeightID,
		CurrentScores:     finalScores,
		IsComplete:        isComplete,
		TotalQuestions:    totalQuestions,
		AnsweredQuestions: answeredCount,
	}, nil
}

// analyzeAndUpdateWeights ユーザーの回答を分析し重み係数を更新
func (s *ChatService) analyzeAndUpdateWeights(ctx context.Context, userID uint, sessionID, message string) error {
	// 「わからない」などの回答パターンを検出
	lowConfidencePatterns := []string{
		"わからない", "分からない", "わかりません", "分かりません",
		"よくわからない", "よく分からない", "不明", "知らない",
		"特にない", "ない", "思いつかない", "特に無い", "ありません",
	}

	isLowConfidence := false
	messageLower := strings.ToLower(message)
	for _, pattern := range lowConfidencePatterns {
		if strings.Contains(messageLower, pattern) {
			isLowConfidence = true
			break
		}
	}

	// わからない回答の場合は、スキップ
	if isLowConfidence {
		fmt.Printf("Low confidence answer detected, skipping analysis\n")
		return nil
	}

	// 会話履歴を取得して文脈を理解
	history, err := s.chatMessageRepo.FindRecentBySessionID(sessionID, 10)
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

	// より詳細な分析プロンプト
	prompt := fmt.Sprintf(`あなたは就職活動の適性診断と企業マッチングの専門家です。以下の会話履歴とユーザーの最新回答を総合的に分析し、
**企業選定に直結する**詳細なスコアリングを行ってください。

## 会話履歴
%s

## 最新の回答
%s

## 評価カテゴリと企業選定への影響

### 1. 技術志向 (-10〜+10)
**評価基準:**
- プログラミングや技術への興味・経験
- 技術的な課題への取り組み方
- 新しい技術への学習意欲
- 技術的な深掘りや探求心

**企業選定への影響:**
- +7以上: 技術主導企業、R&D部門、スタートアップ
- +3〜+6: バランス型企業、開発部門
- -3〜+2: ビジネス寄り企業、サポート部門
- -3以下: 非技術職、営業・企画

### 2. コミュニケーション能力 (-10〜+10)
**評価基準:**
- 説明の分かりやすさ・論理性
- 他者との対話・協調姿勢
- 自分の考えを伝える能力
- 傾聴力や相手を理解する姿勢

**企業選定への影響:**
- +7以上: コンサル、営業、PM職
- +3〜+6: チーム開発重視企業
- -3〜+2: 個人開発メイン企業
- -3以下: 研究職、単独作業

### 3. リーダーシップ (-10〜+10)
**評価基準:**
- チームを率いた経験
- 主体的な意思決定
- 目標設定や計画立案能力
- メンバーをサポート・動機づける力

**企業選定への影響:**
- +7以上: マネジメント志向、リーダー候補
- +3〜+6: チームリード志向
- -3〜+2: メンバー志向
- -3以下: サポート役、スペシャリスト

### 4. チームワーク (-10〜+10)
**評価基準:**
- 協力して作業する姿勢
- 役割分担への理解
- メンバーとの協調性
- チームの目標達成への貢献

**企業選定への影響:**
- +7以上: 大規模チーム企業、協調重視文化
- +3〜+6: 中規模チーム企業
- -3〜+2: 少数精鋭企業
- -3以下: 個人裁量大企業、フリーランス向き

### 5. 問題解決力 (-10〜+10)
**評価基準:**
- 論理的思考力
- 課題の分析・構造化能力
- 複雑な問題への取り組み方
- 解決策の創出と実行力

**企業選定への影響:**
- +7以上: コンサル、戦略系、難易度高プロジェクト
- +3〜+6: 開発・エンジニアリング
- -3〜+2: 運用・保守
- -3以下: 定型業務中心

### 6. 創造性・発想力 (-10〜+10)
**評価基準:**
- 独創的なアイデアの提案
- 既存の枠にとらわれない思考
- 新しいアプローチへの挑戦
- デザイン思考やイノベーション志向

**企業選定への影響:**
- +7以上: スタートアップ、新規事業、R&D
- +3〜+6: 自社サービス開発
- -3〜+2: 受託開発
- -3以下: 既存システム保守

### 7. 計画性・実行力 (-10〜+10)
**評価基準:**
- 目標設定と計画立案
- タスク管理能力
- スケジュール遵守
- 着実な実行と完遂力

**企業選定への影響:**
- +7以上: プロジェクト型企業、SIer
- +3〜+6: 開発プロジェクト
- -3〜+2: アジャイル・柔軟な環境
- -3以下: 探索的・研究開発

### 8. 学習意欲・成長志向 (-10〜+10)
**評価基準:**
- 継続的な学習姿勢
- フィードバックの受容
- 失敗からの学び
- キャリア成長への意識

**企業選定への影響:**
- +7以上: 急成長企業、スタートアップ、教育重視企業
- +3〜+6: 成長機会ある企業
- -3〜+2: 安定企業
- -3以下: ルーチン業務中心

### 9. ストレス耐性・粘り強さ (-10〜+10)
**評価基準:**
- 困難な状況での対処
- プレッシャー下でのパフォーマンス
- あきらめずに取り組む姿勢
- 柔軟な対応力

**企業選定への影響:**
- +7以上: 高負荷環境、ベンチャー、成果主義
- +3〜+6: 通常の開発環境
- -3〜+2: ワークライフバランス重視
- -3以下: 低ストレス環境、安定志向

### 10. ビジネス思考・目標志向 (-10〜+10)
**評価基準:**
- ビジネス価値への理解
- 成果・目標への意識
- 戦略的思考
- 顧客志向

**企業選定への影響:**
- +7以上: 事業会社、プロダクト企業、コンサル
- +3〜+6: 自社サービス開発
- -3〜+2: 受託開発
- -3以下: 技術特化、研究開発

## 重要な注意事項
1. **企業選定に役立つ評価**: 各スコアが具体的な企業タイプ・職種に結びつくように評価
2. **根拠の明確化**: スコアの理由を具体的に記述
3. **総合的判断**: 単一の回答だけでなく、会話全体から判断
4. **判断材料がない場合は0**: 無理に推測せず、情報不足なら0点

## 出力形式
JSON形式で、各カテゴリのスコアと**企業選定に関連する理由**を返してください。

{
  "技術志向": {"score": 8, "reason": "独学でプログラミングを学び、新技術への探求心が強い → 技術主導企業向き"},
  "コミュニケーション能力": {"score": 6, "reason": "論理的な説明ができ、チームでの対話を重視 → チーム開発企業向き"},
  "リーダーシップ": {"score": 0, "reason": "リーダー経験に関する情報なし"}
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
		return fmt.Errorf("invalid JSON response from AI")
	}
	jsonStr := response[jsonStart : jsonEnd+1]

	if err := json.Unmarshal([]byte(jsonStr), &scores); err != nil {
		return fmt.Errorf("failed to parse AI response: %w", err)
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
func (s *ChatService) generateStrategicQuestion(ctx context.Context, history []models.ChatMessage, userID uint, sessionID string, scoreMap map[string]int, allCategories []string, askedTexts map[string]bool, industryID, jobCategoryID uint) (string, uint, error) {
	// 会話履歴を構築
	historyText := ""
	for _, msg := range history {
		historyText += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	// 既に聞いた質問のリスト
	askedQuestionsText := "\n## 既に聞いた質問（これらと重複しないこと）\n"
	for text := range askedTexts {
		askedQuestionsText += fmt.Sprintf("- %s\n", text)
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

	prompt := fmt.Sprintf(`あなたは就職活動の適性診断と企業マッチングの専門家です。
これまでの会話と評価状況を分析し、**企業選定に直結する戦略的な質問**を1つ生成してください。

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

## 注意事項
- 質問のみを返す（説明や前置きは不要）
- 1つの質問で複数の観点を評価できるように工夫
- 答えやすく、かつ企業選定に役立つ情報が得られる内容
- 業界ID: %d, 職種ID: %d を考慮

**質問のみ**を返してください。`,
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

	// 重複チェック（念のため）
	if askedTexts[questionText] {
		// 非常に稀だが、AIが同じ質問を生成した場合は少し変更を加える
		fmt.Printf("Warning: AI generated duplicate question, modifying\n")
		questionText = questionText + "（あなたの経験から具体的に教えてください）"
	}

	// AI生成質問をデータベースに保存
	aiGenQuestion := &models.AIGeneratedQuestion{
		UserID:       userID,
		SessionID:    sessionID,
		TemplateID:   0, // AI生成の場合は0
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
		TemplateID:   0, // AI生成の場合は0
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
