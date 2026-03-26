package repositories

import (
	"Backend/internal/models"
	"encoding/json"

	"gorm.io/gorm"
)

type CollectiveInsightRepository struct {
	db *gorm.DB
}

func NewCollectiveInsightRepository(db *gorm.DB) *CollectiveInsightRepository {
	return &CollectiveInsightRepository{db: db}
}

// LogAction 行動ログを保存する（同意済みユーザーのみ）
func (r *CollectiveInsightRepository) LogAction(log *models.CollectiveInsightLog) error {
	return r.db.Create(log).Error
}

// IsConsentGiven ユーザーが集合知参加に同意しているか確認する
func (r *CollectiveInsightRepository) IsConsentGiven(userID uint) (bool, error) {
	var user models.User
	err := r.db.Select("allow_collective_insight").First(&user, userID).Error
	if err != nil {
		return false, err
	}
	return user.AllowCollectiveInsight, nil
}

// UpdateConsent ユーザーの同意設定を更新する
func (r *CollectiveInsightRepository) UpdateConsent(userID uint, allow bool) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("allow_collective_insight", allow).Error
}

// SimilarUserPassedCompaniesRow 類似ユーザーの通過企業行
type SimilarUserPassedCompaniesRow struct {
	CompanyID       uint
	CompanyName     string
	PassCount       int
	AvgPasserScores string // JSON
	CollectiveScore float64
}

// GetPassedCompaniesBySimilarUsers 類似スコアプロファイルのユーザーが通過した企業を返す
// similarUserHashes: 類似ユーザーの匿名ID一覧
func (r *CollectiveInsightRepository) GetPassedCompaniesBySimilarUsers(
	similarUserHashes []string,
	excludeCompanyIDs []uint,
) ([]SimilarUserPassedCompaniesRow, error) {
	if len(similarUserHashes) == 0 {
		return nil, nil
	}

	rows := []SimilarUserPassedCompaniesRow{}
	query := r.db.Raw(`
		SELECT
			cil.company_id,
			c.name AS company_name,
			COUNT(*) AS pass_count,
			'' AS avg_passer_scores,
			CAST(COUNT(*) AS FLOAT) / ? AS collective_score
		FROM collective_insight_logs cil
		INNER JOIN companies c ON c.id = cil.company_id
		WHERE cil.anonymous_user_id IN ?
		  AND cil.action_type IN ('passed', 'applied')
		GROUP BY cil.company_id, c.name
		ORDER BY pass_count DESC
		LIMIT 20
	`, len(similarUserHashes), similarUserHashes)

	if len(excludeCompanyIDs) > 0 {
		query = r.db.Raw(`
			SELECT
				cil.company_id,
				c.name AS company_name,
				COUNT(*) AS pass_count,
				'' AS avg_passer_scores,
				CAST(COUNT(*) AS FLOAT) / ? AS collective_score
			FROM collective_insight_logs cil
			INNER JOIN companies c ON c.id = cil.company_id
			WHERE cil.anonymous_user_id IN ?
			  AND cil.action_type IN ('passed', 'applied')
			  AND cil.company_id NOT IN ?
			GROUP BY cil.company_id, c.name
			ORDER BY pass_count DESC
			LIMIT 20
		`, len(similarUserHashes), similarUserHashes, excludeCompanyIDs)
	}

	err := query.Scan(&rows).Error
	return rows, err
}

