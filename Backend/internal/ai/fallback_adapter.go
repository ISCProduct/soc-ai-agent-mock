package ai

import (
	"Backend/domain/ai"
	"context"
	"errors"
)

// FallbackAdapter は OpenAI 障害時のフォールバック実装。
// テキスト生成は固定メッセージを返し、埋め込みはエラーを返す。
// 本番でのフォールバック運用と単体テストのモック用途を兼ねる。
type FallbackAdapter struct{}

// NewFallbackAdapter はフォールバックアダプターを返す。
func NewFallbackAdapter() ai.LLMClient {
	return &FallbackAdapter{}
}

func (f *FallbackAdapter) GenerateText(_ context.Context, _, _ string) (string, error) {
	return "現在 AI サービスが利用できません。しばらくしてから再度お試しください。", nil
}

func (f *FallbackAdapter) GenerateJSON(_ context.Context, _, _ string) (string, error) {
	return "{}", nil
}

func (f *FallbackAdapter) GenerateEmbedding(_ context.Context, _ string) ([]float32, error) {
	return nil, errors.New("embedding unavailable: AI service is temporarily down")
}

func (f *FallbackAdapter) TranscribeAudio(_ context.Context, _ []byte, _ string) (string, error) {
	return "", errors.New("transcription unavailable: AI service is temporarily down")
}
