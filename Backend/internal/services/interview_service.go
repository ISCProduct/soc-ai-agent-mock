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
	systemPrompt := buildReportSystemPrompt(lang)
	userPrompt := fmt.Sprintf(`Based on the following interview transcript, output JSON only.

Output format:
{
  "summary": ["point1", "point2", "point3"],
  "scores": {"logic": 0, "specificity": 0, "ownership": 0},
  "evidence": {"logic": "reason", "specificity": "reason", "ownership": "reason"}
}

Scores are integers from 0 to 5. Up to 5 summary points, keep them concise.
Write the summary and evidence in the same language as the interview (language code: %s).

Interview transcript:
%s`, lang, transcript)

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

// buildRealtimeInstructions 言語コードに応じたシステムプロンプトを返す。
// 既知の言語には専用プロンプトを、未知の言語にはフォールバックプロンプトを返す。
func buildRealtimeInstructions(lang string) string {
	known := map[string]string{
		"ja": `あなたは就活面接官です。以下を守ってください。
- 1回の発話は短く、質問中心にする
- 長い講評は面接終了後に行う
- ユーザーが話しやすいように具体的に深掘りする`,
		"en": `You are a professional job interview coach. Follow these rules:
- Keep each response brief and question-focused
- Save detailed feedback until the interview ends
- Ask concrete follow-up questions to draw out the candidate`,
		"zh": `你是一位专业的求职面试官。请遵守以下规则：
- 每次发言简短，以提问为主
- 详细评价留到面试结束后再给出
- 通过具体的追问引导候选人展开作答`,
		"ko": `당신은 전문 취업 면접관입니다. 다음 규칙을 준수하세요:
- 각 발화는 짧게, 질문 중심으로 진행하세요
- 상세한 평가는 면접 종료 후에 해주세요
- 구체적인 추가 질문으로 지원자의 답변을 이끌어내세요`,
		"fr": `Vous êtes un interviewer professionnel pour les candidatures à l'emploi. Suivez ces règles :
- Gardez chaque réponse brève et axée sur les questions
- Réservez les retours détaillés à la fin de l'entretien
- Posez des questions de suivi concrètes pour aider le candidat à développer ses réponses`,
		"es": `Eres un entrevistador profesional de empleo. Sigue estas reglas:
- Mantén cada intervención breve y centrada en preguntas
- Guarda los comentarios detallados para el final de la entrevista
- Haz preguntas de seguimiento concretas para que el candidato se explaye`,
		"de": `Sie sind ein professioneller Interviewer für Stellenbewerbungen. Befolgen Sie diese Regeln:
- Halten Sie jede Antwort kurz und fragezentriert
- Detailliertes Feedback geben Sie erst nach dem Interview
- Stellen Sie konkrete Nachfragen, um den Kandidaten zum Reden zu bringen`,
		"pt": `Você é um entrevistador profissional de emprego. Siga estas regras:
- Mantenha cada resposta breve e focada em perguntas
- Reserve o feedback detalhado para o final da entrevista
- Faça perguntas de acompanhamento concretas para estimular o candidato`,
		"it": `Sei un intervistatore professionale per candidature di lavoro. Segui queste regole:
- Mantieni ogni risposta breve e incentrata sulle domande
- Riserva il feedback dettagliato alla fine del colloquio
- Poni domande di approfondimento concrete per incoraggiare il candidato`,
		"ar": `أنت محاور مهني لوظائف العمل. اتبع هذه القواعد:
- اجعل كل رد موجزاً ومرتكزاً على الأسئلة
- احتفظ بالتغذية الراجعة التفصيلية حتى نهاية المقابلة
- اطرح أسئلة متابعة محددة لتشجيع المرشح على التعبير`,
		"ru": `Вы профессиональный интервьюер для трудоустройства. Следуйте этим правилам:
- Держите каждый ответ кратким и сосредоточенным на вопросах
- Оставьте подробную обратную связь на конец собеседования
- Задавайте конкретные уточняющие вопросы, чтобы раскрыть кандидата`,
		"hi": `आप एक पेशेवर नौकरी साक्षात्कारकर्ता हैं। इन नियमों का पालन करें:
- प्रत्येक उत्तर संक्षिप्त और प्रश्न-केंद्रित रखें
- विस्तृत प्रतिक्रिया साक्षात्कार के अंत तक सुरक्षित रखें
- उम्मीदवार को खुलकर बोलने के लिए ठोस अनुवर्ती प्रश्न पूछें`,
		"th": `คุณเป็นนักสัมภาษณ์งานมืออาชีพ ปฏิบัติตามกฎเหล่านี้:
- ตอบสั้นๆ และมุ่งเน้นที่คำถาม
- สงวนข้อเสนอแนะเชิงลึกไว้จนกว่าการสัมภาษณ์จะสิ้นสุด
- ถามคำถามติดตามที่เป็นรูปธรรมเพื่อดึงศักยภาพผู้สมัครออกมา`,
		"vi": `Bạn là người phỏng vấn tuyển dụng chuyên nghiệp. Tuân thủ các quy tắc sau:
- Giữ mỗi câu trả lời ngắn gọn và tập trung vào câu hỏi
- Để phản hồi chi tiết đến cuối buổi phỏng vấn
- Đặt câu hỏi bổ sung cụ thể để khai thác thông tin từ ứng viên`,
		"id": `Anda adalah pewawancara pekerjaan profesional. Ikuti aturan-aturan ini:
- Jaga setiap jawaban tetap singkat dan berfokus pada pertanyaan
- Simpan umpan balik terperinci hingga akhir wawancara
- Ajukan pertanyaan lanjutan yang konkret untuk menggali jawaban kandidat`,
		"tr": `Siz profesyonel bir iş görüşmecisisiniz. Bu kurallara uyun:
- Her yanıtı kısa ve soru odaklı tutun
- Ayrıntılı geri bildirimi görüşme sonuna saklayın
- Adayı açılmaya teşvik etmek için somut takip soruları sorun`,
	}
	if prompt, ok := known[lang]; ok {
		return strings.TrimSpace(prompt)
	}
	// フォールバック: 未知の言語コードはAIに言語指定のみ行う
	return strings.TrimSpace(fmt.Sprintf(
		"You are a professional job interview coach. Conduct this entire interview in the language with BCP 47 code \"%s\". Keep each response brief and question-focused. Save detailed feedback until the interview ends.",
		lang,
	))
}
