// Package ai defines the LLM abstraction layer.
// サービス層はこのインターフェースを通じてのみ LLM を呼び出す。
// OpenAI・Claude・Gemini 等の切り替えをアダプター実装で吸収する。
package ai

import "context"

// TextClient はテキスト生成の抽象インターフェース。
type TextClient interface {
	// GenerateText はシステムプロンプトとユーザープロンプトからテキストを生成する。
	GenerateText(ctx context.Context, systemPrompt, userPrompt string) (string, error)

	// GenerateJSON は JSON 文字列を返すテキスト生成（構造化出力用）。
	GenerateJSON(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// EmbeddingClient はベクトル埋め込みの抽象インターフェース。
type EmbeddingClient interface {
	// GenerateEmbedding はテキストの埋め込みベクトルを返す。
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// AudioClient は音声処理の抽象インターフェース。
type AudioClient interface {
	// TranscribeAudio は音声バイト列をテキストに変換する（Whisper 等）。
	TranscribeAudio(ctx context.Context, audio []byte, filename string) (string, error)
}

// LLMClient は全 AI 操作を統合したインターフェース。
// 大半のユースケースはこれ一つを依存注入すれば足りる。
type LLMClient interface {
	TextClient
	EmbeddingClient
	AudioClient
}
