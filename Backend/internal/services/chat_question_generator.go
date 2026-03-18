package services

import (
	"Backend/internal/models"
	"context"
	"fmt"
	"strings"
)

// handleSessionStart セッション開始時の初回質問を生成
func (s *ChatService) handleSessionStart(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	fmt.Printf("Starting new session: %s\n", req.SessionID)

	// ユーザー情報を取得
	user, err := s.userRepo.GetUserByID(req.UserID)
	userName := "あなた"
	if err == nil && user != nil && user.Name != "" {
		userName = user.Name
	}

	// 職種選択の質問を生成
	jobQuestion, err := s.jobValidator.GenerateJobSelectionQuestion(ctx)
	if err != nil {
		// エラー時のフォールバック
		jobQuestion = `初めまして！あなたの適性診断をサポートします。

まず、どの職種に興味がありますか？以下から選んでください：

1. エンジニア（プログラミング、開発）
2. 営業（顧客対応、提案）
3. マーケティング（企画、分析）
4. 人事（採用、育成）
5. その他・まだ決めていない

番号で答えても、職種名で答えても構いません。`
	} else {
		jobQuestion = fmt.Sprintf("初めまして、%sさん！あなたの適性診断をサポートします。\n\n%s", userName, jobQuestion)
	}

	response := jobQuestion

	// 初回メッセージを保存
	assistantMsg := &models.ChatMessage{
		SessionID: req.SessionID,
		UserID:    req.UserID,
		Role:      "assistant",
		Content:   response,
	}
	if err := s.chatMessageRepo.Create(assistantMsg); err != nil {
		return nil, fmt.Errorf("failed to save initial message: %w", err)
	}

	return &ChatResponse{
		Response:            response,
		IsComplete:          false,
		TotalQuestions:      15,
		AnsweredQuestions:   0,
		EvaluatedCategories: 0,
		TotalCategories:     10,
	}, nil
}

