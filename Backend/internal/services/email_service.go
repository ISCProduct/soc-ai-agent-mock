package services

import (
	"Backend/internal/models"
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"strconv"
	"time"
)

type EmailService struct {
	host     string
	port     int
	user     string
	password string
	from     string
}

func NewEmailService() *EmailService {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if port == 0 {
		port = 587
	}
	return &EmailService{
		host:     os.Getenv("SMTP_HOST"),
		port:     port,
		user:     os.Getenv("SMTP_USER"),
		password: os.Getenv("SMTP_PASSWORD"),
		from:     os.Getenv("SMTP_FROM"),
	}
}

type EmailReportCompany struct {
	Rank   int
	Name   string
	Score  int
	Reason string
}

type emailReportData struct {
	UserName         string
	SessionID        string
	SentAt           string
	JobScore         string
	InterestScore    string
	AptitudeScore    string
	FutureScore      string
	JobProgress      string
	InterestProgress string
	AptitudeProgress string
	FutureProgress   string
	Companies        []EmailReportCompany
}

const reportEmailTemplate = `<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="UTF-8">
  <title>AI就活分析レポート</title>
  <style>
    body{font-family:'Hiragino Sans','Meiryo',sans-serif;background:#f5f5f5;margin:0;padding:20px;}
    .container{max-width:600px;margin:0 auto;background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.1);}
    .header{background:linear-gradient(135deg,#1976D2,#42A5F5);color:white;padding:32px 24px;text-align:center;}
    .header h1{margin:0;font-size:22px;}
    .header p{margin:8px 0 0;opacity:.9;font-size:13px;}
    .section{padding:20px 24px;border-bottom:1px solid #e0e0e0;}
    .section h2{margin:0 0 14px;font-size:16px;color:#1976D2;}
    .scores-grid{display:grid;grid-template-columns:1fr 1fr;gap:10px;}
    .score-card{background:#f5f5f5;border-radius:8px;padding:12px;text-align:center;}
    .score-label{font-size:12px;color:#666;margin-bottom:4px;}
    .score-value{font-size:26px;font-weight:bold;color:#1976D2;}
    .phase-item{margin-bottom:10px;}
    .phase-header{display:flex;justify-content:space-between;font-size:13px;margin-bottom:4px;}
    .bar-bg{background:#e0e0e0;border-radius:4px;height:8px;}
    .bar-fill{background:#1976D2;border-radius:4px;height:8px;}
    .company-item{background:#f9f9f9;border-radius:8px;padding:14px;margin-bottom:10px;border-left:4px solid #1976D2;}
    .company-rank{font-size:11px;color:#888;}
    .company-name{font-size:15px;font-weight:bold;margin:3px 0;}
    .company-score{font-size:18px;font-weight:bold;color:#1976D2;}
    .company-reason{font-size:12px;color:#555;margin-top:6px;line-height:1.5;}
    .footer{padding:20px 24px;text-align:center;background:#fafafa;color:#999;font-size:11px;}
  </style>
</head>
<body>
<div class="container">
  <div class="header">
    <h1>🎉 AI就活分析レポート</h1>
    <p>{{.UserName}} さんの分析結果</p>
    <p>{{.SentAt}}</p>
  </div>

  <div class="section">
    <h2>📊 4分析スコア</h2>
    <div class="scores-grid">
      <div class="score-card">
        <div class="score-label">職種分析</div>
        <div class="score-value">{{.JobScore}}</div>
      </div>
      <div class="score-card">
        <div class="score-label">興味分析</div>
        <div class="score-value">{{.InterestScore}}</div>
      </div>
      <div class="score-card">
        <div class="score-label">適性分析</div>
        <div class="score-value">{{.AptitudeScore}}</div>
      </div>
      <div class="score-card">
        <div class="score-label">将来分析</div>
        <div class="score-value">{{.FutureScore}}</div>
      </div>
    </div>
  </div>

  <div class="section">
    <h2>📈 フェーズ進捗</h2>
    <div class="phase-item">
      <div class="phase-header"><span>職種分析</span><span>{{.JobProgress}}%</span></div>
      <div class="bar-bg"><div class="bar-fill" style="width:{{.JobProgress}}%"></div></div>
    </div>
    <div class="phase-item">
      <div class="phase-header"><span>興味分析</span><span>{{.InterestProgress}}%</span></div>
      <div class="bar-bg"><div class="bar-fill" style="width:{{.InterestProgress}}%"></div></div>
    </div>
    <div class="phase-item">
      <div class="phase-header"><span>適性分析</span><span>{{.AptitudeProgress}}%</span></div>
      <div class="bar-bg"><div class="bar-fill" style="width:{{.AptitudeProgress}}%"></div></div>
    </div>
    <div class="phase-item">
      <div class="phase-header"><span>将来分析</span><span>{{.FutureProgress}}%</span></div>
      <div class="bar-bg"><div class="bar-fill" style="width:{{.FutureProgress}}%"></div></div>
    </div>
  </div>

  {{if .Companies}}
  <div class="section">
    <h2>🏢 おすすめ企業</h2>
    {{range .Companies}}
    <div class="company-item">
      <div class="company-rank">第{{.Rank}}位</div>
      <div class="company-name">{{.Name}}</div>
      <div class="company-score">適合度 {{.Score}}%</div>
      <div class="company-reason">{{.Reason}}</div>
    </div>
    {{end}}
  </div>
  {{end}}

  <div class="footer">
    <p>このメールはAI就活エージェントから自動送信されました。</p>
    <p>セッションID: {{.SessionID}}</p>
  </div>
</div>
</body>
</html>`

