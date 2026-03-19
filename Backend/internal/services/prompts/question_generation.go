// Package prompts はAIプロンプト文字列を集約管理するパッケージです。
// プロンプトの変更は必ずこのパッケージ内で行い、サービス層での直接定義を避けてください。
package prompts

import "fmt"

// ──────────────────────────────────────────────
// 共通ガイドライン（新卒・中途）
// ──────────────────────────────────────────────

// NewGradGuidelines は新卒学生向け質問作成の共通ガイドラインです。
// 複数の質問生成関数から参照し、重複を防ぎます。
const NewGradGuidelines = `## 【重要】新卒学生向け質問作成ガイドライン

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
- **自然な文脈で**: 会話の流れに沿って、資格や経験について質問する（例: 技術の話題が出たら「その技術を使った経験はありますか？」）`

// MidCareerGuidelines は中途向け質問作成の共通ガイドラインです。
const MidCareerGuidelines = `## 【重要】中途向け質問ガイドライン
- 実務経験・業務・プロジェクト・成果・数値に触れる
- 役割・判断・工夫・関係者との調整を具体的に聞く
- 抽象的ではなく、具体的なシーンを想定して聞く
- 質問は1つのみ、説明や前置きは不要
- 既出質問と重複しない`

// ──────────────────────────────────────────────
// フェーズ別評価観点
// ──────────────────────────────────────────────

// PhaseEvaluationPoints は各フェーズで引き出すべき情報を明示した評価観点マップです。
// フェーズ名をキーとして、対応する評価観点セクション文字列を返します。
var PhaseEvaluationPoints = map[string]string{
	"job_analysis": `## このフェーズの評価観点（職種分析）
- 志望する職種タイプ（技術系 / 非技術系 / 企画系）の志向を確認する
- IT・デジタルへの親しみやすさ、実際の経験の有無を把握する
- 業種・業界への興味・関心の方向性を把握する`,

	"interest_analysis": `## このフェーズの評価観点（興味分析）
- 内発的動機（好奇心・楽しさ・やりたいこと）vs 外発的動機（給与・安定・評価）の度合いを見る
- どのような場面で「夢中になれるか」を引き出す
- 仕事に求める意味・価値観（社会貢献・技術的挑戦・人との関わりなど）を把握する`,

	"aptitude_analysis": `## このフェーズの評価観点（適性分析）
- 個人作業 vs チーム作業の好み・得意不得意を確認する
- コミュニケーションスタイル（聴き役・発信役・調整役）を把握する
- 問題発生時の対処スタイル（相談・独力解決・回避）を見る`,

	"future_analysis": `## このフェーズの評価観点（将来分析）
- 5年後・10年後のビジョンの具体性（役割・スキル・生活スタイル）を確認する
- 成長意欲（新しい挑戦を求める）vs 安定志向（実績を積み上げる）のバランスを把握する
- 働き方の優先度（収入・ワークライフバランス・キャリアアップ・職場環境）を確認する`,
}

// ──────────────────────────────────────────────
// 戦略的質問生成プロンプト（generateStrategicQuestion 用）
// ──────────────────────────────────────────────

// BuildStrategicQuestionPrompt は戦略的質問生成用のプロンプトを構築します。
// targetLevel が "中途" の場合は中途向けプロンプトを返します。
func BuildStrategicQuestionPrompt(
	targetLevel, phaseContext, choiceGuidance,
	historyText, scoreAnalysis, askedQuestionsText,
	questionPurpose, targetCategory, description,
	jobCategoryName string, industryID, jobCategoryID uint,
) string {
	// フェーズ別評価観点を追加
	phaseEvalPoint := ""
	// phaseContext からフェーズ名を特定する必要があるため、呼び出し元から渡す設計とせず
	// phaseContext 内に評価観点を埋め込む形で構築済みであることを前提とする

	if targetLevel == "中途" {
		return fmt.Sprintf(`あなたは中途向けの就職適性診断の専門家です。
これまでの会話と評価状況を分析し、**実務経験を引き出しやすく、企業選定に役立つ質問**を1つ生成してください。
%s
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

%s

**志望職種: %s, 業界ID: %d, 職種ID: %d を考慮して、この職種に相応しい文脈で質問を生成してください。**

質問のみを返してください。説明や補足は一切不要です。`,
			phaseContext,
			choiceGuidance,
			phaseEvalPoint,
			historyText,
			scoreAnalysis,
			askedQuestionsText,
			questionPurpose,
			targetCategory,
			description,
			MidCareerGuidelines,
			jobCategoryName,
			industryID,
			jobCategoryID)
	}

	return fmt.Sprintf(`あなたは新卒学生向けの就職適性診断の専門家です。
これまでの会話と評価状況を分析し、**学生が答えやすく、企業選定に役立つ質問**を1つ生成してください。
%s
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

%s

**志望職種: %s, 業界ID: %d, 職種ID: %d を考慮して、この職種に相応しい文脈で質問を生成してください。特に「技術志向」を評価する場合は、職種がエンジニアであればプログラミングについて、非エンジニア職種ではITツール活用や効率化の関心について聞き、プログラミング経験を前提としないでください。**

質問のみを返してください。説明や補足は一切不要です。`,
		phaseContext,
		choiceGuidance,
		phaseEvalPoint,
		historyText,
		scoreAnalysis,
		askedQuestionsText,
		questionPurpose,
		targetCategory,
		description,
		NewGradGuidelines,
		jobCategoryName,
		industryID,
		jobCategoryID)
}

