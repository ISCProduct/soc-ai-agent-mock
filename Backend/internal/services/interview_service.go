package services

import (
	"Backend/internal/models"
	"Backend/internal/openai"
	"Backend/internal/repositories"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

type InterviewService struct {
	sessionRepo  *repositories.InterviewSessionRepository
	utterRepo    *repositories.InterviewUtteranceRepository
	reportRepo   *repositories.InterviewReportRepository
	userRepo     *repositories.UserRepository
	emailService *EmailService
	openaiClient *openai.Client
	jobCh        chan uint
	workerOnce   sync.Once
}

func NewInterviewService(
	sessionRepo *repositories.InterviewSessionRepository,
	utterRepo *repositories.InterviewUtteranceRepository,
	reportRepo *repositories.InterviewReportRepository,
	userRepo *repositories.UserRepository,
	emailService *EmailService,
	openaiClient *openai.Client,
) *InterviewService {
	return &InterviewService{
		sessionRepo:  sessionRepo,
		utterRepo:    utterRepo,
		reportRepo:   reportRepo,
		userRepo:     userRepo,
		emailService: emailService,
		openaiClient: openaiClient,
		jobCh:        make(chan uint, 100),
	}
}

func (s *InterviewService) StartWorker() {
	s.workerOnce.Do(func() {
		go s.runWorker()
	})
}

func (s *InterviewService) runWorker() {
	for sessionID := range s.jobCh {
		if err := s.generateReport(context.Background(), sessionID); err != nil {
			fmt.Printf("[Interview] Report generation failed for session %d: %v\n", sessionID, err)
			continue
		}
		fmt.Printf("[Interview] Report generation completed for session %d\n", sessionID)
	}
}

type InterviewSessionResponse struct {
	ID               uint       `json:"id"`
	UserID           uint       `json:"user_id"`
	Status           string     `json:"status"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	EndedAt          *time.Time `json:"ended_at,omitempty"`
	EstimatedCostUSD float64    `json:"estimated_cost_usd"`
	TemplateVersion  string     `json:"template_version"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type InterviewDetailResponse struct {
	Session    InterviewSessionResponse    `json:"session"`
	Utterances []models.InterviewUtterance `json:"utterances"`
	Report     *models.InterviewReport     `json:"report,omitempty"`
}

func (s *InterviewService) CreateSession(userID uint) (*InterviewSessionResponse, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}
	session := &models.InterviewSession{
		UserID:          userID,
		Status:          "ready",
		TemplateVersion: getEnv("INTERVIEW_TEMPLATE_VERSION", "v1"),
	}
	if err := s.sessionRepo.Create(session); err != nil {
		return nil, err
	}
	return toSessionResponse(session), nil
}

func (s *InterviewService) StartSession(userID uint, sessionID uint) (*InterviewSessionResponse, error) {
	session, err := s.sessionRepo.FindByID(sessionID)
	if err != nil {
		return nil, err
	}
	if !s.isAllowed(userID, session.UserID) {
		return nil, errors.New("forbidden")
	}
	if session.Status == "finished" {
		return nil, errors.New("session already finished")
	}
	if session.StartedAt == nil {
		now := time.Now()
		session.StartedAt = &now
	}
	session.Status = "in_progress"
	if err := s.sessionRepo.Update(session); err != nil {
		return nil, err
	}
	return toSessionResponse(session), nil
}

func (s *InterviewService) FinishSession(userID uint, sessionID uint) (*InterviewSessionResponse, error) {
	session, err := s.sessionRepo.FindByID(sessionID)
	if err != nil {
		return nil, err
	}
	if !s.isAllowed(userID, session.UserID) {
		return nil, errors.New("forbidden")
	}
	if session.EndedAt == nil {
		now := time.Now()
		session.EndedAt = &now
	}
	session.Status = "finished"
	session.EstimatedCostUSD = s.estimateCost(session.StartedAt, session.EndedAt)
	if err := s.sessionRepo.Update(session); err != nil {
		return nil, err
	}
	s.jobCh <- sessionID
	return toSessionResponse(session), nil
}

func (s *InterviewService) SaveUtterance(userID uint, sessionID uint, role string, text string) error {
	session, err := s.sessionRepo.FindByID(sessionID)
	if err != nil {
		return err
	}
	if !s.isAllowed(userID, session.UserID) {
		return errors.New("forbidden")
	}
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "user" && role != "ai" {
		return errors.New("invalid role")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("empty text")
	}
	utter := &models.InterviewUtterance{
		SessionID: sessionID,
		Role:      role,
		Text:      text,
	}
	return s.utterRepo.Create(utter)
}

func (s *InterviewService) ListSessions(userID uint, all bool, limit int, offset int) ([]InterviewSessionResponse, int64, error) {
	if all {
		user, err := s.userRepo.GetUserByID(userID)
		if err != nil || user == nil || !user.IsAdmin {
			return nil, 0, errors.New("forbidden")
		}
		total, err := s.sessionRepo.CountAll()
		if err != nil {
			return nil, 0, err
		}
		sessions, err := s.sessionRepo.ListAll(limit, offset)
		if err != nil {
			return nil, 0, err
		}
		return toSessionResponses(sessions), total, nil
	}
	total, err := s.sessionRepo.CountByUser(userID)
	if err != nil {
		return nil, 0, err
	}
	sessions, err := s.sessionRepo.ListByUser(userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return toSessionResponses(sessions), total, nil
}

func (s *InterviewService) GetSessionDetail(userID uint, sessionID uint) (*InterviewDetailResponse, error) {
	session, err := s.sessionRepo.FindByID(sessionID)
	if err != nil {
		return nil, err
	}
	if !s.isAllowed(userID, session.UserID) {
		return nil, errors.New("forbidden")
	}
	utterances, err := s.utterRepo.FindBySessionID(sessionID)
	if err != nil {
		return nil, err
	}
	var report *models.InterviewReport
	report, err = s.reportRepo.FindBySessionID(sessionID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		report = nil
	}
	return &InterviewDetailResponse{
		Session:    *toSessionResponse(session),
		Utterances: utterances,
		Report:     report,
	}, nil
}

func (s *InterviewService) CreateRealtimeToken(ctx context.Context, userID uint, sessionID uint) (string, error) {
	session, err := s.sessionRepo.FindByID(sessionID)
	if err != nil {
		return "", err
	}
	if !s.isAllowed(userID, session.UserID) {
		return "", errors.New("forbidden")
	}
	if session.Status == "finished" {
		return "", errors.New("session already finished")
	}
	model := getEnv("OPENAI_REALTIME_MODEL", "gpt-4o-realtime-preview")
	voice := getEnv("OPENAI_REALTIME_VOICE", "alloy")
	transcribeModel := getEnv("OPENAI_REALTIME_TRANSCRIBE_MODEL", "gpt-4o-mini-transcribe")
	maxTokens := getIntEnv("OPENAI_REALTIME_MAX_OUTPUT_TOKENS", 120)
	req := openai.RealtimeSessionRequest{
		Model:        model,
		Modalities:   []string{"audio"},
		Voice:        voice,
		Instructions: buildRealtimeInstructions(),
		InputAudioTranscription: map[string]interface{}{
			"model": transcribeModel,
		},
		TurnDetection: map[string]interface{}{
			"type":                "server_vad",
			"threshold":           0.5,
			"silence_duration_ms": 700,
			"prefix_padding_ms":   300,
			"create_response":     true,
		},
		MaxResponseOutputTokens: maxTokens,
	}
	resp, err := s.openaiClient.CreateRealtimeClientSecret(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.ClientSecret.Value, nil
}

func (s *InterviewService) estimateCost(start, end *time.Time) float64 {
	if start == nil || end == nil {
		return 0
	}
	minutes := end.Sub(*start).Minutes()
	if minutes < 0 {
		return 0
	}
	rate := getFloatEnv("INTERVIEW_COST_PER_MIN_USD", 0.18)
	return minutes * rate
}

// TurnResult は1ターンの結果（AIテキスト + TTS音声バイト列）
type TurnResult struct {
	UserText string
	AIText   string
	Audio    []byte
}

// Turn はユーザー音声を受け取り、STT→Chat→TTSを実行してTurnResultを返します
func (s *InterviewService) Turn(ctx context.Context, userID uint, sessionID uint, audioData []byte, history []map[string]string, companyName, position, companyInfo string) (*TurnResult, error) {
	session, err := s.sessionRepo.FindByID(sessionID)
	if err != nil {
		return nil, err
	}
	if !s.isAllowed(userID, session.UserID) {
		return nil, errors.New("forbidden")
	}

	// STT: Whisper でユーザー音声をテキスト化
	userText, err := s.openaiClient.Transcribe(ctx, audioData, "audio.webm")
	if err != nil {
		return nil, fmt.Errorf("transcribe error: %w", err)
	}
	if strings.TrimSpace(userText) == "" {
		userText = "（聞き取れませんでした）"
	}

	// 履歴にユーザー発言を追加
	history = append(history, map[string]string{"role": "user", "content": userText})

	// Chat: 面接官として返答生成
	aiText, err := s.openaiClient.ChatInterview(ctx, buildInterviewSystemPrompt(companyName, position, companyInfo), history)
	if err != nil {
		return nil, fmt.Errorf("chat error: %w", err)
	}

	// TTS: AI返答を音声化
	voice := getEnv("OPENAI_TTS_VOICE", "alloy")
	audio, err := s.openaiClient.TTS(ctx, aiText, voice)
	if err != nil {
		return nil, fmt.Errorf("tts error: %w", err)
	}

	return &TurnResult{UserText: userText, AIText: aiText, Audio: audio}, nil
}

// StartTurn は面接開始の最初のAI発話を生成します
func (s *InterviewService) StartTurn(ctx context.Context, userID uint, sessionID uint, companyName, position, companyInfo string) (*TurnResult, error) {
	session, err := s.sessionRepo.FindByID(sessionID)
	if err != nil {
		return nil, err
	}
	if !s.isAllowed(userID, session.UserID) {
		return nil, errors.New("forbidden")
	}

	aiText, err := s.openaiClient.ChatInterview(ctx, buildInterviewSystemPrompt(companyName, position, companyInfo), []map[string]string{
		{"role": "user", "content": "面接を開始してください。最初の自己紹介・志望動機の質問からお願いします。"},
	})
	if err != nil {
		return nil, fmt.Errorf("chat error: %w", err)
	}

	voice := getEnv("OPENAI_TTS_VOICE", "alloy")
	audio, err := s.openaiClient.TTS(ctx, aiText, voice)
	if err != nil {
		return nil, fmt.Errorf("tts error: %w", err)
	}

	return &TurnResult{AIText: aiText, Audio: audio}, nil
}

func buildInterviewSystemPrompt(companyName, position, companyInfo string) string {
	base := `あなたは日本語の就活面接官です。以下を守ってください。
- 1回の返答は2〜3文以内で短くまとめる
- 必ず1つの質問で締めくくる
- 応募者が話しやすいよう具体的に深掘りする
- 評価・講評は面接終了まで行わない`

	if companyName != "" || position != "" {
		base += "\n\n【面接情報】"
		if companyName != "" {
			base += "\n志望企業: " + companyName
		}
		if position != "" {
			base += "\n応募職種: " + position
		}
		if companyInfo != "" {
			base += "\n企業概要: " + companyInfo
		}
		base += "\n\n上記の企業・職種に合わせた質問を行ってください。"
	}
	return strings.TrimSpace(base)
}

func (s *InterviewService) isAllowed(actorID uint, ownerID uint) bool {
	if actorID == ownerID {
		return true
	}
	user, err := s.userRepo.GetUserByID(actorID)
	if err != nil || user == nil {
		return false
	}
	return user.IsAdmin
}

func (s *InterviewService) generateReport(ctx context.Context, sessionID uint) error {
	utterances, err := s.utterRepo.FindBySessionID(sessionID)
	if err != nil {
		return err
	}
	if len(utterances) == 0 {
		// utterances が0件の場合は空レポートを保存して正常終了
		empty := &models.InterviewReport{
			SessionID:    sessionID,
			SummaryText:  "発話データがありませんでした。",
			ScoresJSON:   `{"logic":0,"specificity":0,"ownership":0}`,
			EvidenceJSON: `{}`,
		}
		return s.reportRepo.Upsert(empty)
	}
	var transcriptBuilder strings.Builder
	for _, u := range utterances {
		role := "ユーザー"
		if u.Role == "ai" {
			role = "面接官"
		}
		transcriptBuilder.WriteString(role)
		transcriptBuilder.WriteString(": ")
		transcriptBuilder.WriteString(strings.TrimSpace(u.Text))
		transcriptBuilder.WriteString("\n")
	}
	transcript := transcriptBuilder.String()
	systemPrompt := "あなたは就活面接のアシスタントです。面接ログを読み、短い日本語で要約・評価をJSONで返してください。"
	userPrompt := fmt.Sprintf(`以下の面接ログに基づき、JSONのみで出力してください。

出力フォーマット:
{
  "summary": ["箇条書き1", "箇条書き2", "箇条書き3"],
  "scores": {"logic": 0, "specificity": 0, "ownership": 0},
  "evidence": {"logic": "根拠", "specificity": "根拠", "ownership": "根拠"}
}

スコアは0〜5の整数。summaryは最大5件で簡潔に。

面接ログ:
%s`, transcript)

	model := getEnv("INTERVIEW_REPORT_MODEL", "")
	raw, err := s.openaiClient.ChatCompletionJSON(ctx, systemPrompt, userPrompt, 0.4, 400, model)
	if err != nil {
		return err
	}
	type reportPayload struct {
		Summary  []string          `json:"summary"`
		Scores   map[string]int    `json:"scores"`
		Evidence map[string]string `json:"evidence"`
	}
	var payload reportPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return fmt.Errorf("invalid report json: %w", err)
	}
	summaryText := strings.Join(payload.Summary, "\n")
	scoresJSON, _ := json.Marshal(payload.Scores)
	evidenceJSON, _ := json.Marshal(payload.Evidence)

	report := &models.InterviewReport{
		SessionID:    sessionID,
		SummaryText:  summaryText,
		ScoresJSON:   string(scoresJSON),
		EvidenceJSON: string(evidenceJSON),
	}
	return s.reportRepo.Upsert(report)
}

// SendReportEmail 面接レポートをメールで送信
func (s *InterviewService) SendReportEmail(userID, sessionID uint) error {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil || user == nil {
		return errors.New("user not found")
	}
	if user.IsGuest {
		return errors.New("guest users cannot receive email reports")
	}

	report, err := s.reportRepo.FindBySessionID(sessionID)
	if err != nil {
		return errors.New("report not found")
	}

	var scores map[string]int
	json.Unmarshal([]byte(report.ScoresJSON), &scores)
	var evidence map[string]string
	json.Unmarshal([]byte(report.EvidenceJSON), &evidence)

	summary := strings.Split(strings.TrimSpace(report.SummaryText), "\n")
	var filtered []string
	for _, line := range summary {
		if strings.TrimSpace(line) != "" {
			filtered = append(filtered, strings.TrimSpace(line))
		}
	}

	data := InterviewReportEmailData{
		SessionID:  fmt.Sprintf("%d", sessionID),
		Summary:    filtered,
		LogicScore: scores["logic"],
		SpecScore:  scores["specificity"],
		OwnScore:   scores["ownership"],
		LogicEvid:  evidence["logic"],
		SpecEvid:   evidence["specificity"],
		OwnEvid:    evidence["ownership"],
	}
	return s.emailService.SendInterviewReport(user, data)
}

func toSessionResponse(session *models.InterviewSession) *InterviewSessionResponse {
	return &InterviewSessionResponse{
		ID:               session.ID,
		UserID:           session.UserID,
		Status:           session.Status,
		StartedAt:        session.StartedAt,
		EndedAt:          session.EndedAt,
		EstimatedCostUSD: session.EstimatedCostUSD,
		TemplateVersion:  session.TemplateVersion,
		CreatedAt:        session.CreatedAt,
		UpdatedAt:        session.UpdatedAt,
	}
}

func toSessionResponses(sessions []models.InterviewSession) []InterviewSessionResponse {
	out := make([]InterviewSessionResponse, 0, len(sessions))
	for i := range sessions {
		out = append(out, *toSessionResponse(&sessions[i]))
	}
	return out
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func getIntEnv(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getFloatEnv(key string, def float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return n
}

func buildRealtimeInstructions() string {
	return strings.TrimSpace(`あなたは就活面接官です。以下を守ってください。
- 1回の発話は短く、質問中心にする
- 長い講評は面接終了後に行う
- ユーザーが話しやすいように具体的に深掘りする
`)
}
