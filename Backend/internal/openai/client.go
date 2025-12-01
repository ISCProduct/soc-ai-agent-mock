package openai

import (
	"context"
	"errors"
	"os"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// Client は go-openai SDK をラップします。
type Client struct {
	c            *openai.Client
	DefaultModel string
}

func NewFromEnv(optionalModel string) (*Client, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, errors.New("OPENAI_API_KEY is not set")
	}

	model := optionalModel
	if model == "" {
		model = os.Getenv("OPENAI_MODEL")
	}
	if model == "" {
		model = "gpt-4o-mini"
	}

	cli := openai.NewClient(key)
	return &Client{c: cli, DefaultModel: model}, nil
}

func (cli *Client) Responses(ctx context.Context, input string, modelOverride ...string) (string, error) {
	if cli == nil || cli.c == nil {
		return "", errors.New("openai client is nil")
	}

	model := cli.DefaultModel
	if len(modelOverride) > 0 && modelOverride[0] != "" {
		model = modelOverride[0]
	}
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		ctxReq, cancel := context.WithTimeout(ctx, 30*time.Second)

		resp, err := cli.c.CreateChatCompletion(ctxReq, openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: input,
				},
			},
			MaxCompletionTokens: 600,
		})
		cancel()

		if err == nil {
			if len(resp.Choices) > 0 {
				return resp.Choices[0].Message.Content, nil
			}
			return "", nil
		}

		lastErr = err
		println("OpenAI API error (attempt", attempt, "):", err.Error())
		time.Sleep(time.Duration(attempt) * time.Second)
	}

	if lastErr == nil {
		lastErr = errors.New("no response from model")
	}
	return "", lastErr
}
