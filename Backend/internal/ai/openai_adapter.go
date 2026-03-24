// Package ai implements the LLM adapter layer (domain/ai interfaces).
package ai

import (
	"Backend/domain/ai"
	"Backend/internal/openai"
	"context"
)

// OpenAIAdapter は openai.Client を domain/ai.LLMClient に適合させるアダプター。
// 依存注入によって OpenAI 以外の実装に差し替え可能。
type OpenAIAdapter struct {
	client *openai.Client
}

// NewOpenAIAdapter は既存の openai.Client をラップしたアダプターを返す。
func NewOpenAIAdapter(client *openai.Client) ai.LLMClient {
	return &OpenAIAdapter{client: client}
}

// GenerateText はシステムプロンプト＋ユーザープロンプトでテキストを生成する。
func (a *OpenAIAdapter) GenerateText(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return a.client.ResponsesWithTemperature(ctx, systemPrompt, userPrompt, 0.7)
}

// GenerateJSON は JSON 文字列を返すテキスト生成（構造化出力用）。
func (a *OpenAIAdapter) GenerateJSON(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return a.client.ChatCompletionJSON(ctx, systemPrompt, userPrompt, 0.0, 4096)
}

// GenerateEmbedding はテキストの埋め込みベクトルを返す。
func (a *OpenAIAdapter) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return a.client.Embedding(ctx, text)
}

// TranscribeAudio は音声バイト列をテキストに変換する。
func (a *OpenAIAdapter) TranscribeAudio(ctx context.Context, audio []byte, filename string) (string, error) {
	return a.client.Transcribe(ctx, audio, filename)
}