// generateStrategicQuestion AIが戦略的に次の質問を生成
func (s *ChatService) generateStrategicQuestion(ctx context.Context, history []models.ChatMessage, userID uint, sessionID string, scoreMap map[string]int, allCategories []string, askedTexts map[string]bool, industryID, jobCategoryID uint, targetLevel string, currentPhase *models.UserAnalysisProgress) (string, uint, error) {
	// 会話履歴を構築
	historyText := ""
	for _, msg := range history {
		historyText += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	// 既に聞いた質問のリスト（重複防止を徹底）
	askedQuestionsText := "\n## 【重要】既に聞いた質問（絶対に重複させないこと）\n"
	if len(askedTexts) == 0 {
		askedQuestionsText += "（まだ質問していません）\n"
	} else {
		questionCount := 0
		for text := range askedTexts {
			questionCount++
			askedQuestionsText += fmt.Sprintf("%d. %s\n", questionCount, text)
		}
		askedQuestionsText += fmt.Sprintf("\n**上記%d個の質問と類似・重複する質問は絶対に生成しないでください**\n", questionCount)
	}

	phaseCategories := map[string][]string{
		"job_analysis":      {"技術志向", "創造性志向", "成長志向", "安定志向"},
		"interest_analysis": {"技術志向", "創造性志向", "成長志向", "チャレンジ志向"},
		"aptitude_analysis": {"コミュニケーション力", "チームワーク志向", "リーダーシップ志向", "細部志向"},
		"future_analysis":   {"安定志向", "成長志向", "ワークライフバランス", "チャレンジ志向"},
	}

	allowedCategories := allCategories
	phaseName := ""
	if currentPhase != nil && currentPhase.Phase != nil {
		phaseName = currentPhase.Phase.PhaseName
		if phaseAllowed, ok := phaseCategories[phaseName]; ok && len(phaseAllowed) > 0 {
			allowedCategories = phaseAllowed
		}
	}

	// スコア状況の分析（フェーズ対象カテゴリのみ）
	scoreAnalysis := "## 現在の評価状況\n"
	evaluatedCategories := []string{}
	unevaluatedCategories := []string{}

	for _, cat := range allowedCategories {
		score, exists := scoreMap[cat]
		if exists && score != 0 {
			scoreAnalysis += fmt.Sprintf("- %s: %d点\n", cat, score)
			evaluatedCategories = append(evaluatedCategories, cat)
		} else {
			unevaluatedCategories = append(unevaluatedCategories, cat)
		}
	}

	// 職種名と業界名を取得
	jobCategoryName := "指定なし"
	if jobCategoryID != 0 {
		if jc, err := s.jobCategoryRepo.FindByID(jobCategoryID); err == nil && jc != nil {
			jobCategoryName = jc.Name
		}
	}

	// 企業選定に必要な情報を特定
	var targetCategory string
	var questionPurpose string

	if len(unevaluatedCategories) > 0 {
		// 未評価カテゴリがあれば優先
		targetCategory = unevaluatedCategories[0]
		questionPurpose = fmt.Sprintf("まだ評価できていない「%s」を評価するため", targetCategory)
	} else {
		// 全カテゴリ評価済みなら、スコアが中途半端なものを深掘り
		targetCategory = ""
		for _, cat := range allowedCategories {
			score := scoreMap[cat]
			if score > -3 && score < 3 {
				targetCategory = cat
				questionPurpose = fmt.Sprintf("評価が曖昧な「%s」をより明確に判定するため", cat)
				break
			}
		}

		if targetCategory == "" {
			// 最もスコアが高いカテゴリを深掘り
			highestScore := -100
			for _, cat := range allowedCategories {
				score := scoreMap[cat]
				if score > highestScore {
					highestScore = score
					targetCategory = cat
				}
			}
			questionPurpose = fmt.Sprintf("強みである「%s」をさらに深く評価し、最適な企業を絞り込むため", targetCategory)
		}
	}

	categoryDescriptions := map[string]string{
		"技術志向":       "技術やデジタル活用への興味、学習経験（授業、趣味、独学）→ 技術主導企業か事業主導企業か",
		"コミュニケーション力": "対話力、説明力、プレゼン経験（授業発表、サークル）→ チーム重視企業か個人裁量企業か",
		"リーダーシップ志向":  "主導性、提案力、まとめ役経験（グループワーク、サークル）→ マネジメント志向かスペシャリスト志向か",
		"チームワーク志向":   "協力、役割認識、グループ活動経験（授業、サークル、バイト）→ 大規模チーム企業か少数精鋭企業か",
		"創造性志向":      "独創性、アイデア発想、工夫した経験（課題、趣味）→ スタートアップか大企業か",
		"安定志向":       "長期的キャリア観、安定性重視 → 大手企業かベンチャーか",
		"成長志向":       "学習意欲、自己成長、新しい挑戦（資格、自主学習）→ 教育重視企業か実践重視企業か",
		"チャレンジ志向":    "困難への挑戦、失敗を恐れない姿勢 → 挑戦推奨文化か安定志向文化か",
		"細部志向":       "丁寧さ、正確性、品質へのこだわり → 品質重視企業かスピード重視企業か",
		"ワークライフバランス": "仕事と私生活のバランス観 → ワークライフバランス重視企業か成果主義企業か",
	}
	categoryDescriptionsMid := map[string]string{
		"技術志向":       "技術への興味、業務での技術活用や改善経験 → 技術主導企業か事業主導企業か",
		"コミュニケーション力": "関係者との調整、説明力、合意形成の経験 → チーム重視企業か個人裁量企業か",
		"リーダーシップ志向":  "意思決定、主導性、チームや案件の推進経験 → マネジメント志向かスペシャリスト志向か",
		"チームワーク志向":   "協力、役割認識、チームでの成果創出経験 → 大規模チーム企業か少数精鋭企業か",
		"創造性志向":      "改善提案、業務の工夫、新しいアプローチ → スタートアップか大企業か",
		"安定志向":       "長期的キャリア観、安定性重視 → 大手企業かベンチャーか",
		"成長志向":       "学習意欲、自己成長、新しい挑戦 → 教育重視企業か実践重視企業か",
		"チャレンジ志向":    "困難への挑戦、失敗を恐れない姿勢 → 挑戦推奨文化か安定志向文化か",
		"細部志向":       "丁寧さ、正確性、品質へのこだわり → 品質重視企業かスピード重視企業か",
		"ワークライフバランス": "仕事と私生活のバランス観 → ワークライフバランス重視企業か成果主義企業か",
	}

	// フェーズ情報を追加
	phaseContext := ""
	if currentPhase != nil && currentPhase.Phase != nil {
		if currentPhase.Phase.MaxQuestions > 0 {
			phaseContext = fmt.Sprintf(`
## 現在の分析フェーズ: %s
%s
このフェーズでは%dつ〜%dつの質問を行います。現在%d個目の質問です。
フェーズの目的に沿った質問を生成してください。
`, currentPhase.Phase.DisplayName, currentPhase.Phase.Description,
				currentPhase.Phase.MinQuestions, currentPhase.Phase.MaxQuestions,
				currentPhase.QuestionsAsked+1)
		} else {
			phaseContext = fmt.Sprintf(`
## 現在の分析フェーズ: %s
%s
このフェーズでは最低%dつの質問を行います。現在%d個目の質問です。
フェーズの目的に沿った質問を生成してください。
`, currentPhase.Phase.DisplayName, currentPhase.Phase.Description,
				currentPhase.Phase.MinQuestions,
				currentPhase.QuestionsAsked+1)
		}
	}
	choiceGuidance := ""
	// phaseName はフェーズカテゴリ選定で取得済み
	forceTextQuestion := shouldForceTextQuestion(history, currentPhase)
	if phaseName != "" {
		switch phaseName {
		case "job_analysis":
			choiceGuidance = "- 職種分析では選択肢中心で質問を構成する\n- 4〜5択で興味や方向性を選ばせ、最後に「その他（自由記述）」を用意する\n- 選択肢は必ず「A)」「B)」または「1)」「2)」形式で改行区切りで列挙する\n- 出力は『質問文 + 選択肢列挙』の形式とし、選択肢がない質問は不可\n- 文章でないと判定できない場合のみ自由記述にする（その場合も「その他（自由記述）」として選択肢に含める）"
		case "interest_analysis":
			choiceGuidance = "- 興味分析では選択肢中心で質問を構成する\n- 可能な限り4〜5択で提示し、最後に「その他（自由記述）」を用意する\n- 選択肢は必ず「A)」「B)」または「1)」「2)」形式で改行区切りで列挙する\n- 出力は『質問文 + 選択肢列挙』の形式とし、選択肢がない質問は不可\n- 文章必須の深掘りが必要な場合のみ自由記述にする（その場合も「その他（自由記述）」として選択肢に含める）"
		case "aptitude_analysis":
			choiceGuidance = "- 適性分析では選択肢中心で質問を構成する\n- 4〜5択で具体的な行動や傾向を選ばせる\n- 選択肢は必ず「A)」「B)」または「1)」「2)」形式で改行区切りで列挙する\n- 出力は『質問文 + 選択肢列挙』の形式とし、選択肢がない質問は不可\n- 文章でないと判定できない場合のみ自由記述にする（その場合も「その他（自由記述）」として選択肢に含める）"
		case "future_analysis":
			choiceGuidance = "- 将来分析（待遇・働き方の希望を含む）では選択肢中心で質問を構成する\n- 4〜5択で希望や優先順位を選ばせ、最後に「その他（自由記述）」を用意する\n- 選択肢は必ず「A)」「B)」または「1)」「2)」形式で改行区切りで列挙する\n- 出力は『質問文 + 選択肢列挙』の形式とし、選択肢がない質問は不可\n- 理由や背景が必要な場合のみ自由記述にする（その場合も「その他（自由記述）」として選択肢に含める）"
		}
	}
	if forceTextQuestion {
		choiceGuidance = "- このフェーズでは最低限の自由記述質問が必要です\n- 今回は必ず自由記述で質問を作成する\n- 選択肢は出さない"
	}
	if choiceGuidance != "" {
		choiceGuidance = fmt.Sprintf("## 質問形式の方針\n%s\n", choiceGuidance)
	}

	if strings.TrimSpace(targetLevel) == "" {
		targetLevel = "新卒"
	}

	requiresChoice := currentPhase != nil && !forceTextQuestion && (phaseName == "" || phaseName == "job_analysis" || phaseName == "interest_analysis" || phaseName == "aptitude_analysis" || phaseName == "future_analysis")

	description := categoryDescriptions[targetCategory]
	if targetLevel == "中途" {
		description = categoryDescriptionsMid[targetCategory]
	}

	var prompt string
	if targetLevel == "中途" {
		prompt = fmt.Sprintf(`あなたは中途向けの就職適性診断の専門家です。
これまでの会話と評価状況を分析し、**実務経験を引き出しやすく、企業選定に役立つ質問**を1つ生成してください。
%s
%s
## これまでの会話
%s

%s

%s

## 質問の目的
%s

## 対象カテゴリ: %s
%s

## 【重要】中途向け質問ガイドライン
- 実務経験・業務・プロジェクト・成果・数値に触れる
- 役割・判断・工夫・関係者との調整を具体的に聞く
- 抽象的ではなく、具体的なシーンを想定して聞く
- 質問は1つのみ、説明や前置きは不要
- 既出質問と重複しない

**志望職種: %s, 業界ID: %d, 職種ID: %d を考慮して、この職種に相応しい文脈で質問を生成してください。**

質問のみを返してください。説明や補足は一切不要です。`,
			phaseContext,
			choiceGuidance,
			historyText,
			scoreAnalysis,
			askedQuestionsText,
			questionPurpose,
			targetCategory,
			description,
			jobCategoryName,
			industryID,
			jobCategoryID)
	} else {
		prompt = fmt.Sprintf(`あなたは新卒学生向けの就職適性診断の専門家です。
これまでの会話と評価状況を分析し、**学生が答えやすく、企業選定に役立つ質問**を1つ生成してください。
%s
%s
## これまでの会話
%s

%s

%s

## 質問の目的
%s

## 対象カテゴリ: %s
%s

## 【重要】新卒学生向け質問作成ガイドライン

### 1. **実務経験を前提としない**
❌ 悪い例: 「プロジェクトリーダーとしての経験は？」
✅ 良い例: 「グループ活動で、自分から提案したことはありますか？」

❌ 悪い例: 「業務での課題解決経験は？」
✅ 良い例: 「授業やサークルで困ったとき、どのように対処しましたか？」

### 2. **学生生活で答えられる質問**
以下のような場面を想定：
- 授業、ゼミ、グループワーク
- サークル、部活動
- アルバイト
- 趣味、個人の活動
- 資格勉強、自主学習

### 3. **具体的で答えやすい**
抽象的な質問より、具体的なシーンを想定：
✅ 「グループワークで意見が分かれたとき、どうしましたか？」
✅ 「新しい技術やツールに触れ始めたきっかけは何ですか？」
✅ 「サークルやバイトで、どんな役割が多かったですか？」

### 4. **小さな経験も評価**
「どんな小さなことでも構いません」と添える：
✅ 「リーダー経験がなくても、自分から提案したことはありますか？」
✅ 「技術に触れた経験が少なくても、興味はありますか？」

### 5. **選択肢や例を示す**
完全にオープンではなく、具体例を示す：
✅ 「勉強するとき、A) 一人で集中する、B) 友人と一緒に、C) 先生に質問、どれが多いですか？」

## 質問の例（新卒向け・良い例）

**技術志向:**
「身近なITツールや新しい技術に触れることに興味はありますか？もし触れたことがあれば、授業、趣味、独学など、どんな形でも良いので教えてください。」

**チームワーク:**
「グループワークやサークル活動で、メンバーと協力したことはありますか？その時、あなたはどんな役割でしたか？」

**リーダーシップ:**
「グループで何かをするとき、自分から提案したり、まとめ役をしたことはありますか？どんな小さなことでも構いません。」

**問題解決:**
「課題やレポートで行き詰まったとき、どうやって解決しますか？最近の例があれば教えてください。」

**学習意欲:**
「新しいことを学ぶのは好きですか？最近、何か新しく始めたことや、挑戦したことはありますか？」

**コミュニケーション:**
「人と話すことや、自分の考えを伝えることは得意ですか？授業やサークルでの発表、アルバイトでの接客など、経験があれば教えてください。」

## 【重要】避けるべき表現

❌ 「プロジェクト」→ ✅ 「グループワーク」「課題」
❌ 「業務」→ ✅ 「活動」「勉強」
❌ 「クライアント」→ ✅ 「相手」「メンバー」
❌ 「マネジメント」→ ✅ 「まとめ役」「リーダー」
❌ 「実績」→ ✅ 「経験」「やったこと」
❌ 「スキル」→ ✅ 「できること」「学んだこと」

## 【重要】質問生成の制約
1. **重複厳禁**: 既出質問と同じ内容や類似する質問は絶対に生成しないこと
2. **簡潔明瞭**: 質問は1つのみ、説明や前置きは不要
3. **学生が答えられる**: 実務経験不要、学生生活で答えられる内容
4. **具体例を促す**: 「どんな小さなことでも」「例えば授業やサークルで」
5. **文脈の活用**: これまでの会話の流れを自然に継続
6. **進捗表示禁止**: 質問に進捗状況（例: 📊 進捗: X/10カテゴリ評価済み）を含めないこと
7. **親しみやすい言葉**: 堅苦しくなく、話しかけるような口調

**技術志向・専門性を評価する場合:**
「授業や個人制作などで取り組んだものづくりの経験があれば教えてください。使った技術やツール、担当したことがあれば教えてください。」

## 質問生成時の重要な指針
- **資格・認定について**: 適切なタイミングで、保有資格や勉強中の資格について尋ねることで、学習意欲や専門性を評価する
- **経験・実績について**: プロジェクト経験、インターン、アルバイト、課外活動などの具体的な経験を聞き出し、スキルレベルと適性を判断する
- **自然な文脈で**: 会話の流れに沿って、資格や経験について質問する（例: 技術の話題が出たら「その技術を使った経験はありますか？」）

**志望職種: %s, 業界ID: %d, 職種ID: %d を考慮して、この職種に相応しい文脈で質問を生成してください。特に「技術志向」を評価する場合は、職種がエンジニアであればプログラミングについて、非エンジニア職種ではITツール活用や効率化の関心について聞き、プログラミング経験を前提としないでください。**

質問のみを返してください。説明や補足は一切不要です。`,
			phaseContext,
			choiceGuidance,
			historyText,
			scoreAnalysis,
			askedQuestionsText,
			questionPurpose,
			targetCategory,
			description,
			jobCategoryName,
			industryID,
			jobCategoryID)
	}

	questionText, err := s.aiCallWithRetries(ctx, prompt)
	if err != nil {
		return "", 0, err
	}

	// 質問文をクリーンアップ
	questionText = strings.TrimSpace(questionText)
	questionText = strings.Trim(questionText, `"「」`)

	// フォールバック: AIが空を返した場合は簡易質問を使用する
	if questionText == "" {
		fallbackQuestion := s.selectFallbackQuestion(targetCategory, jobCategoryID, targetLevel, askedTexts)
		if fallbackQuestion != "" {
			questionText = fallbackQuestion
		} else {
			questionText = "すみません、質問を生成できませんでした。少し時間をおいてからもう一度お試しください。"
		}
	}

	// 選択肢必須フェーズで選択肢がない場合は再生成
	if requiresChoice && isTextBasedQuestion(questionText) {
		for attempt := 0; attempt < 2; attempt++ {
			choicePrompt := fmt.Sprintf(`以下の質問は選択肢が不足しています。
"%s"

必ず4〜5個の選択肢を「A)」「B)」「C)」「D)」「E)」または「1)」「2)」「3)」「4)」「5)」形式で改行区切りで列挙し、最後に「その他（自由記述）」を含めてください。

質問文は1つのみ。説明は不要です。質問文の後に選択肢を列挙してください。`, questionText)

			regenerated, err := s.aiCallWithRetries(ctx, choicePrompt)
			if err != nil {
				break
			}
			regenerated = strings.TrimSpace(regenerated)
			regenerated = strings.Trim(regenerated, `"「」`)
			if regenerated != "" {
				questionText = regenerated
			}
			if !isTextBasedQuestion(questionText) {
				break
			}
		}
		if isTextBasedQuestion(questionText) {
			questionText = buildChoiceFallback(questionText, phaseName)
		}
	}

	// 重複チェック（完全一致および類似度チェック）を最大3回まで試行
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		isDuplicate := false
		duplicateReason := ""

		// 完全一致チェック
		if askedTexts[questionText] {
			isDuplicate = true
			duplicateReason = fmt.Sprintf("完全一致: %s", questionText)
		} else {
			// 類似度チェック
			for askedQ := range askedTexts {
				similarity := calculateSimilarity(questionText, askedQ)
				if similarity > 0.6 { // 閾値を0.6に下げて、より厳格に
					isDuplicate = true
					duplicateReason = fmt.Sprintf("類似度%.2f: %s", similarity, askedQ)
					break
				}
			}
		}

		if !isDuplicate {
			break // 重複なし、使用可能
		}

		fmt.Printf("Retry %d: Duplicate detected (%s)\n", attempt+1, duplicateReason)

		// 再生成プロンプト
		retryPrompt := fmt.Sprintf(`以下の質問は既に聞いているか類似しています：
"%s"

既に聞いた全ての質問：
%s

これらと完全に異なる新しい質問を生成してください。
対象カテゴリ: %s
**質問のみ**を返してください。説明は不要です。`,
			questionText,
			func() string {
				var list string
				count := 0
				for q := range askedTexts {
					count++
					list += fmt.Sprintf("%d. %s\n", count, q)
				}
				return list
			}(),
			targetCategory)

		questionText, err = s.aiCallWithRetries(ctx, retryPrompt)
		if err != nil {
			return "", 0, err
		}
		questionText = strings.TrimSpace(questionText)
		questionText = strings.Trim(questionText, `"「」`)

		// 最後の試行で重複してもそのまま使用（無限ループ防止）
		if attempt == maxRetries-1 {
			fmt.Printf("Max retries reached, using question anyway: %s\n", questionText)
		}
	}

	// AI生成質問をデータベースに保存（空文字は保存しない）
	questionText = strings.TrimSpace(questionText)
	if questionText == "" {
		fmt.Printf("Warning: AI generated empty question even after fallback, not saving. user=%d session=%s\n", userID, sessionID)
		return "", 0, fmt.Errorf("ai returned empty question")
	}

	aiGenQuestion := &models.AIGeneratedQuestion{
		UserID:       userID,
		SessionID:    sessionID,
		TemplateID:   nil, // AI生成の場合はNULL
		QuestionText: questionText,
		Weight:       7, // 戦略的質問は重み高め
		IsAnswered:   false,
		ContextData:  fmt.Sprintf(`{"target_category": "%s", "purpose": "%s"}`, targetCategory, questionPurpose),
	}

	if err := s.aiGeneratedQuestionRepo.Create(aiGenQuestion); err != nil {
		return "", 0, fmt.Errorf("failed to save AI generated question: %w", err)
	}

	return questionText, aiGenQuestion.ID, nil
}

