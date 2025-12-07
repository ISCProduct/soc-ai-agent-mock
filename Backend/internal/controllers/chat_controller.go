package controllers

import (
	"Backend/internal/models"
	"Backend/internal/services"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type ChatController struct {
	chatService     *services.ChatService
	matchingService *services.MatchingService
}

func NewChatController(chatService *services.ChatService, matchingService *services.MatchingService) *ChatController {
	return &ChatController{
		chatService:     chatService,
		matchingService: matchingService,
	}
}

// Chat チャット処理
func (c *ChatController) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req services.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// バリデーション
	if req.UserID == 0 || req.SessionID == "" || req.Message == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	resp, err := c.chatService.ProcessChat(r.Context(), req)
	if err != nil {
		// エラーログを詳細に出力
		println("Error in ProcessChat:", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// マッチング計算を非同期で実行（レスポンスは待たない）
	go func() {
		if err := c.matchingService.CalculateMatching(r.Context(), req.UserID, req.SessionID); err != nil {
			fmt.Printf("[Chat] Background matching calculation failed: %v\n", err)
		} else {
			fmt.Printf("[Chat] Background matching calculation completed for user %d\n", req.UserID)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetHistory チャット履歴取得
func (c *ChatController) GetHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	history, err := c.chatService.GetChatHistory(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// GetScores ユーザースコア取得
func (c *ChatController) GetScores(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	sessionID := r.URL.Query().Get("session_id")

	if userIDStr == "" || sessionID == "" {
		http.Error(w, "user_id and session_id are required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	scores, err := c.chatService.GetUserScores(uint(userID), sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scores)
}

// GetRecommendations トップ適性企業を取得
func (c *ChatController) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	sessionID := r.URL.Query().Get("session_id")
	limitStr := r.URL.Query().Get("limit")

	if userIDStr == "" || sessionID == "" {
		http.Error(w, "user_id and session_id are required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	limit := 10 // デフォルト
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err == nil && l > 0 {
			limit = l
		}
	}

	// 既存のマッチング結果を取得（事前計算済みを想定）
	fmt.Printf("[GetRecommendations] Fetching pre-calculated matches for user %d, session %s\n", userID, sessionID)
	matches, err := c.matchingService.GetTopMatches(r.Context(), uint(userID), sessionID, limit)
	fmt.Printf("[GetRecommendations] Retrieved %d matches in fast mode\n", len(matches))

	if err != nil || len(matches) == 0 {
		fmt.Printf("[GetRecommendations] No matching results found, returning empty result\n")
		// マッチング結果がない場合は空の配列を返す
		type RecommendationResponse struct {
			Recommendations []interface{} `json:"recommendations"`
		}

		response := RecommendationResponse{
			Recommendations: []interface{}{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// フロントエンド用のレスポンス形式に変換
	type CompanyRecommendation struct {
		ID           int      `json:"id"`
		CategoryName string   `json:"category_name"` // 企業名
		Score        int      `json:"score"`         // マッチスコア
		Reason       string   `json:"reason"`        // マッチ理由
		Industry     string   `json:"industry"`
		Location     string   `json:"location"`
		Employees    string   `json:"employees"`
		TechStack    []string `json:"tech_stack"`
	}

	type RecommendationResponse struct {
		Recommendations []CompanyRecommendation `json:"recommendations"`
	}

	var items []CompanyRecommendation
	for _, match := range matches {
		if match.Company.ID == 0 {
			continue
		}

		employeeCount := "未定"
		if match.Company.EmployeeCount > 0 {
			employeeCount = strconv.Itoa(match.Company.EmployeeCount) + "名"
		}

		techStack := []string{}
		if match.Company.TechStack != "" {
			// 簡易的にカンマ区切りで分割
			techStack = splitTechStack(match.Company.TechStack)
		}

		items = append(items, CompanyRecommendation{
			ID:           int(match.Company.ID),
			CategoryName: match.Company.Name,
			Score:        int(match.MatchScore),
			Reason:       generateMatchReason(match),
			Industry:     match.Company.Industry,
			Location:     match.Company.Location,
			Employees:    employeeCount,
			TechStack:    techStack,
		})
	}

	response := RecommendationResponse{
		Recommendations: items,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateReasonForCategory カテゴリごとのマッチング理由を生成（フォールバック用）
func generateReasonForCategory(category string, score int) string {
	reasons := map[string]string{
		"技術志向":        "最新技術への探求心と技術的な深掘りが評価されています。技術主導型の企業で活躍できるでしょう。",
		"コミュニケーション能力": "優れた対話力と説明力が認められています。チーム協業が重視される企業に適しています。",
		"リーダーシップ":     "主導性と意思決定力が強みです。マネジメント志向のキャリアパスが開かれています。",
		"チームワーク":      "協働と協調性に優れています。大規模チームでの開発に向いています。",
		"問題解決力":       "論理思考と分析力が際立っています。課題解決型のプロジェクトで力を発揮できます。",
		"創造性・発想力":     "独創性と革新的思考が光ります。スタートアップや新規事業で活躍できる素質があります。",
		"計画性・実行力":     "目標設定とタスク管理能力が高く評価されています。プロジェクト型企業に最適です。",
		"学習意欲・成長志向":   "継続学習と成長意識が強みです。教育体制が充実した企業で大きく成長できるでしょう。",
		"ストレス耐性・粘り強さ": "困難への対処力とプレッシャー対応力が優れています。高負荷環境でも安定したパフォーマンスを発揮できます。",
		"ビジネス思考・目標志向": "ビジネス価値の理解と成果志向が強みです。事業会社での活躍が期待されます。",
	}

	if reason, ok := reasons[category]; ok {
		return reason
	}
	return "あなたの特性が評価されています。この分野で活躍できる企業とマッチングしました。"
}

// generateMatchReason 企業マッチングの理由を生成
func generateMatchReason(match *models.UserCompanyMatch) string {
	// AIで生成された理由があればそれを使用
	if match.MatchReason != "" {
		return match.MatchReason
	}

	// 各スコアを収集
	scores := map[string]float64{
		"技術力":        match.TechnicalMatch,
		"チームワーク":     match.TeamworkMatch,
		"リーダーシップ":    match.LeadershipMatch,
		"創造性・発想力":    match.CreativityMatch,
		"安定志向":       match.StabilityMatch,
		"成長意欲":       match.GrowthMatch,
		"ワークライフバランス": match.WorkLifeMatch,
		"挑戦意欲":       match.ChallengeMatch,
		"緻密さ":        match.DetailMatch,
		"コミュニケーション力": match.CommunicationMatch,
	}

	// トップ3を抽出
	type scoreItem struct {
		name  string
		score float64
	}
	var items []scoreItem
	for name, score := range scores {
		items = append(items, scoreItem{name, score})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})

	// 具体的な理由を生成
	if len(items) >= 3 {
		return fmt.Sprintf("総合マッチ度%.0f%% - あなたの%s(%.0f%%)、%s(%.0f%%)、%s(%.0f%%)が企業の求める人材像と高く一致しています。特に%sを活かせる環境で、即戦力として活躍が期待できます。",
			match.MatchScore,
			items[0].name, items[0].score,
			items[1].name, items[1].score,
			items[2].name, items[2].score,
			items[0].name)
	} else if len(items) >= 1 {
		return fmt.Sprintf("総合マッチ度%.0f%% - 特にあなたの%sが企業文化と合致しており、スムーズな適応が見込めます。",
			match.MatchScore, items[0].name)
	}

	return fmt.Sprintf("総合マッチ度%.0f%% - あなたの適性と企業の求める人材像が合致しています。", match.MatchScore)
}

// splitTechStack 技術スタック文字列を配列に変換
func splitTechStack(techStack string) []string {
	if techStack == "" {
		return []string{}
	}
	// JSONパース試行、失敗したらカンマ区切り
	var stack []string
	if err := json.Unmarshal([]byte(techStack), &stack); err == nil {
		return stack
	}
	// カンマ区切りにフォールバック
	parts := strings.Split(techStack, ",")
	result := []string{}
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// GetSessions ユーザーのチャットセッション一覧を取得
func (c *ChatController) GetSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	sessions, err := c.chatService.GetUserChatSessions(uint(userID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}
