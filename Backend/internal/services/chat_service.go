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
	aiClient            *openai.Client
	questionWeightRepo  *repositories.QuestionWeightRepository
	chatMessageRepo     *repositories.ChatMessageRepository
	userWeightScoreRepo *repositories.UserWeightScoreRepository
}

func NewChatService(
	aiClient *openai.Client,
	questionWeightRepo *repositories.QuestionWeightRepository,
	chatMessageRepo *repositories.ChatMessageRepository,
	userWeightScoreRepo *repositories.UserWeightScoreRepository,
) *ChatService {
	return &ChatService{
		aiClient:            aiClient,
		questionWeightRepo:  questionWeightRepo,
		chatMessageRepo:     chatMessageRepo,
		userWeightScoreRepo: userWeightScoreRepo,
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
	Response         string                   `json:"response"`
	QuestionWeightID uint                     `json:"question_weight_id,omitempty"`
	CurrentScores    []models.UserWeightScore `json:"current_scores,omitempty"`
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

	// 4. 次の質問をデータベースから取得
	nextQuestion, err := s.questionWeightRepo.GetRandomQuestion(req.IndustryID, req.JobCategoryID)
	var questionWeightID uint
	var aiResponse string

	if err != nil {
		// データベースに質問がない場合、AIに生成させる
		fmt.Printf("No question found in DB, generating with AI: %v\n", err)
		aiResponse, err = s.generateQuestionWithAI(ctx, history, req.IndustryID, req.JobCategoryID)
		if err != nil {
			return nil, fmt.Errorf("failed to generate question: %w", err)
		}
	} else {
		aiResponse = nextQuestion.Question
		questionWeightID = nextQuestion.ID
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
	scores, err := s.userWeightScoreRepo.FindByUserAndSession(req.UserID, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scores: %w", err)
	}

	return &ChatResponse{
		Response:         aiResponse,
		QuestionWeightID: questionWeightID,
		CurrentScores:    scores,
	}, nil
}

// analyzeAndUpdateWeights ユーザーの回答を分析し重み係数を更新
func (s *ChatService) analyzeAndUpdateWeights(ctx context.Context, userID uint, sessionID, message string) error {
	// AIに回答を分析させる
	prompt := fmt.Sprintf(`以下のユーザーの回答を分析し、就活適性のカテゴリごとにスコアを付けてください。
スコアは-10から+10の範囲で、そのカテゴリに適性があれば正の値、適性が低ければ負の値を返してください。

カテゴリ例: 技術志向, コミュニケーション, リーダーシップ, 創造性, 分析思考, チームワーク

ユーザーの回答: %s

JSON形式で返してください。例:
{
  "技術志向": 8,
  "コミュニケーション": 5,
  "リーダーシップ": 3
}`, message)

	response, err := s.aiClient.Responses(ctx, prompt)
	if err != nil {
		return err
	}

	// JSONパース
	var scores map[string]int
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

	// スコアを更新
	for category, score := range scores {
		if err := s.userWeightScoreRepo.UpdateScore(userID, sessionID, category, score); err != nil {
			return fmt.Errorf("failed to update score for %s: %w", category, err)
		}
	}

	return nil
}

// generateQuestionWithAI AIで質問を生成
func (s *ChatService) generateQuestionWithAI(ctx context.Context, history []models.ChatMessage, industryID, jobCategoryID uint) (string, error) {
	// 会話履歴を構築
	historyText := ""
	for _, msg := range history {
		historyText += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	prompt := fmt.Sprintf(`あなたは就活適性診断のためのインタビュアーです。
これまでの会話:
%s

次の質問を生成してください。質問は以下の点を考慮してください:
- ユーザーの適性（技術志向、コミュニケーション、リーダーシップなど）を判定できる内容
- 自然な会話の流れ
- 業界ID: %d, 職種ID: %d に関連する内容

質問のみを返してください。`, historyText, industryID, jobCategoryID)

	return s.aiClient.Responses(ctx, prompt)
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