func (s *ChatService) generateQuestionWithAI(ctx context.Context, history []models.ChatMessage, userID uint, sessionID string, industryID, jobCategoryID uint) (string, uint, error) {
	// 会話履歴を構築
	historyText := ""
	hasLowConfidenceAnswer := false
	lastQuestion := ""

	for i, msg := range history {
		historyText += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)

		if msg.Role == "assistant" {
			lastQuestion = msg.Content
		}

		// 最後のユーザー回答が「わからない」系かチェック
		if i == len(history)-1 && msg.Role == "user" {
			lowConfidencePatterns := []string{
				"わからない", "わからない", "わかりません", "分かりません",
				"よくわからない", "特にない", "思いつかない", "ありません",
			}
			for _, pattern := range lowConfidencePatterns {
				if strings.Contains(strings.ToLower(msg.Content), pattern) {
					hasLowConfidenceAnswer = true
					break
				}
			}
		}
	}

	// 現在のスコアを取得して、まだ評価が不十分な領域を特定
	scores, err := s.userWeightScoreRepo.FindByUserAndSession(userID, sessionID)
	if err != nil {
		fmt.Printf("Warning: failed to get scores for question generation: %v\n", err)
	}

	// スコア分布を分析
	scoreMap := make(map[string]int)
	for _, score := range scores {
		scoreMap[score.WeightCategory] = score.Score
	}

	// まだ評価されていないカテゴリを特定（職種に応じて並び順を調整）
	allCategories := s.getCategoryOrder(jobCategoryID)

	unevaluatedCategories := []string{}
	for _, cat := range allCategories {
		if _, exists := scoreMap[cat]; !exists {
			unevaluatedCategories = append(unevaluatedCategories, cat)
		}
	}

	var prompt string
	if hasLowConfidenceAnswer {
		// わからない回答の場合は、同じカテゴリで別の角度から質問
		prompt = fmt.Sprintf(`あなたは新卒学生向けの適性診断インタビュアーです。

## これまでの会話
%s

## 状況
学生が前の質問「%s」に答えられなかったようです。
同じカテゴリで、**より答えやすい質問**を生成してください。

## 【重要】新卒学生向け質問ガイドライン

### 1. 実務経験を前提としない
❌ 「プロジェクトでの経験は？」
✅ 「授業やサークルでの経験は？」

❌ 悪い例: 「業務での課題解決経験は？」
✅ 良い例: 「授業やサークルで困ったとき、どのように対処しましたか？」

### 2. より具体的なシーンを提示
❌ 「リーダーシップについて教えて」
✅ 「グループワークで、自分から提案したことはありますか？」

### 3. 小さな経験も評価
「どんな小さなことでも構いません」と添える

### 4. 身近な例を挙げる
「例えば、授業、サークル、アルバイト、趣味など」

### 5. 選択肢や例を示す
完全にオープンではなく、具体例を示す

## 質問の例（答えやすい良い例）

**技術志向:**
「身近なITツールや新しい技術に興味はありますか？授業で触れた程度でも、使ったことがあれば教えてください。」

**チームワーク:**
「グループで作業するとき、どんな役割が多いですか？例えば、まとめ役、アイデアを出す人、サポート役など。」

**リーダーシップ:**
「友達と遊ぶ計画を立てるとき、自分から提案することはありますか？」

**コミュニケーション:**
「授業で発表したり、アルバイトで接客したりする経験はありますか？」

**避けるべき言葉:**
- プロジェクト → グループワーク、課題
- 業務 → 活動、勉強
- マネジメント → まとめ役
- 実績 → 経験、やったこと

業界ID: %d, 職種ID: %d

**質問のみ**を1つ返してください。説明や補足は不要です。`, historyText, lastQuestion, industryID, jobCategoryID)
	} else if len(unevaluatedCategories) > 0 {
		// 未評価のカテゴリがある場合は、それを重点的に評価
		targetCategory := unevaluatedCategories[0]

		categoryDescriptions := map[string]string{
			"技術志向":       "技術やデジタル活用への興味（授業、趣味、独学）",
			"コミュニケーション力": "人と話すこと、説明すること、協力すること",
			"リーダーシップ志向":  "自分から提案、まとめ役、メンバーのサポート",
			"チームワーク志向":   "グループでの協力、役割分担、助け合い",
			"創造性志向":      "アイデア発想、工夫、新しいアプローチ",
			"安定志向":       "長期的キャリア観、安定性への考え方",
			"成長志向":       "学習意欲、自己成長、新しい挑戦",
			"チャレンジ志向":    "困難への挑戦、失敗を恐れない姿勢",
			"細部志向":       "丁寧さ、正確性、品質へのこだわり",
			"ワークライフバランス": "仕事と私生活のバランス観",
		}

		description := categoryDescriptions[targetCategory]

		prompt = fmt.Sprintf(`あなたは新卒学生向けの適性診断インタビュアーです。

## これまでの会話
%s

## 次に評価すべきカテゴリ
**%s** (%s)

## 【重要】新卒学生向け質問ガイドライン

### 1. 実務経験を前提としない
学生生活で答えられる質問：
- 授業、ゼミ、グループワーク
- サークル、部活動
- アルバイト
- 趣味、個人活動

### 2. 具体的で答えやすい
❌ 「プロジェクトでの問題解決経験は？」
✅ 「課題やレポートで行き詰まったとき、どうしましたか？」

### 3. 小さな経験も評価
「どんな小さなことでも構いません」と添える

### 4. 自然な会話の流れ
これまでの会話を踏まえた質問

## 良い質問の例（新卒向け）

**技術志向:**
「技術やツールに触れた経験で、楽しかったことや苦労したことはありますか？」

**チームワーク:**
「グループ活動で、メンバーと協力してうまくいったとき、どんな気持ちでしたか？」

**リーダーシップ:**
「自分から提案したとき、周りの反応はどうでしたか？やりがいを感じましたか？」

**成長志向:**
「新しいことを学ぶとき、どんなことに気をつけていますか？直近で学んだことはありますか？」

業界ID: %d, 職種ID: %d

**質問のみ**を1つ返してください。説明や補足は不要です。`, historyText, targetCategory, description, industryID, jobCategoryID)
	} else {
		// 全カテゴリ評価済みの場合は、深掘り質問
		// スコアが高いカテゴリをさらに深掘り
		var highestCategory string
		highestScore := -100
		for cat, score := range scoreMap {
			if score > highestScore {
				highestScore = score
				highestCategory = cat
			}
		}

		prompt = fmt.Sprintf(`あなたは新卒学生向けの適性診断インタビュアーです。

## これまでの会話
%s

## 現在の評価状況
学生の強みとして「%s」が見えてきました（スコア: %d）。
この強みを深掘りし、具体的なエピソードや考え方を引き出す質問を作成してください。

## 【重要】新卒学生向け深掘り質問ガイドライン

### 1. 実務経験を前提としない
学生生活で答えられる質問：
- 授業、ゼミ、グループワーク
- サークル、部活動
- アルバイト
- 趣味、個人活動

### 2. 具体的なエピソードを引き出す
「その中で、特に印象に残っている経験はありますか？」
「それをどう感じましたか？」

### 3. 考え方や価値観を探る
「なぜそう思ったのですか？」
「それがあなたにとって大切な理由は？」

### 4. 強みの本質を確認
表面的でなく、本質的な能力や価値観を探る

### 5. 小さな経験も大切に
「どんな小さなことでも構いません」と添える

## 良い深掘り質問の例

**技術志向が強い場合:**
「新しい技術やツールに触れる中で、一番楽しかった瞬間や達成感を感じたことはありますか？」

**チームワークが強い場合:**
「グループ活動で、メンバーと協力してうまくいったとき、どんな気持ちでしたか？」

**リーダーシップが強い場合:**
「自分から提案したとき、周りの反応はどうでしたか？やりがいを感じましたか？」

**成長志向が強い場合:**
「新しいことを学び続けるモチベーションは何ですか？」

業界ID: %d, 職種ID: %d

**質問のみ**を1つ返してください。説明や補足は不要です。`, historyText, highestCategory, highestScore, industryID, jobCategoryID)
	}

	questionText, err := s.aiCallWithRetries(ctx, prompt)
	if err != nil {
		return "", 0, err
	}

	// 質問文をクリーンアップ
	questionText = strings.TrimSpace(questionText)
	questionText = strings.Trim(questionText, `"「」`)

	// AI生成質問をデータベースに保存
	aiGenQuestion := &models.AIGeneratedQuestion{
		UserID:       userID,
		SessionID:    sessionID,
		TemplateID:   nil, // AI生成の場合はNULL
		QuestionText: questionText,
		Weight:       5, // デフォルト重み
		IsAnswered:   false,
	}

	if err := s.aiGeneratedQuestionRepo.Create(aiGenQuestion); err != nil {
		return "", 0, fmt.Errorf("failed to save AI generated question: %w", err)
	}

	return questionText, aiGenQuestion.ID, nil
}

