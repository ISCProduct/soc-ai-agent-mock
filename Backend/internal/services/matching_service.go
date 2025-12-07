package services

import (
	"Backend/internal/models"
	"Backend/internal/repositories"
	"context"
	"fmt"
	"math"
)

type MatchingService struct {
	userWeightScoreRepo *repositories.UserWeightScoreRepository
	companyRepo         *repositories.CompanyRepository
	matchRepo           *repositories.UserCompanyMatchRepository
}

func NewMatchingService(
	userWeightScoreRepo *repositories.UserWeightScoreRepository,
	companyRepo *repositories.CompanyRepository,
	matchRepo *repositories.UserCompanyMatchRepository,
) *MatchingService {
	return &MatchingService{
		userWeightScoreRepo: userWeightScoreRepo,
		companyRepo:         companyRepo,
		matchRepo:           matchRepo,
	}
}

// CalculateMatching ユーザーと企業のマッチングを計算（高速化のため無効化）
func (s *MatchingService) CalculateMatching(ctx context.Context, userID uint, sessionID string) error {
	// 事前計算済みデータを使用するため、実際の計算はスキップ
	fmt.Printf("[CalculateMatching] Skipped (using pre-calculated data) for user %d, session %s\n", userID, sessionID)
	return nil
}

// calculateMatchScore ユーザースコアと企業プロファイルからマッチングスコアを計算
func (s *MatchingService) calculateMatchScore(
	userScores map[string]float64,
	companyProfile *models.CompanyWeightProfile,
) *models.UserCompanyMatch {
	match := &models.UserCompanyMatch{}

	// 各カテゴリのマッチ度を計算（0-100のスケールで）
	// マッチ度 = 100 - |ユーザースコア - 企業重視度|
	match.TechnicalMatch = calculateCategoryMatch(userScores["技術志向"], float64(companyProfile.TechnicalOrientation))
	match.TeamworkMatch = calculateCategoryMatch(userScores["チームワーク志向"], float64(companyProfile.TeamworkOrientation))
	match.LeadershipMatch = calculateCategoryMatch(userScores["リーダーシップ志向"], float64(companyProfile.LeadershipOrientation))
	match.CreativityMatch = calculateCategoryMatch(userScores["創造性志向"], float64(companyProfile.CreativityOrientation))
	match.StabilityMatch = calculateCategoryMatch(userScores["安定志向"], float64(companyProfile.StabilityOrientation))
	match.GrowthMatch = calculateCategoryMatch(userScores["成長志向"], float64(companyProfile.GrowthOrientation))
	match.WorkLifeMatch = calculateCategoryMatch(userScores["ワークライフバランス"], float64(companyProfile.WorkLifeBalance))
	match.ChallengeMatch = calculateCategoryMatch(userScores["チャレンジ志向"], float64(companyProfile.ChallengeSeeking))
	match.DetailMatch = calculateCategoryMatch(userScores["細部志向"], float64(companyProfile.DetailOrientation))
	match.CommunicationMatch = calculateCategoryMatch(userScores["コミュニケーション力"], float64(companyProfile.CommunicationSkill))

	// 総合マッチ度を計算（全カテゴリの平均）
	match.MatchScore = (match.TechnicalMatch + match.TeamworkMatch + match.LeadershipMatch +
		match.CreativityMatch + match.StabilityMatch + match.GrowthMatch +
		match.WorkLifeMatch + match.ChallengeMatch + match.DetailMatch +
		match.CommunicationMatch) / 10.0

	return match
}

// calculateCategoryMatch カテゴリごとのマッチ度を計算
// ユーザースコアと企業重視度の差が小さいほど高スコア
func calculateCategoryMatch(userScore, companyWeight float64) float64 {
	diff := math.Abs(userScore - companyWeight)
	return math.Max(0, 100.0-diff)
}

// GetTopMatches マッチング度の高い企業を取得
func (s *MatchingService) GetTopMatches(ctx context.Context, userID uint, sessionID string, limit int) ([]*models.UserCompanyMatch, error) {
	return s.matchRepo.FindTopMatchesByUserAndSession(userID, sessionID, limit)
}

// GenerateMatchReason AIを使ってマッチング理由を生成（オプション）
func (s *MatchingService) GenerateMatchReason(ctx context.Context, match *models.UserCompanyMatch) (string, error) {
	// TODO: OpenAI APIを使って、ユーザーの適性と企業の特徴を基にマッチング理由を生成
	reason := fmt.Sprintf(
		"あなたの適性スコアと企業の求める人材像が%0.1f%%マッチしています。"+
			"特に、技術志向(%0.1f%%)とチームワーク志向(%0.1f%%)において高い親和性が見られます。",
		match.MatchScore,
		match.TechnicalMatch,
		match.TeamworkMatch,
	)
	return reason, nil
}
