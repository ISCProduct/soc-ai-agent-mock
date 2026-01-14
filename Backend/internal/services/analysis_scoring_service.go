package services

import (
	"Backend/internal/models"
	"Backend/internal/repositories"
	"context"
	"encoding/json"
	"math"
	"strings"
)

type AnalysisScores struct {
	JobScore      float64 `json:"job_score"`
	InterestScore float64 `json:"interest_score"`
	AptitudeScore float64 `json:"aptitude_score"`
	FutureScore   float64 `json:"future_score"`
	FinalScore    float64 `json:"final_score"`
}

type AnalysisProgress struct {
	Job      float64 `json:"job"`
	Interest float64 `json:"interest"`
	Aptitude float64 `json:"aptitude"`
	Future   float64 `json:"future"`
	Overall  float64 `json:"overall"`
}

type AxisScore struct {
	Axis  string  `json:"axis"`
	Score float64 `json:"score"`
}

type CategoryRecommendation struct {
	Category string `json:"category"`
	Score    int    `json:"score"`
}

type CompanyRecommendation struct {
	ID    uint    `json:"id"`
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

type AnalysisRecommendations struct {
	TopCategories []CategoryRecommendation `json:"top_categories"`
	TopCompanies  []CompanyRecommendation  `json:"top_companies"`
}

type AnalysisSummary struct {
	Scores          AnalysisScores          `json:"scores"`
	Progress        AnalysisProgress        `json:"progress"`
	AptitudeAxes    []AxisScore             `json:"aptitude_axes"`
	FutureSignals   []string                `json:"future_signals,omitempty"`
	Recommendations AnalysisRecommendations `json:"recommendations"`
}

type FutureAnalyzer interface {
	Score(messages []models.ChatMessage) (float64, []string)
}

type RuleBasedFutureAnalyzer struct {
	keywords  []string
	threshold int
}

func NewRuleBasedFutureAnalyzer() *RuleBasedFutureAnalyzer {
	return &RuleBasedFutureAnalyzer{
		keywords: []string{
			"成長", "挑戦", "将来", "キャリア", "スキル", "学び", "伸ば", "向上",
			"リーダー", "マネジメント", "起業", "海外", "グローバル", "研究", "開発",
		},
		threshold: 5,
	}
}

func (a *RuleBasedFutureAnalyzer) Score(messages []models.ChatMessage) (float64, []string) {
	if len(messages) == 0 {
		return 0, nil
	}

	found := make(map[string]bool)
	for _, msg := range messages {
		if msg.Role != "user" {
			continue
		}
		text := strings.ToLower(msg.Content)
		for _, keyword := range a.keywords {
			if strings.Contains(text, strings.ToLower(keyword)) {
				found[keyword] = true
			}
		}
	}

	if len(found) == 0 {
		return 0, nil
	}

	matched := make([]string, 0, len(found))
	for keyword := range found {
		matched = append(matched, keyword)
	}

	score := math.Min(1, float64(len(found))/float64(a.threshold))
	return score, matched
}

type AnalysisScoringService struct {
	userWeightScoreRepo     *repositories.UserWeightScoreRepository
	chatMessageRepo         *repositories.ChatMessageRepository
	progressRepo            *repositories.UserAnalysisProgressRepository
	conversationContextRepo *repositories.ConversationContextRepository
	userEmbeddingRepo       *repositories.UserEmbeddingRepository
	jobEmbeddingRepo        *repositories.JobCategoryEmbeddingRepository
	matchRepo               *repositories.UserCompanyMatchRepository
	futureAnalyzer          FutureAnalyzer
}

func NewAnalysisScoringService(
	userWeightScoreRepo *repositories.UserWeightScoreRepository,
	chatMessageRepo *repositories.ChatMessageRepository,
	progressRepo *repositories.UserAnalysisProgressRepository,
	conversationContextRepo *repositories.ConversationContextRepository,
	userEmbeddingRepo *repositories.UserEmbeddingRepository,
	jobEmbeddingRepo *repositories.JobCategoryEmbeddingRepository,
	matchRepo *repositories.UserCompanyMatchRepository,
	futureAnalyzer FutureAnalyzer,
) *AnalysisScoringService {
	if futureAnalyzer == nil {
		futureAnalyzer = NewRuleBasedFutureAnalyzer()
	}
	return &AnalysisScoringService{
		userWeightScoreRepo:     userWeightScoreRepo,
		chatMessageRepo:         chatMessageRepo,
		progressRepo:            progressRepo,
		conversationContextRepo: conversationContextRepo,
		userEmbeddingRepo:       userEmbeddingRepo,
		jobEmbeddingRepo:        jobEmbeddingRepo,
		matchRepo:               matchRepo,
		futureAnalyzer:          futureAnalyzer,
	}
}

func (s *AnalysisScoringService) BuildAnalysisSummary(ctx context.Context, userID uint, sessionID string) (*AnalysisSummary, error) {
	_ = ctx
	jobScore, err := s.calculateJobScore(userID, sessionID)
	if err != nil {
		return nil, err
	}
	interestScore := s.calculateInterestScore(userID, sessionID)
	aptitudeScore, axes := s.calculateAptitudeScore(userID, sessionID)
	futureScore, signals := s.calculateFutureScore(sessionID)

	finalScore := (jobScore * 0.4) + (interestScore * 0.25) + (aptitudeScore * 0.2) + (futureScore * 0.15)

	progress := s.calculateProgress(userID, sessionID)
	recommendations := s.buildRecommendations(userID, sessionID)

	return &AnalysisSummary{
		Scores: AnalysisScores{
			JobScore:      jobScore,
			InterestScore: interestScore,
			AptitudeScore: aptitudeScore,
			FutureScore:   futureScore,
			FinalScore:    finalScore,
		},
		Progress:        progress,
		AptitudeAxes:    axes,
		FutureSignals:   signals,
		Recommendations: recommendations,
	}, nil
}

func (s *AnalysisScoringService) calculateJobScore(userID uint, sessionID string) (float64, error) {
	if s.userEmbeddingRepo == nil || s.jobEmbeddingRepo == nil || s.conversationContextRepo == nil {
		return 0, nil
	}

	jobCategoryID, err := s.conversationContextRepo.GetJobCategoryID(sessionID)
	if err != nil || jobCategoryID == 0 {
		return 0, nil
	}

	userEmbedding, err := s.userEmbeddingRepo.FindByUserAndSession(userID, sessionID)
	if err != nil {
		return 0, nil
	}
	jobEmbedding, err := s.jobEmbeddingRepo.FindByJobCategoryID(jobCategoryID)
	if err != nil {
		return 0, nil
	}

	userVector, err := parseEmbedding(userEmbedding.Embedding)
	if err != nil {
		return 0, err
	}
	jobVector, err := parseEmbedding(jobEmbedding.Embedding)
	if err != nil {
		return 0, err
	}

	return cosineSimilarity(userVector, jobVector), nil
}

func (s *AnalysisScoringService) calculateInterestScore(userID uint, sessionID string) float64 {
	if s.matchRepo == nil {
		return 0
	}

	stats, err := s.matchRepo.GetMatchStatistics(userID, sessionID)
	if err != nil {
		return 0
	}

	totalMatches, _ := stats["total_matches"].(int64)
	viewedCount, _ := stats["viewed_count"].(int64)
	favoritedCount, _ := stats["favorited_count"].(int64)
	appliedCount, _ := stats["applied_count"].(int64)

	raw := (float64(viewedCount) * 0.7) + (float64(appliedCount) * 1.0) + (float64(favoritedCount) * 1.2)
	max := float64(totalMatches) * (0.7 + 1.0 + 1.2)
	if max <= 0 {
		return 0
	}
	return clamp01(raw / max)
}

func (s *AnalysisScoringService) calculateAptitudeScore(userID uint, sessionID string) (float64, []AxisScore) {
	scores, err := s.userWeightScoreRepo.FindByUserAndSession(userID, sessionID)
	if err != nil {
		return 0, nil
	}

	scoreMap := make(map[string]float64, len(scores))
	for _, score := range scores {
		scoreMap[score.WeightCategory] = float64(score.Score)
	}

	axisCategories := map[string][]string{
		"論理性": {"技術志向", "細部志向"},
		"協調性": {"チームワーク志向", "コミュニケーション力"},
		"自律性": {"成長志向", "チャレンジ志向", "リーダーシップ志向"},
	}

	var axisScores []AxisScore
	var sum float64
	var count float64

	for axis, categories := range axisCategories {
		axisScore := averageCategoryScore(scoreMap, categories)
		axisScores = append(axisScores, AxisScore{
			Axis:  axis,
			Score: axisScore,
		})
		sum += axisScore
		count++
	}

	if count == 0 {
		return 0, axisScores
	}
	return clamp01(sum / count), axisScores
}

func (s *AnalysisScoringService) calculateFutureScore(sessionID string) (float64, []string) {
	if s.chatMessageRepo == nil || s.futureAnalyzer == nil {
		return 0, nil
	}
	messages, err := s.chatMessageRepo.FindBySessionID(sessionID)
	if err != nil {
		return 0, nil
	}
	score, signals := s.futureAnalyzer.Score(messages)
	return clamp01(score), signals
}

func (s *AnalysisScoringService) calculateProgress(userID uint, sessionID string) AnalysisProgress {
	progress := AnalysisProgress{}
	if s.progressRepo == nil {
		return progress
	}

	records, err := s.progressRepo.FindByUserAndSession(userID, sessionID)
	if err != nil {
		return progress
	}

	for _, record := range records {
		if record.Phase == nil {
			continue
		}
		score := clamp01(record.CompletionScore / 100.0)
		switch record.Phase.PhaseName {
		case "job_analysis":
			progress.Job = score
		case "interest_analysis":
			progress.Interest = score
		case "aptitude_analysis":
			progress.Aptitude = score
		case "future_analysis":
			progress.Future = score
		}
	}

	progress.Overall = clamp01((progress.Job + progress.Interest + progress.Aptitude + progress.Future) / 4.0)
	return progress
}

func (s *AnalysisScoringService) buildRecommendations(userID uint, sessionID string) AnalysisRecommendations {
	recommendations := AnalysisRecommendations{}

	topCategories, err := s.userWeightScoreRepo.FindTopCategories(userID, sessionID, 3)
	if err == nil {
		for _, score := range topCategories {
			recommendations.TopCategories = append(recommendations.TopCategories, CategoryRecommendation{
				Category: score.WeightCategory,
				Score:    score.Score,
			})
		}
	}

	if s.matchRepo == nil {
		return recommendations
	}

	topMatches, err := s.matchRepo.FindTopMatchesByUserAndSession(userID, sessionID, 3)
	if err != nil {
		return recommendations
	}

	for _, match := range topMatches {
		if match.Company.ID == 0 {
			continue
		}
		recommendations.TopCompanies = append(recommendations.TopCompanies, CompanyRecommendation{
			ID:    match.Company.ID,
			Name:  match.Company.Name,
			Score: match.MatchScore,
		})
	}
	return recommendations
}

func parseEmbedding(raw string) ([]float64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var vec []float64
	if err := json.Unmarshal([]byte(raw), &vec); err != nil {
		return nil, err
	}
	return vec, nil
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}
	return clamp01(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func averageCategoryScore(scoreMap map[string]float64, categories []string) float64 {
	if len(categories) == 0 {
		return 0
	}
	var sum float64
	var count float64
	for _, category := range categories {
		score, ok := scoreMap[category]
		if !ok {
			continue
		}
		sum += clamp01(score / 100.0)
		count++
	}
	if count == 0 {
		return 0
	}
	return clamp01(sum / count)
}