func (s *ChatService) fallbackQuestionForCategory(category string, jobCategoryID uint, targetLevel string) string {
	switch category {
	case "技術志向":
		return s.techInterestQuestion(jobCategoryID, targetLevel)
	case "コミュニケーション能力":
		if targetLevel == "中途" {
			return "業務で関係者と調整した経験はありますか？どんな場面で、どのように進めましたか？"
		}
		return "グループワークであなたがよく担当する役割は何ですか？（例: アイデア出し、まとめ役、サポートなど）"
	case "リーダーシップ":
		if targetLevel == "中途" {
			return "業務でチームや案件をリードした経験はありますか？どのように進めましたか？"
		}
		return "グループで何かをまとめた経験はありますか？どんな場面でしたか？"
	case "チームワーク":
		if targetLevel == "中途" {
			return "チームで協力して成果を出した経験はありますか？あなたの役割も教えてください。"
		}
		return "サークルや授業で、チームで取り組んだ経験はありますか？どんな役割でしたか？"
	case "問題解決力":
		if targetLevel == "中途" {
			return "業務で課題が起きたとき、どのように解決しましたか？最近の例を教えてください。"
		}
		return "課題やレポートで困ったとき、どのように解決しましたか？最近の例を教えてください。"
	case "創造性・発想力":
		if targetLevel == "中途" {
			return "業務で改善や工夫を提案した経験はありますか？どんな内容でしたか？"
		}
		return "新しいアイデアを出した経験はありますか？どんな工夫をしましたか？"
	case "計画性・実行力":
		if targetLevel == "中途" {
			return "業務で計画を立てて実行した経験を教えてください。どのように進めましたか？"
		}
		return "何かを計画して実行した経験を教えてください。どのように進めましたか？"
	case "学習意欲・成長志向":
		if targetLevel == "中途" {
			return "業務に役立てるために学んだことはありますか？直近の例があれば教えてください。"
		}
		return "新しいことを学ぶとき、どうやって学習を進めますか？直近で学んだことはありますか？"
	case "ストレス耐性・粘り強さ":
		if targetLevel == "中途" {
			return "業務で困難に直面したとき、どのように乗り越えましたか？具体例があれば教えてください。"
		}
		return "困難に直面したとき、どのように乗り越えましたか？具体例があれば教えてください。"
	case "ビジネス思考・目標志向":
		if targetLevel == "中途" {
			return "業務で目標を立てて達成した経験はありますか？どんな目標でしたか？"
		}
		return "目標を立てて達成した経験はありますか？どんな目標でしたか？"
	default:
		return ""
	}
}

