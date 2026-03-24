package services

import (
	"Backend/internal/models"
	"Backend/internal/repositories"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// モデル別単価テーブル (USD per 1M tokens)
var modelPricing = map[string][2]float64{
	// input, output per 1M tokens
	"gpt-4o":              {2.50, 10.00},
	"gpt-4o-mini":         {0.15, 0.60},
	"gpt-4-turbo":         {10.00, 30.00},
	"gpt-4":               {30.00, 60.00},
	"gpt-3.5-turbo":       {0.50, 1.50},
	"gpt-5.2":             {2.50, 10.00},  // treated as gpt-4o class
	"o1":                  {15.00, 60.00},
	"o1-mini":             {3.00, 12.00},
	"o3":                  {10.00, 40.00},
	"text-embedding-3-small": {0.02, 0.02},
	"text-embedding-3-large": {0.13, 0.13},
}

// calculateCost は入力/出力トークン数とモデル名からUSDコストを計算する
func calculateCost(model string, promptTokens, completionTokens int) float64 {
	lower := strings.ToLower(strings.TrimSpace(model))

	// prefix match for versioned model names (e.g. gpt-4o-2024-08-06)
	var pricing [2]float64
	var found bool
	for k, v := range modelPricing {
		if lower == k || strings.HasPrefix(lower, k+"-") || strings.HasPrefix(lower, k+":") {
			pricing = v
			found = true
			break
		}
	}
	if !found {
		// Default to gpt-4o pricing
		pricing = [2]float64{2.50, 10.00}
	}

	inputCost := float64(promptTokens) * pricing[0] / 1_000_000
	outputCost := float64(completionTokens) * pricing[1] / 1_000_000
	return inputCost + outputCost
}

// APICostService はAPIコスト記録・集計を担当する
type APICostService struct {
	repo            *repositories.APICallLogRepository
	alertThresholdUSD float64 // 月額閾値
}

func NewAPICostService(repo *repositories.APICallLogRepository) *APICostService {
	threshold := 100.0
	if v := os.Getenv("API_COST_ALERT_THRESHOLD_USD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			threshold = f
		}
	}
	return &APICostService{repo: repo, alertThresholdUSD: threshold}
}

// LogCall は非同期でAPIコールログをDBに記録する
func (s *APICostService) LogCall(model string, promptTokens, completionTokens int) {
	go func() {
		cost := calculateCost(model, promptTokens, completionTokens)
		entry := &models.APICallLog{
			Model:            model,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
			CostUSD:          cost,
			CalledAt:         time.Now().UTC(),
		}
		if err := s.repo.Create(entry); err != nil {
			log.Printf("[APICost] failed to log: %v", err)
		}
		s.checkThreshold()
	}()
}

// checkThreshold は月額閾値を超えていたらログ警告を出す
func (s *APICostService) checkThreshold() {
	since := time.Now().UTC().AddDate(0, -1, 0)
	total, err := s.repo.TotalCostSince(since)
	if err != nil {
		return
	}
	if total > s.alertThresholdUSD {
		log.Printf("[APICost] ALERT: Monthly API cost $%.4f exceeds threshold $%.2f", total, s.alertThresholdUSD)
	}
}

type DailyCostSummary struct {
	Date         string  `json:"date"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	TotalTokens  int64   `json:"total_tokens"`
	CallCount    int64   `json:"call_count"`
}

type MonthlyCostSummary struct {
	Month        string  `json:"month"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	TotalTokens  int64   `json:"total_tokens"`
	CallCount    int64   `json:"call_count"`
}

type ModelCostSummary struct {
	Model        string  `json:"model"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	TotalTokens  int64   `json:"total_tokens"`
	CallCount    int64   `json:"call_count"`
}

func (s *APICostService) GetDailyCosts(nDays int) ([]DailyCostSummary, error) {
	rows, err := s.repo.DailyCosts(nDays)
	if err != nil {
		return nil, err
	}
	result := make([]DailyCostSummary, len(rows))
	for i, r := range rows {
		result[i] = DailyCostSummary{
			Date:         r.Date,
			TotalCostUSD: r.TotalCostUSD,
			TotalTokens:  r.TotalTokens,
			CallCount:    r.CallCount,
		}
	}
	return result, nil
}

func (s *APICostService) GetMonthlyCosts(nMonths int) ([]MonthlyCostSummary, error) {
	rows, err := s.repo.MonthlyCosts(nMonths)
	if err != nil {
		return nil, err
	}
	result := make([]MonthlyCostSummary, len(rows))
	for i, r := range rows {
		result[i] = MonthlyCostSummary{
			Month:        r.Month,
			TotalCostUSD: r.TotalCostUSD,
			TotalTokens:  r.TotalTokens,
			CallCount:    r.CallCount,
		}
	}
	return result, nil
}

func (s *APICostService) GetModelBreakdown(since time.Time) ([]ModelCostSummary, error) {
	rows, err := s.repo.ModelBreakdown(since)
	if err != nil {
		return nil, err
	}
	result := make([]ModelCostSummary, len(rows))
	for i, r := range rows {
		result[i] = ModelCostSummary{
			Model:        r.Model,
			TotalCostUSD: r.TotalCostUSD,
			TotalTokens:  r.TotalTokens,
			CallCount:    r.CallCount,
		}
	}
	return result, nil
}

func (s *APICostService) GetCurrentMonthTotal() (float64, error) {
	now := time.Now().UTC()
	since := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return s.repo.TotalCostSince(since)
}