func (s *EmailService) SendAnalysisReport(user *models.User, summary *AnalysisSummary, companies []EmailReportCompany, sessionID string) error {
	tmpl, err := template.New("report").Parse(reportEmailTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	pct := func(v float64) string { return fmt.Sprintf("%.0f", v*100) }

	data := emailReportData{
		UserName:         user.Name,
		SessionID:        sessionID,
		SentAt:           time.Now().Format("2006年01月02日 15:04"),
		JobScore:         fmt.Sprintf("%.0f", summary.Scores.JobScore),
		InterestScore:    fmt.Sprintf("%.0f", summary.Scores.InterestScore),
		AptitudeScore:    fmt.Sprintf("%.0f", summary.Scores.AptitudeScore),
		FutureScore:      fmt.Sprintf("%.0f", summary.Scores.FutureScore),
		JobProgress:      pct(summary.Progress.Job),
		InterestProgress: pct(summary.Progress.Interest),
		AptitudeProgress: pct(summary.Progress.Aptitude),
		FutureProgress:   pct(summary.Progress.Future),
		Companies:        companies,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	htmlBody := buf.String()

	// SMTP未設定の場合はログ出力のみ（開発環境向け）
	if s.host == "" {
		fmt.Printf("[EmailService] SMTP not configured. Simulating send to %s (body: %d bytes)\n", user.Email, len(htmlBody))
		return nil
	}

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: AI就活分析レポート\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.from, user.Email, htmlBody,
	)
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	auth := smtp.PlainAuth("", s.user, s.password, s.host)

	if err := smtp.SendMail(addr, auth, s.from, []string{user.Email}, []byte(msg)); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Printf("[EmailService] Email sent successfully to %s\n", user.Email)
	return nil
}

// InterviewReportEmailData 面接レポートメールのデータ
type InterviewReportEmailData struct {
	UserName    string
	SessionID   string
	SentAt      string
	Summary     []string
	LogicScore  int
	SpecScore   int
	OwnScore    int
	LogicEvid   string
	SpecEvid    string
	OwnEvid     string
}

const interviewReportEmailTemplate = `<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="UTF-8">
  <title>AI面接練習レポート</title>
  <style>
    body{font-family:'Hiragino Sans','Meiryo',sans-serif;background:#f5f5f5;margin:0;padding:20px;}
    .container{max-width:600px;margin:0 auto;background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.1);}
    .header{background:linear-gradient(135deg,#ec5b13,#ff8a50);color:white;padding:32px 24px;text-align:center;}
    .header h1{margin:0;font-size:22px;}
    .header p{margin:8px 0 0;opacity:.9;font-size:13px;}
    .section{padding:20px 24px;border-bottom:1px solid #e0e0e0;}
    .section h2{margin:0 0 14px;font-size:16px;color:#ec5b13;}
    .summary-item{display:flex;gap:8px;margin-bottom:8px;font-size:13px;line-height:1.6;color:#333;}
    .bullet{color:#ec5b13;font-weight:bold;flex-shrink:0;}
    .scores-grid{display:grid;grid-template-columns:1fr 1fr 1fr;gap:10px;margin-bottom:16px;}
    .score-card{background:#fff8f5;border:1px solid #ffd5c0;border-radius:8px;padding:12px;text-align:center;}
    .score-label{font-size:11px;color:#888;margin-bottom:4px;}
    .score-value{font-size:28px;font-weight:bold;color:#ec5b13;}
    .score-max{font-size:12px;color:#aaa;}
    .bar-bg{background:#e0e0e0;border-radius:4px;height:8px;margin-top:4px;}
    .bar-fill{background:#ec5b13;border-radius:4px;height:8px;}
    .evidence-item{background:#f9f9f9;border-radius:6px;padding:12px;margin-bottom:8px;}
    .evidence-label{font-size:11px;color:#ec5b13;font-weight:bold;margin-bottom:4px;}
    .evidence-text{font-size:13px;color:#555;line-height:1.5;}
    .footer{padding:20px 24px;text-align:center;background:#fafafa;color:#999;font-size:11px;}
  </style>
</head>
<body>
<div class="container">
  <div class="header">
    <h1>🎤 AI面接練習レポート</h1>
    <p>{{.UserName}} さんの面接結果</p>
    <p>{{.SentAt}}</p>
  </div>

  {{if .Summary}}
  <div class="section">
    <h2>📝 総合フィードバック</h2>
    {{range .Summary}}
    <div class="summary-item"><span class="bullet">▶</span><span>{{.}}</span></div>
    {{end}}
  </div>
  {{end}}

  <div class="section">
    <h2>📊 評価スコア（5点満点）</h2>
    <div class="scores-grid">
      <div class="score-card">
        <div class="score-label">論理性</div>
        <div class="score-value">{{.LogicScore}}</div>
        <div class="score-max">/ 5</div>
        <div class="bar-bg"><div class="bar-fill" style="width:{{.LogicScore}}0%"></div></div>
      </div>
      <div class="score-card">
        <div class="score-label">具体性</div>
        <div class="score-value">{{.SpecScore}}</div>
        <div class="score-max">/ 5</div>
        <div class="bar-bg"><div class="bar-fill" style="width:{{.SpecScore}}0%"></div></div>
      </div>
      <div class="score-card">
        <div class="score-label">主体性</div>
        <div class="score-value">{{.OwnScore}}</div>
        <div class="score-max">/ 5</div>
        <div class="bar-bg"><div class="bar-fill" style="width:{{.OwnScore}}0%"></div></div>
      </div>
    </div>
  </div>

  <div class="section">
    <h2>🔍 評価の根拠</h2>
    {{if .LogicEvid}}
    <div class="evidence-item">
      <div class="evidence-label">論理性</div>
      <div class="evidence-text">{{.LogicEvid}}</div>
    </div>
    {{end}}
    {{if .SpecEvid}}
    <div class="evidence-item">
      <div class="evidence-label">具体性</div>
      <div class="evidence-text">{{.SpecEvid}}</div>
    </div>
    {{end}}
    {{if .OwnEvid}}
    <div class="evidence-item">
      <div class="evidence-label">主体性</div>
      <div class="evidence-text">{{.OwnEvid}}</div>
    </div>
    {{end}}
  </div>

  <div class="footer">
    <p>このメールはAI就活エージェントから自動送信されました。</p>
    <p>セッションID: {{.SessionID}}</p>
  </div>
</div>
</body>
</html>`

// SendInterviewReport 面接練習レポートをメールで送信
func (s *EmailService) SendInterviewReport(user *models.User, data InterviewReportEmailData) error {
	tmpl, err := template.New("interview_report").Parse(interviewReportEmailTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	data.SentAt = time.Now().Format("2006年01月02日 15:04")
	data.UserName = user.Name

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	htmlBody := buf.String()

	if s.host == "" {
		fmt.Printf("[EmailService] SMTP not configured. Simulating interview report send to %s (body: %d bytes)\n", user.Email, len(htmlBody))
		return nil
	}

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: AI面接練習レポート\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.from, user.Email, htmlBody,
	)
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	auth := smtp.PlainAuth("", s.user, s.password, s.host)

	if err := smtp.SendMail(addr, auth, s.from, []string{user.Email}, []byte(msg)); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Printf("[EmailService] Interview report email sent successfully to %s\n", user.Email)
	return nil
}

const registrationEmailTemplate = `<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="UTF-8">
  <title>メールアドレス確認</title>
  <style>
    body{font-family:'Hiragino Sans','Meiryo',sans-serif;background:#f5f5f5;margin:0;padding:20px;}
    .container{max-width:600px;margin:0 auto;background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.1);}
    .header{background:linear-gradient(135deg,#1976D2,#42A5F5);color:white;padding:32px 24px;text-align:center;}
    .header h1{margin:0;font-size:22px;}
    .body{padding:24px;}
    .btn{display:inline-block;margin:20px 0;padding:14px 28px;background:#1976D2;color:white;text-decoration:none;border-radius:6px;font-size:15px;}
    .footer{padding:20px 24px;text-align:center;background:#fafafa;color:#999;font-size:11px;}
  </style>
</head>
<body>
<div class="container">
  <div class="header"><h1>メールアドレス確認</h1></div>
  <div class="body">
    <p>IT業界キャリアエージェントへのご登録ありがとうございます。</p>
    <p>以下のボタンをクリックしてメールアドレスを確認し、アカウント情報の入力に進んでください。</p>
    <a class="btn" href="{{.VerifyURL}}">メールアドレスを確認する</a>
    <p style="color:#888;font-size:12px;">このリンクは24時間有効です。身に覚えのない場合は無視してください。</p>
  </div>
  <div class="footer"><p>IT業界キャリアエージェント</p></div>
</div>
</body>
</html>`

// SendRegistrationEmail 仮登録確認メールを送信
func (s *EmailService) SendRegistrationEmail(email, token string) error {
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	verifyURL := fmt.Sprintf("%s/verify-registration?token=%s", frontendURL, token)

	type data struct{ VerifyURL string }
	tmpl, err := template.New("reg").Parse(registrationEmailTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data{VerifyURL: verifyURL}); err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}
	htmlBody := buf.String()

	if s.host == "" {
		fmt.Printf("[EmailService] SMTP not configured. Simulating registration email to %s (verify_url: %s)\n", email, verifyURL)
		return nil
	}

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: メールアドレスの確認\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.from, email, htmlBody,
	)
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	auth := smtp.PlainAuth("", s.user, s.password, s.host)
	if err := smtp.SendMail(addr, auth, s.from, []string{email}, []byte(msg)); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	fmt.Printf("[EmailService] Registration email sent to %s\n", email)
	return nil
}