func (s *ChatService) fallbackQuestionsForCategory(category string, jobCategoryID uint, targetLevel string) []string {
	switch category {
	case "技術志向":
		return []string{
			s.techInterestQuestion(jobCategoryID, targetLevel),
			"最近触れた技術やツールはありますか？どんなことでも大丈夫です。",
		}
	case "コミュニケーション能力":
		if targetLevel == "中途" {
			return []string{
				"業務で相手に説明するとき、意識していることは何ですか？",
				"関係者とのやり取りで工夫したことはありますか？",
			}
		}
		return []string{
			"人に説明するとき、意識していることは何ですか？",
			"授業やサークルで発表した経験はありますか？",
		}
	case "リーダーシップ":
		if targetLevel == "中途" {
			return []string{
				"業務で主導したことはありますか？どんな場面でしたか？",
				"周りを巻き込んで進めた経験はありますか？",
			}
		}
		return []string{
			"自分から提案したりまとめ役をしたことはありますか？",
			"人をまとめた経験があれば教えてください。",
		}
	case "チームワーク":
		if targetLevel == "中途" {
			return []string{
				"チームで協力して進めた仕事はありますか？",
				"メンバーと連携する際に意識していることは？",
			}
		}
		return []string{
			"グループで協力した経験はありますか？",
			"チームで取り組んだときの役割を教えてください。",
		}
	case "問題解決力":
		if targetLevel == "中途" {
			return []string{
				"業務で困ったとき、どう解決しましたか？",
				"トラブル対応で工夫したことはありますか？",
			}
		}
		return []string{
			"困ったとき、どうやって解決しましたか？",
			"課題で行き詰まったときの対処法を教えてください。",
		}
	case "創造性・発想力":
		if targetLevel == "中途" {
			return []string{
				"業務で改善案を出したことはありますか？",
				"新しいアイデアを提案した経験はありますか？",
			}
		}
		return []string{
			"新しいアイデアを出した経験はありますか？",
			"いつもと違う工夫をしたことはありますか？",
		}
	case "計画性・実行力":
		if targetLevel == "中途" {
			return []string{
				"業務で計画を立てて進めた経験はありますか？",
				"期限に向けて進めた仕事はありますか？",
			}
		}
		return []string{
			"計画を立てて進めた経験はありますか？",
			"期限を意識して進めたことはありますか？",
		}
	case "学習意欲・成長志向":
		if targetLevel == "中途" {
			return []string{
				"最近学んだことはありますか？",
				"仕事のために学習したことはありますか？",
			}
		}
		return []string{
			"最近学んだことはありますか？",
			"新しく始めたことはありますか？",
		}
	case "ストレス耐性・粘り強さ":
		if targetLevel == "中途" {
			return []string{
				"大変だった仕事をどう乗り越えましたか？",
				"プレッシャーのある場面での対処を教えてください。",
			}
		}
		return []string{
			"大変なとき、どうやって乗り越えましたか？",
			"うまくいかない時の気持ちの切り替え方は？",
		}
	case "ビジネス思考・目標志向":
		if targetLevel == "中途" {
			return []string{
				"目標を立てて取り組んだ経験はありますか？",
				"成果を意識して進めた仕事はありますか？",
			}
		}
		return []string{
			"目標を立てて取り組んだ経験はありますか？",
			"目標達成のために工夫したことはありますか？",
		}
	default:
		return []string{s.fallbackQuestionForCategory(category, jobCategoryID, targetLevel)}
	}
}

