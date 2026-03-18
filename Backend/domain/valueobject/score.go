package valueobject

import "fmt"

// AnalysisScore 就活適性診断の総合スコアを表す値オブジェクト
type AnalysisScore struct {
	JobScore      float64 // 職種適性スコア (0-1)
	InterestScore float64 // 興味・意欲スコア (0-1)
	AptitudeScore float64 // 多面的適性スコア (0-1)
	FutureScore   float64 // 将来志向スコア (0-1)
	FinalScore    float64 // 総合スコア (0-1)
}

// NewAnalysisScore 重み付き総合スコアを計算して生成する
func NewAnalysisScore(job, interest, aptitude, future float64) AnalysisScore {
	final := (job * 0.4) + (interest * 0.25) + (aptitude * 0.2) + (future * 0.15)
	return AnalysisScore{
		JobScore:      job,
		InterestScore: interest,
		AptitudeScore: aptitude,
		FutureScore:   future,
		FinalScore:    final,
	}
}

// FinalPercent 総合スコアをパーセント文字列で返す
func (s AnalysisScore) FinalPercent() string {
	return fmt.Sprintf("%.1f%%", s.FinalScore*100)
}

// IsHighPerformer 総合スコアが高い（80%以上）かどうか
func (s AnalysisScore) IsHighPerformer() bool {
	return s.FinalScore >= 0.8
}

// IsBalanced バランスの取れたスコア（全項目が60%以上）かどうか
func (s AnalysisScore) IsBalanced() bool {
	return s.JobScore >= 0.6 &&
		s.InterestScore >= 0.6 &&
		s.AptitudeScore >= 0.6 &&
		s.FutureScore >= 0.6
}
