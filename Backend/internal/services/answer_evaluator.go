package services

import (
	"Backend/internal/models"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// AnswerEvaluator 回答の評価サービス
type AnswerEvaluator struct{}

func NewAnswerEvaluator() *AnswerEvaluator {
	return &AnswerEvaluator{}
}

// EvaluationResult 評価結果
type EvaluationResult struct {
	Score           int      `json:"score"`
	Confidence      string   `json:"confidence"` // "high", "medium", "low"
	MatchedKeywords []string `json:"matched_keywords"`
	AppliedRules    []string `json:"applied_rules"`
	NeedsFollowUp   bool     `json:"needs_follow_up"`
	FollowUpTrigger string   `json:"follow_up_trigger"`
	Explanation     string   `json:"explanation"`
}

// Evaluate ルールベースで回答を評価
func (e *AnswerEvaluator) Evaluate(question *models.PredefinedQuestion, answer string) (*EvaluationResult, error) {
	result := &EvaluationResult{
		Score:           0,
		MatchedKeywords: []string{},
		AppliedRules:    []string{},
	}

	answerLower := strings.ToLower(answer)
	answerLength := len([]rune(answer))

	// 1. 回答の長さチェック（基本的な信頼性判定）
	if answerLength < 10 {
		result.Score -= 3
		result.Confidence = "low"
		result.NeedsFollowUp = true
		result.FollowUpTrigger = "too_short"
		result.Explanation = "回答が短すぎます（10文字未満）"
		return result, nil
	}

	// 2. ネガティブキーワードチェック
	var negativeKeywords []string
	if question.NegativeKeywords != "" {
		json.Unmarshal([]byte(question.NegativeKeywords), &negativeKeywords)
	}

	for _, keyword := range negativeKeywords {
		if strings.Contains(answerLower, strings.ToLower(keyword)) {
			result.Score -= 2
			result.MatchedKeywords = append(result.MatchedKeywords, fmt.Sprintf("-%s", keyword))
			result.NeedsFollowUp = true
			result.FollowUpTrigger = "negative_keyword"
		}
	}

	// 3. ポジティブキーワードチェック
	var positiveKeywords []string
	if question.PositiveKeywords != "" {
		json.Unmarshal([]byte(question.PositiveKeywords), &positiveKeywords)
	}

	matchedCount := 0
	for _, keyword := range positiveKeywords {
		if strings.Contains(answerLower, strings.ToLower(keyword)) {
			result.Score += 1
			result.MatchedKeywords = append(result.MatchedKeywords, fmt.Sprintf("+%s", keyword))
			matchedCount++
		}
	}

	// 4. スコアリングルールの適用
	var scoreRules []models.ScoreRule
	if question.ScoreRules != "" {
		json.Unmarshal([]byte(question.ScoreRules), &scoreRules)
	}

	for _, rule := range scoreRules {
		if e.evaluateRule(rule, answer, answerLength) {
			result.Score += rule.ScoreChange
			result.AppliedRules = append(result.AppliedRules, rule.Description)
		}
	}

	// 5. 信頼度の判定
	if matchedCount == 0 && result.Score <= 0 {
		result.Confidence = "low"
		result.NeedsFollowUp = true
		result.FollowUpTrigger = "no_keywords"
		result.Explanation = "関連キーワードが見つかりませんでした"
	} else if answerLength > 100 && matchedCount >= 2 {
		result.Confidence = "high"
		result.Explanation = "具体的で詳細な回答です"
	} else {
		result.Confidence = "medium"
		result.Explanation = "ある程度の評価ができました"
	}

	// 6. 追加質問が必要かチェック
	var followUpRules []models.FollowUpRule
	if question.FollowUpRules != "" {
		json.Unmarshal([]byte(question.FollowUpRules), &followUpRules)
	}

	for _, rule := range followUpRules {
		if e.shouldTriggerFollowUp(rule, result) {
			result.NeedsFollowUp = true
			result.FollowUpTrigger = rule.Trigger
			break
		}
	}

	return result, nil
}

// evaluateRule 個別のスコアリングルールを評価
func (e *AnswerEvaluator) evaluateRule(rule models.ScoreRule, answer string, answerLength int) bool {
	answerLower := strings.ToLower(answer)

	switch rule.Condition {
	case "contains_any":
		// いずれかのキーワードを含む
		for _, keyword := range rule.Keywords {
			if strings.Contains(answerLower, strings.ToLower(keyword)) {
				return true
			}
		}
		return false

	case "contains_all":
		// すべてのキーワードを含む
		for _, keyword := range rule.Keywords {
			if !strings.Contains(answerLower, strings.ToLower(keyword)) {
				return false
			}
		}
		return true

	case "length_gt":
		// 文字数が指定値より大きい
		if len(rule.Keywords) > 0 {
			threshold := 0
			fmt.Sscanf(rule.Keywords[0], "%d", &threshold)
			return answerLength > threshold
		}
		return false

	case "length_lt":
		// 文字数が指定値より小さい
		if len(rule.Keywords) > 0 {
			threshold := 0
			fmt.Sscanf(rule.Keywords[0], "%d", &threshold)
			return answerLength < threshold
		}
		return false

	case "regex":
		// 正規表現マッチ
		if len(rule.Keywords) > 0 {
			pattern := rule.Keywords[0]
			matched, err := regexp.MatchString(pattern, answer)
			if err != nil {
				return false
			}
			return matched
		}
		return false

	case "has_example":
		// 具体例を含んでいるか（「例えば」「たとえば」「〜した時」など）
		examplePatterns := []string{
			"例えば", "たとえば", "具体的には", "実際に", "した時", "したとき",
			"経験", "〜で", "〜では", "ことがあ",
		}
		for _, pattern := range examplePatterns {
			if strings.Contains(answerLower, pattern) {
				return true
			}
		}
		return false

	default:
		return false
	}
}

// shouldTriggerFollowUp 追加質問が必要か判定
func (e *AnswerEvaluator) shouldTriggerFollowUp(rule models.FollowUpRule, result *EvaluationResult) bool {
	switch rule.Trigger {
	case "low_confidence":
		return result.Confidence == "low"

	case "high_score":
		return result.Score >= 5

	case "no_keywords":
		return len(result.MatchedKeywords) == 0

	case "negative_keyword":
		return result.FollowUpTrigger == "negative_keyword"

	default:
		return false
	}
}

// GetConfidenceLevel スコアから信頼度レベルを取得
func (e *AnswerEvaluator) GetConfidenceLevel(score int, keywordCount int, answerLength int) string {
	if score <= 0 || keywordCount == 0 {
		return "low"
	}

	if score >= 5 && keywordCount >= 2 && answerLength > 50 {
		return "high"
	}

	return "medium"
}
