package services

import (
	"Backend/internal/models"
	"Backend/internal/repositories"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

type RealtimeDailySummary struct {
	Date                 string  `json:"date"`
	TotalCostUSD         float64 `json:"total_cost_usd"`
	TotalDurationSeconds int64   `json:"total_duration_seconds"`
	SessionCount         int64   `json:"session_count"`
	UserCount            int64   `json:"user_count"`
}

type RealtimeMonthlySummary struct {
	Month                string  `json:"month"`
	TotalCostUSD         float64 `json:"total_cost_usd"`
	TotalDurationSeconds int64   `json:"total_duration_seconds"`
	SessionCount         int64   `json:"session_count"`
	UserCount            int64   `json:"user_count"`
}

type RealtimeUserSummary struct {
	UserID               uint    `json:"user_id"`
	TotalCostUSD         float64 `json:"total_cost_usd"`
	TotalDurationSeconds int64   `json:"total_duration_seconds"`
	SessionCount         int64   `json:"session_count"`
}

type RealtimeUsageService struct {
	repo                    *repositories.RealtimeUsageRepository
	emailService            *EmailService
	alertThresholdUSD       float64
	maxConcurrent           int64
	ratePerMinuteUSD        float64
	sessionDurationMinutes  int

	mu               sync.Mutex
	lastAlertMonthID string
}

func NewRealtimeUsageService(repo *repositories.RealtimeUsageRepository, emailService *EmailService) *RealtimeUsageService {
	threshold := getFloatEnv("REALTIME_MONTHLY_ALERT_THRESHOLD_USD", 200.0)
	maxConcurrent := int64(getIntEnv("REALTIME_MAX_CONCURRENT_CONNECTIONS", 30))
	if maxConcurrent <= 0 {
		maxConcurrent = 30
	}
	rate := getFloatEnv("INTERVIEW_COST_PER_MIN_USD", 0.18)
	sessionMinutes := getIntEnv("INTERVIEW_SESSION_MINUTES", 10)
	if sessionMinutes <= 0 {
		sessionMinutes = 10
	}
	return &RealtimeUsageService{
		repo:                   repo,
		emailService:           emailService,
		alertThresholdUSD:      threshold,
		maxConcurrent:          maxConcurrent,
		ratePerMinuteUSD:       rate,
		sessionDurationMinutes: sessionMinutes,
	}
}

// SessionDurationMinutes はユーザー向けのセッション時間（分）を返す。
// コスト上限は内部管理のみとし、UXには時間として公開する。
func (s *RealtimeUsageService) SessionDurationMinutes() int {
	return s.sessionDurationMinutes
}

func (s *RealtimeUsageService) CanOpenNewConnection() (bool, int64, int64, error) {
	active, err := s.repo.CountActiveSessions()
	if err != nil {
		return false, 0, s.maxConcurrent, err
	}
	return active < s.maxConcurrent, active, s.maxConcurrent, nil
}

func (s *RealtimeUsageService) EnsureSessionStarted(userID, sessionID uint) error {
	if _, err := s.repo.FindActiveBySessionID(sessionID); err == nil {
		return nil
	}
	entry := &models.RealtimeUsageLog{
		UserID:             userID,
		InterviewSessionID: sessionID,
		Status:             "active",
		StartedAt:          time.Now().UTC(),
	}
	if err := s.repo.Create(entry); err != nil {
		return err
	}
	go s.checkAndNotifyThreshold()
	return nil
}

func (s *RealtimeUsageService) CloseSession(sessionID uint, endedAt time.Time) (durationSec int64, costUSD float64, err error) {
	entry, err := s.repo.FindActiveBySessionID(sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return -1, 0, nil
		}
		return 0, 0, err
	}
	dur := int64(endedAt.Sub(entry.StartedAt).Seconds())
	if dur < 0 {
		dur = 0
	}
	cost := (float64(dur) / 60.0) * s.ratePerMinuteUSD
	entry.EndedAt = &endedAt
	entry.DurationSeconds = int(dur)
	entry.CostUSD = cost
	entry.Status = "finished"
	if err := s.repo.Update(entry); err != nil {
		return 0, 0, err
	}
	go s.checkAndNotifyThreshold()
	return dur, cost, nil
}