// BuildStrategicQuestionPromptWithPhase はフェーズ名を受け取り、評価観点を含む戦略的質問プロンプトを構築します。
func BuildStrategicQuestionPromptWithPhase(
	targetLevel, phaseName, phaseContext, choiceGuidance,
	historyText, scoreAnalysis, askedQuestionsText,
	questionPurpose, targetCategory, description,
	jobCategoryName string, industryID, jobCategoryID uint,
) string {
	phaseEvalPoint := ""
	if ep, ok := PhaseEvaluationPoints[phaseName]; ok {
		phaseEvalPoint = ep + "\n"
	}

	if targetLevel == "中途" {
		return fmt.Sprintf(`あなたは中途向けの就職適性診断の専門家です。
これまでの会話と評価状況を分析し、**実務経験を引き出しやすく、企業選定に役立つ質問**を1つ生成してください。
%s
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

%s

**志望職種: %s, 業界ID: %d, 職種ID: %d を考慮して、この職種に相応しい文脈で質問を生成してください。**

質問のみを返してください。説明や補足は一切不要です。`,
			phaseContext,
			choiceGuidance,
			phaseEvalPoint,
			historyText,
			scoreAnalysis,
			askedQuestionsText,
			questionPurpose,
			targetCategory,
			description,
			MidCareerGuidelines,
			jobCategoryName,
			industryID,
			jobCategoryID)
	}

	return fmt.Sprintf(`あなたは新卒学生向けの就職適性診断の専門家です。
これまでの会話と評価状況を分析し、**学生が答えやすく、企業選定に役立つ質問**を1つ生成してください。
%s
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

%s

**志望職種: %s, 業界ID: %d, 職種ID: %d を考慮して、この職種に相応しい文脈で質問を生成してください。特に「技術志向」を評価する場合は、職種がエンジニアであればプログラミングについて、非エンジニア職種ではITツール活用や効率化の関心について聞き、プログラミング経験を前提としないでください。**

質問のみを返してください。説明や補足は一切不要です。`,
		phaseContext,
		choiceGuidance,
		phaseEvalPoint,
		historyText,
		scoreAnalysis,
		askedQuestionsText,
		questionPurpose,
		targetCategory,
		description,
		NewGradGuidelines,
		jobCategoryName,
		industryID,
		jobCategoryID)
}

// ──────────────────────────────────────────────
// フォールバック質問生成プロンプト（generateQuestionWithAI 用）
// ──────────────────────────────────────────────

// BuildLowConfidenceQuestionPrompt は「わからない」系の回答後の再質問プロンプトを構築します。
func BuildLowConfidenceQuestionPrompt(historyText, lastQuestion string, industryID, jobCategoryID uint) string {
	return fmt.Sprintf(`あなたは新卒学生向けの適性診断インタビュアーです。

## これまでの会話
%s

## 状況
学生が前の質問「%s」に答えられなかったようです。
同じカテゴリで、**より答えやすい質問**を生成してください。

%s

業界ID: %d, 職種ID: %d

**質問のみ**を1つ返してください。説明や補足は不要です。`, historyText, lastQuestion, NewGradGuidelines, industryID, jobCategoryID)
}

// BuildUnevaluatedCategoryQuestionPrompt は未評価カテゴリに対する質問プロンプトを構築します。
func BuildUnevaluatedCategoryQuestionPrompt(historyText, targetCategory, description string, industryID, jobCategoryID uint) string {
	return fmt.Sprintf(`あなたは新卒学生向けの適性診断インタビュアーです。

## これまでの会話
%s

## 次に評価すべきカテゴリ
**%s** (%s)

%s

業界ID: %d, 職種ID: %d

**質問のみ**を1つ返してください。説明や補足は不要です。`, historyText, targetCategory, description, NewGradGuidelines, industryID, jobCategoryID)
}

// BuildDeepeningQuestionPrompt は全カテゴリ評価済み後の深掘り質問プロンプトを構築します。
func BuildDeepeningQuestionPrompt(historyText, highestCategory string, highestScore int, industryID, jobCategoryID uint) string {
	return fmt.Sprintf(`あなたは新卒学生向けの適性診断インタビュアーです。

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

// ──────────────────────────────────────────────
// 質問簡略化プロンプト（simplifyQuestionWithAI 用）
// ──────────────────────────────────────────────

// BuildSimplifyQuestionPrompt は質問簡略化用のプロンプトを構築します。
// 元の意図を保ちながら短く言い換えるよう、自己検証ステップを含みます。
func BuildSimplifyQuestionPrompt(question string) string {
	return fmt.Sprintf(`次の質問を、新卒でも答えやすい短い質問に言い換えてください。

## 制約
- 1文で、40〜80文字程度
- 例示やカッコ補足は入れない
- 元の質問の意図・キーワードを必ず保持する
- 質問文のみを返す

## 自己検証
言い換えた質問が以下を満たすか確認してから出力してください：
1. 元の質問が問いたい「評価対象（技術志向・リーダーシップ等）」が伝わるか
2. 新卒学生が学生生活の経験で答えられる内容か
3. 40〜80文字の範囲に収まっているか

質問:
%s`, question)
}