func (s *ChatService) selectFallbackQuestion(category string, jobCategoryID uint, targetLevel string, askedTexts map[string]bool) string {
	options := s.fallbackQuestionsForCategory(category, jobCategoryID, targetLevel)
	for _, q := range options {
		if strings.TrimSpace(q) == "" {
			continue
		}
		if !askedTexts[q] {
			return q
		}
	}
	generic := []string{}
	if targetLevel == "中途" {
		generic = []string{
			"最近取り組んだ仕事やタスクはありますか？簡単に教えてください。",
			"仕事で工夫したことがあれば教えてください。",
		}
	} else {
		generic = []string{
			"最近頑張ったことはありますか？",
			"新しく挑戦したことはありますか？",
		}
	}
	for _, q := range generic {
		if strings.TrimSpace(q) == "" {
			continue
		}
		if !askedTexts[q] {
			return q
		}
	}
	return ""
}

func (s *ChatService) techInterestQuestion(jobCategoryID uint, targetLevel string) string {
	code := s.getJobCategoryCode(jobCategoryID)
	if targetLevel == "中途" {
		switch {
		case strings.HasPrefix(code, "ENG"):
			return "業務で使った技術や、最近取り組んだ開発について教えてください。"
		case strings.HasPrefix(code, "SALES"):
			return "営業活動でITツールや仕組みを活用した経験はありますか？どのように使いましたか？"
		case strings.HasPrefix(code, "MKT"):
			return "データやデジタルを使った施策の経験はありますか？内容を教えてください。"
		case strings.HasPrefix(code, "HR"):
			return "人事領域でITツールや仕組みを使った経験はありますか？具体例があれば教えてください。"
		case strings.HasPrefix(code, "FIN"):
			return "数値管理や分析で使ったツール・仕組みがあれば教えてください。"
		case strings.HasPrefix(code, "CONS"):
			return "業務でデータやツールを使って課題整理をした経験はありますか？"
		default:
			return "業務でITツールや仕組みを活用した経験はありますか？"
		}
	}
	switch {
	case strings.HasPrefix(code, "ENG"):
		return "プログラミングや技術に触れるのは好きですか？授業や趣味、独学で触れたことがあれば教えてください。"
	case strings.HasPrefix(code, "SALES"):
		return "営業で役立ちそうなITツールやアプリを使うことに興味はありますか？授業やアルバイトで使ったことがあれば教えてください。"
	case strings.HasPrefix(code, "MKT"):
		return "データやSNS分析など、デジタルを使って考えることに興味はありますか？授業や趣味で触れたことがあれば教えてください。"
	case strings.HasPrefix(code, "HR"):
		return "人事の仕事で役立ちそうなITツールや仕組みに興味はありますか？授業やアルバイトで使ったことがあれば教えてください。"
	case strings.HasPrefix(code, "FIN"):
		return "数字を扱う作業や表計算などのツールを使うのは好きですか？授業やアルバイトで使ったことがあれば教えてください。"
	case strings.HasPrefix(code, "CONS"):
		return "調べた情報をまとめるためにITツールやデータを使うことに興味はありますか？授業や課題での経験があれば教えてください。"
	default:
		return "身近なITツールやアプリを使って作業を効率化することに興味はありますか？授業やアルバイトで使った例があれば教えてください。"
	}
}