func (s *RealtimeUsageService) CurrentMonthTotalCost() (float64, error) {
	return s.repo.CurrentMonthTotalCostEstimated(s.ratePerMinuteUSD)
}

func (s *RealtimeUsageService) CurrentActiveCount() (int64, error) {
	return s.repo.CountActiveSessions()
}

func (s *RealtimeUsageService) GetDailyUsage(nDays int) ([]RealtimeDailySummary, error) {
	rows, err := s.repo.DailyUsage(nDays, s.ratePerMinuteUSD)
	if err != nil {
		return nil, err
	}
	out := make([]RealtimeDailySummary, len(rows))
	for i, r := range rows {
		out[i] = RealtimeDailySummary{
			Date:                 r.Date,
			TotalCostUSD:         r.TotalCostUSD,
			TotalDurationSeconds: r.TotalDurationSeconds,
			SessionCount:         r.SessionCount,
			UserCount:            r.UserCount,
		}
	}
	return out, nil
}

func (s *RealtimeUsageService) GetMonthlyUsage(nMonths int) ([]RealtimeMonthlySummary, error) {
	rows, err := s.repo.MonthlyUsage(nMonths, s.ratePerMinuteUSD)
	if err != nil {
		return nil, err
	}
	out := make([]RealtimeMonthlySummary, len(rows))
	for i, r := range rows {
		out[i] = RealtimeMonthlySummary{
			Month:                r.Month,
			TotalCostUSD:         r.TotalCostUSD,
			TotalDurationSeconds: r.TotalDurationSeconds,
			SessionCount:         r.SessionCount,
			UserCount:            r.UserCount,
		}
	}
	return out, nil
}

func (s *RealtimeUsageService) GetUserBreakdown(days int, limit int) ([]RealtimeUserSummary, error) {
	if days <= 0 {
		days = 30
	}
	since := time.Now().UTC().AddDate(0, 0, -days)
	rows, err := s.repo.UserBreakdown(since, s.ratePerMinuteUSD, limit)
	if err != nil {
		return nil, err
	}
	out := make([]RealtimeUserSummary, len(rows))
	for i, r := range rows {
		out[i] = RealtimeUserSummary{
			UserID:               r.UserID,
			TotalCostUSD:         r.TotalCostUSD,
			TotalDurationSeconds: r.TotalDurationSeconds,
			SessionCount:         r.SessionCount,
		}
	}
	return out, nil
}

func (s *RealtimeUsageService) checkAndNotifyThreshold() {
	total, err := s.CurrentMonthTotalCost()
	if err != nil {
		return
	}
	if total < s.alertThresholdUSD {
		return
	}
	monthID := time.Now().UTC().Format("2006-01")
	s.mu.Lock()
	if s.lastAlertMonthID == monthID {
		s.mu.Unlock()
		return
	}
	s.lastAlertMonthID = monthID
	s.mu.Unlock()

	subject := fmt.Sprintf("[SOC AI] Realtime月額コスト閾値超過 (%s)", monthID)
	body := fmt.Sprintf(
		"Realtime API monthly cost exceeded threshold.\nmonth=%s\ntotal_usd=%.4f\nthreshold_usd=%.4f\nactive_limit=%d\n",
		monthID, total, s.alertThresholdUSD, s.maxConcurrent,
	)
	log.Printf("[RealtimeUsage] ALERT %s", strings.ReplaceAll(body, "\n", " "))

	if webhook := strings.TrimSpace(os.Getenv("REALTIME_ALERT_SLACK_WEBHOOK_URL")); webhook != "" {
		if err := postSlackAlert(webhook, subject+"\n"+body); err != nil {
			log.Printf("[RealtimeUsage] slack alert failed: %v", err)
		}
	}

	if s.emailService != nil {
		recipients := parseEmailsEnv("REALTIME_ALERT_EMAILS")
		if len(recipients) > 0 {
			if err := s.emailService.SendSystemAlertEmail(recipients, subject, body); err != nil {
				log.Printf("[RealtimeUsage] email alert failed: %v", err)
			}
		}
	}
}

func postSlackAlert(webhookURL, text string) error {
	payload := map[string]string{"text": text}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("status=%d", resp.StatusCode)
	}
	return nil
}

func parseEmailsEnv(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		addr := strings.TrimSpace(p)
		if addr != "" {
			out = append(out, addr)
		}
	}
	return out
}
