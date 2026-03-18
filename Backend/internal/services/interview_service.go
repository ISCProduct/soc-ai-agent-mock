package services

import (
	"Backend/domain/repository"
	"Backend/internal/models"
	"Backend/internal/openai"
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
	sessionRepo  repository.InterviewSessionRepository
	utterRepo    repository.InterviewUtteranceRepository
	reportRepo   repository.InterviewReportRepository
	userRepo     repository.UserRepository
	emailService *EmailService
	openaiClient *openai.Client
	jobCh        chan uint
	workerOnce   sync.Once
}

func NewInterviewService(
	sessionRepo repository.InterviewSessionRepository,
	utterRepo repository.InterviewUtteranceRepository,
	reportRepo repository.InterviewReportRepository,
	userRepo repository.UserRepository,
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
	Language         string     `json:"language"`
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

func (s *InterviewService) CreateSession(userID uint, language string) (*InterviewSessionResponse, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}
	if language == "" {
		language = "ja"
	}
	session := &models.InterviewSession{
		UserID:          userID,
		Status:          "ready",
		Language:        language,
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

// ListAllSessionsAdmin lists all interview sessions without performing a user-level admin check.
// The caller (admin middleware) is responsible for ensuring only admins can invoke this.
func (s *InterviewService) ListAllSessionsAdmin(limit int, offset int) ([]InterviewSessionResponse, int64, error) {
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
	lang := session.Language
	if lang == "" {
		lang = "ja"
	}
	model := getEnv("OPENAI_REALTIME_MODEL", "gpt-4o-realtime-preview")
	voice := realtimeVoiceForLang(lang)
	transcribeModel := getEnv("OPENAI_REALTIME_TRANSCRIBE_MODEL", "gpt-4o-mini-transcribe")
	maxTokens := getIntEnv("OPENAI_REALTIME_MAX_OUTPUT_TOKENS", 120)
	req := openai.RealtimeSessionRequest{
		Model:        model,
		Modalities:   []string{"audio"},
		Voice:        voice,
		Instructions: buildRealtimeInstructions(lang),
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
	session, err := s.sessionRepo.FindByID(sessionID)
	if err != nil {
		return err
	}
	lang := session.Language
	if lang == "" {
		lang = "ja"
	}

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
		role := "User"
		if u.Role == "ai" {
			role = "Interviewer"
		}
		transcriptBuilder.WriteString(role)
		transcriptBuilder.WriteString(": ")
		transcriptBuilder.WriteString(strings.TrimSpace(u.Text))
		transcriptBuilder.WriteString("\n")
	}
	transcript := transcriptBuilder.String()
	systemPrompt := "あなたは就職面接の評価者です。面接ログを読んで、応募者の回答を客観的に評価し、JSONのみで返してください。"
	userPrompt := fmt.Sprintf(`以下の面接ログを読み、下記の評価基準に従ってJSONのみで出力してください。

## 評価基準（各スコアは0〜5の整数）
- logic（論理性）: 回答が筋道立っているか、主張に一貫性があるか
- specificity（具体性）: 具体的なエピソードや数値が含まれているか
- ownership（主体性）: 「私が〜した」という自分起点の表現があるか

## 出力フォーマット（このキーと型を厳守してください）
{
  "summary": ["評価コメント1", "評価コメント2", "評価コメント3"],
  "scores": {"logic": 3, "specificity": 2, "ownership": 4},
  "evidence": {"logic": "論理性の根拠となった発言", "specificity": "具体性の根拠となった発言", "ownership": "主体性の根拠となった発言"}
}

※ summaryは最大5件で日本語の簡潔な文章。scoresは実際の会話内容に基づいて正直に採点してください（全て同じ値は避ける）。

Interview transcript:
%s`, lang, transcript)

	model := getEnv("INTERVIEW_REPORT_MODEL", "")
	raw, err := s.openaiClient.ChatCompletionJSON(ctx, systemPrompt, userPrompt, 0.4, 1000, model)
	if err != nil {
		return err
	}
	type reportPayload struct {
		Summary  []string          `json:"summary"`
		Scores   map[string]int    `json:"scores"`
		Evidence map[string]string `json:"evidence"`
	}
	// markdown コードブロック除去（モデルによっては ```json ... ``` で包まれることがある）
	cleaned := strings.TrimSpace(raw)
	if idx := strings.Index(cleaned, "{"); idx > 0 {
		cleaned = cleaned[idx:]
	}
	if idx := strings.LastIndex(cleaned, "}"); idx >= 0 && idx < len(cleaned)-1 {
		cleaned = cleaned[:idx+1]
	}
	var payload reportPayload
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
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
	lang := session.Language
	if lang == "" {
		lang = "ja"
	}
	return &InterviewSessionResponse{
		ID:               session.ID,
		UserID:           session.UserID,
		Status:           session.Status,
		Language:         lang,
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

// buildReportSystemPrompt 言語コードに応じたレポート生成用システムプロンプトを返す。
func buildReportSystemPrompt(lang string) string {
	known := map[string]string{
		"ja": "あなたは就活面接のアシスタントです。面接ログを読み、要約・評価をJSONで返してください。",
		"en": "You are a job interview assessment assistant. Read the interview transcript and return evaluation as JSON.",
		"zh": "你是一位求职面试评估助手。请阅读面试记录并以JSON格式返回评估结果。",
		"ko": "당신은 취업 면접 평가 어시스턴트입니다. 면접 기록을 읽고 JSON 형식으로 평가를 반환하세요。",
		"fr": "Vous êtes un assistant d'évaluation d'entretien d'embauche. Lisez la transcription et retournez l'évaluation en JSON.",
		"es": "Eres un asistente de evaluación de entrevistas de trabajo. Lee la transcripción y devuelve la evaluación en JSON.",
		"de": "Sie sind ein Assistent zur Bewertung von Vorstellungsgesprächen. Lesen Sie das Transkript und geben Sie die Bewertung als JSON zurück.",
		"pt": "Você é um assistente de avaliação de entrevistas de emprego. Leia a transcrição e retorne a avaliação em JSON.",
		"it": "Sei un assistente per la valutazione dei colloqui di lavoro. Leggi la trascrizione e restituisci la valutazione in JSON.",
		"ar": "أنت مساعد تقييم مقابلات العمل. اقرأ النص وأعد التقييم بصيغة JSON.",
		"ru": "Вы ассистент по оценке собеседований. Прочитайте транскрипт и верните оценку в формате JSON.",
		"hi": "आप नौकरी साक्षात्कार मूल्यांकन सहायक हैं। साक्षात्कार का विवरण पढ़ें और मूल्यांकन JSON में लौटाएं।",
		"th": "คุณเป็นผู้ช่วยประเมินการสัมภาษณ์งาน อ่านบทสนทนาแล้วส่งคืนการประเมินในรูปแบบ JSON",
		"vi": "Bạn là trợ lý đánh giá phỏng vấn tuyển dụng. Đọc bản ghi và trả về đánh giá dưới dạng JSON.",
		"id": "Anda adalah asisten evaluasi wawancara kerja. Baca transkrip dan kembalikan evaluasi dalam format JSON.",
		"tr": "Siz bir iş görüşmesi değerlendirme asistanısınız. Metni okuyun ve değerlendirmeyi JSON formatında döndürün.",
	}
	if prompt, ok := known[lang]; ok {
		return prompt
	}
	return fmt.Sprintf("You are a job interview assessment assistant. Read the interview transcript and return evaluation as JSON. Use language code \"%s\" for the summary and evidence fields.", lang)
}

// realtimeVoiceForLang 言語コードに応じた推奨ボイスを返す。
// 環境変数 OPENAI_REALTIME_VOICE が設定されている場合はそちらを優先する。
func realtimeVoiceForLang(lang string) string {
	if v := getEnv("OPENAI_REALTIME_VOICE", ""); v != "" {
		return v
	}
	switch lang {
	case "ja":
		return "alloy"
	case "en":
		return "shimmer"
	case "zh":
		return "nova"
	case "ko":
		return "alloy"
	default:
		return "alloy"
	}
}

// buildRealtimeInstructions 面接官AIへのシステムプロンプトを返す。
// デフォルトは日本語で進行し、面接者から別言語を求められた場合は即座に切り替える。
func buildRealtimeInstructions(_ string) string {
	return strings.TrimSpace(`あなたはプロの就活面接官です。以下のルールに従ってください。

【言語対応】
- デフォルトは日本語で面接を行う
- 面接者から別の言語での面接を求められた場合（例：「英語でお願いします」「Please switch to English」「请用中文」など）は、即座にその言語に切り替えて面接を継続する
- 一度切り替えた言語は、面接者から変更を求められるまで維持する

【質問の意図が伝わらない場合】
- 面接者から「意味がわかりません」「質問の意図を教えてください」「もう少し詳しく教えてください」などの発言があった場合は、同じ質問を別の言い方で言い換えるか、具体的な例を添えて再度問いかける
- 言い換えても理解が難しそうな場合は「では少し視点を変えて〜」と切り出し、関連する別の質問に移る
- 面接者が質問に詰まっている場合は「焦らずに考えてみてください」と一言添えてから待つ

【面接の進め方】
- 1回の発話は短く、質問中心にする
- 詳細な講評・フィードバックは面接終了後に行う
- 面接者が話しやすいよう、具体的に深掘りする`)
}