func (s *ChatService) getCategoryOrder(jobCategoryID uint) []string {
	defaultOrder := []string{
		"技術志向", "コミュニケーション能力", "リーダーシップ", "チームワーク",
		"問題解決力", "創造性・発想力", "計画性・実行力", "学習意欲・成長志向",
		"ストレス耐性・粘り強さ", "ビジネス思考・目標志向",
	}
	undecidedOrder := []string{
		"コミュニケーション能力", "学習意欲・成長志向", "問題解決力", "チームワーク",
		"ビジネス思考・目標志向", "計画性・実行力", "創造性・発想力", "ストレス耐性・粘り強さ",
		"リーダーシップ", "技術志向",
	}

	if jobCategoryID == 0 {
		return undecidedOrder
	}

	code := s.getJobCategoryCode(jobCategoryID)
	switch {
	case strings.HasPrefix(code, "ENG"):
		return []string{
			"技術志向", "問題解決力", "学習意欲・成長志向", "創造性・発想力",
			"計画性・実行力", "チームワーク", "コミュニケーション能力", "ストレス耐性・粘り強さ",
			"ビジネス思考・目標志向", "リーダーシップ",
		}
	case strings.HasPrefix(code, "SALES"):
		return []string{
			"コミュニケーション能力", "ビジネス思考・目標志向", "チームワーク", "ストレス耐性・粘り強さ",
			"計画性・実行力", "学習意欲・成長志向", "問題解決力", "リーダーシップ",
			"創造性・発想力", "技術志向",
		}
	case strings.HasPrefix(code, "MKT"):
		return []string{
			"創造性・発想力", "問題解決力", "コミュニケーション能力", "ビジネス思考・目標志向",
			"学習意欲・成長志向", "計画性・実行力", "チームワーク", "リーダーシップ",
			"ストレス耐性・粘り強さ", "技術志向",
		}
	case strings.HasPrefix(code, "HR"):
		return []string{
			"コミュニケーション能力", "チームワーク", "リーダーシップ", "学習意欲・成長志向",
			"計画性・実行力", "問題解決力", "ストレス耐性・粘り強さ", "ビジネス思考・目標志向",
			"創造性・発想力", "技術志向",
		}
	case strings.HasPrefix(code, "FIN"):
		return []string{
			"計画性・実行力", "問題解決力", "ビジネス思考・目標志向", "ストレス耐性・粘り強さ",
			"学習意欲・成長志向", "コミュニケーション能力", "チームワーク", "リーダーシップ",
			"創造性・発想力", "技術志向",
		}
	case strings.HasPrefix(code, "CONS"):
		return []string{
			"問題解決力", "コミュニケーション能力", "学習意欲・成長志向", "ビジネス思考・目標志向",
			"チームワーク", "リーダーシップ", "計画性・実行力", "ストレス耐性・粘り強さ",
			"創造性・発想力", "技術志向",
		}
	default:
		return defaultOrder
	}
}