// GetUserScoreSnapshots 類似ユーザー検索のためにスコアスナップショットを取得する
// action_type = 'applied' のログからスコア分布を集める
func (r *CollectiveInsightRepository) GetUserScoreSnapshots(limit int) ([]models.CollectiveInsightLog, error) {
	var logs []models.CollectiveInsightLog
	err := r.db.Where("action_type = ? AND score_snapshot != ''", "applied").
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// UpsertBehaviorSummary 企業別行動サマリーを更新する
func (r *CollectiveInsightRepository) UpsertBehaviorSummary(summary *models.AnonymizedBehaviorSummary) error {
	return r.db.Save(summary).Error
}

// GetBehaviorSummary 企業の行動サマリーを取得する
func (r *CollectiveInsightRepository) GetBehaviorSummary(companyID uint) (*models.AnonymizedBehaviorSummary, error) {
	var s models.AnonymizedBehaviorSummary
	err := r.db.Where("company_id = ?", companyID).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetTopPassRateCompanies 通過率上位の企業サマリーを返す
func (r *CollectiveInsightRepository) GetTopPassRateCompanies(limit int) ([]models.AnonymizedBehaviorSummary, error) {
	var summaries []models.AnonymizedBehaviorSummary
	err := r.db.Where("apply_count >= 3").
		Order("pass_rate DESC").
		Limit(limit).
		Find(&summaries).Error
	return summaries, err
}

// GetAllBehaviorSummaries バッチ集計用: 全企業サマリーを取得
func (r *CollectiveInsightRepository) GetAllBehaviorSummaries() ([]models.AnonymizedBehaviorSummary, error) {
	var summaries []models.AnonymizedBehaviorSummary
	err := r.db.Find(&summaries).Error
	return summaries, err
}

// RebuildSummaries 全企業の行動サマリーを再集計する（バッチ用）
func (r *CollectiveInsightRepository) RebuildSummaries() error {
	type rawSummary struct {
		CompanyID   uint
		ViewCount   int
		ApplyCount  int
		PassCount   int
	}

	var raws []rawSummary
	err := r.db.Raw(`
		SELECT
			company_id,
			SUM(CASE WHEN action_type = 'viewed' THEN 1 ELSE 0 END) AS view_count,
			SUM(CASE WHEN action_type = 'applied' THEN 1 ELSE 0 END) AS apply_count,
			SUM(CASE WHEN action_type = 'passed' THEN 1 ELSE 0 END) AS pass_count
		FROM collective_insight_logs
		GROUP BY company_id
	`).Scan(&raws).Error
	if err != nil {
		return err
	}

	for _, raw := range raws {
		passRate := 0.0
		if raw.ApplyCount > 0 {
			passRate = float64(raw.PassCount) / float64(raw.ApplyCount) * 100
		}

		// 通過ユーザーの平均スコアを計算
		avgScores, _ := r.calcAvgPasserScores(raw.CompanyID)
		avgJSON, _ := json.Marshal(avgScores)

		summary := &models.AnonymizedBehaviorSummary{
			CompanyID:       raw.CompanyID,
			ViewCount:       raw.ViewCount,
			ApplyCount:      raw.ApplyCount,
			PassCount:       raw.PassCount,
			PassRate:        passRate,
			AvgPasserScores: string(avgJSON),
		}

		r.db.Where("company_id = ?", raw.CompanyID).
			Assign(summary).
			FirstOrCreate(summary)
	}
	return nil
}

// calcAvgPasserScores 通過ユーザーのカテゴリ別平均スコアを計算する
func (r *CollectiveInsightRepository) calcAvgPasserScores(companyID uint) (map[string]float64, error) {
	var logs []models.CollectiveInsightLog
	err := r.db.Where("company_id = ? AND action_type = 'passed' AND score_snapshot != ''", companyID).
		Find(&logs).Error
	if err != nil || len(logs) == 0 {
		return nil, err
	}

	sums := map[string]float64{}
	counts := map[string]int{}
	for _, log := range logs {
		var scores map[string]float64
		if err := json.Unmarshal([]byte(log.ScoreSnapshot), &scores); err != nil {
			continue
		}
		for cat, score := range scores {
			sums[cat] += score
			counts[cat]++
		}
	}

	result := map[string]float64{}
	for cat, sum := range sums {
		result[cat] = sum / float64(counts[cat])
	}
	return result, nil
}