func (s *ChatService) simplifyQuestionWithAI(ctx context.Context, question string) (string, error) {
	prompt := fmt.Sprintf(`次の質問を、新卒でも答えやすい短い質問に言い換えてください。

制約:
- 1文で、40〜80文字程度
- 例示やカッコ補足は入れない
- 同じ意味を保つ
- 質問文のみを返す

質問:
%s`, question)
	return s.aiCallWithRetries(ctx, prompt)
}

func (s *ChatService) tryGetPredefinedQuestion(userID uint, sessionID string, prioritizeCategory string, industryID, jobCategoryID uint, targetLevel string, askedTexts map[string]bool, currentPhase string) (*models.PredefinedQuestion, error) {
	if jobCategoryID == 0 {
		// 職種未決定の場合はAI質問に任せる
		return nil, nil
	}
	if strings.TrimSpace(targetLevel) == "" {
		targetLevel = "新卒"
	}

	// すべての事前定義質問を取得して、質問文でフィルタ
	allQuestions, err := s.predefinedQuestionRepo.FindActiveQuestions(targetLevel, &industryID, &jobCategoryID, currentPhase)
	if err != nil {
		return nil, err
	}

	// 職種に合う質問のみ残す（汎用質問はAIに任せる）
	jobSpecificQuestions := make([]*models.PredefinedQuestion, 0, len(allQuestions))
	for _, q := range allQuestions {
		if q.JobCategoryID == nil || *q.JobCategoryID != jobCategoryID {
			continue
		}
		jobSpecificQuestions = append(jobSpecificQuestions, q)
	}

	if len(jobSpecificQuestions) == 0 {
		return nil, nil
	}

	// 優先カテゴリで質問を検索（該当がなければAIに任せる）
	var selected *models.PredefinedQuestion
	for _, q := range jobSpecificQuestions {
		if _, asked := askedTexts[q.QuestionText]; asked {
			continue
		}
		if prioritizeCategory != "" && q.Category != prioritizeCategory {
			continue
		}
		if selected == nil || q.Priority > selected.Priority || (q.Priority == selected.Priority && q.ID < selected.ID) {
			selected = q
		}
	}

	if selected == nil {
		return nil, nil
	}

	return selected, nil
}

func (s *ChatService) isJobSelectionQuestion(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	keywords := []string{
		"職種", "どの職種", "IT職種", "興味がありますか", "選んでください",
		"まだ決めていない", "番号で答えても",
	}
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func (s *ChatService) shouldValidateJobCategory(history []models.ChatMessage) bool {
	lastAssistant := s.getLastAssistantMessage(history)
	if strings.TrimSpace(lastAssistant) == "" {
		return true
	}
	return s.isJobSelectionQuestion(lastAssistant)
}
