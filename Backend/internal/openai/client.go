package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// Client は go-openai SDK をラップします。
type Client struct {
	c            *openai.Client
	DefaultModel string
	apiKey       string
}

func init() {
	// ジッター用の乱数初期化
	rand.Seed(time.Now().UnixNano())
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
		model = "gpt-5.2"
	}

	cli := openai.NewClient(key)
	return &Client{c: cli, DefaultModel: model, apiKey: key}, nil
}

func (cli *Client) callResponsesAPI(ctx context.Context, input interface{}, model string, temperature *float32, maxOutputTokens int, includeTextFormat bool) (string, error) {
	if cli.apiKey == "" {
		return "", errors.New("openai api key is not set")
	}

	type responsesRequest struct {
		Model           string      `json:"model"`
		Input           interface{} `json:"input"`
		MaxOutputTokens int         `json:"max_output_tokens,omitempty"`
		Temperature     *float32    `json:"temperature,omitempty"`
		Text            interface{} `json:"text,omitempty"`
		Reasoning       interface{} `json:"reasoning,omitempty"`
	}

	payload := responsesRequest{
		Model:           model,
		Input:           input,
		MaxOutputTokens: maxOutputTokens,
		Temperature:     temperature,
		Reasoning: map[string]string{
			"effort": "low",
		},
	}
	if includeTextFormat {
		payload.Text = map[string]interface{}{
			"format": map[string]string{
				"type": "text",
			},
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/responses", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+cli.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.New(string(respBody))
	}

	type responsesContent struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		Refusal string `json:"refusal"`
	}
	type responsesOutput struct {
		Content []responsesContent `json:"content"`
	}
	type responsesResponse struct {
		Output     []responsesOutput `json:"output"`
		OutputText string            `json:"output_text"`
		IncompleteDetails struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
	}
	type responsesError struct {
		Message string `json:"message"`
	}
	type responsesErrorWrapper struct {
		Error responsesError `json:"error"`
	}

	var parsed responsesResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	if strings.TrimSpace(parsed.OutputText) != "" {
		return strings.TrimSpace(parsed.OutputText), nil
	}
	var parsedErr responsesErrorWrapper
	if err := json.Unmarshal(respBody, &parsedErr); err == nil {
		if strings.TrimSpace(parsedErr.Error.Message) != "" {
			return "", errors.New(parsedErr.Error.Message)
		}
	}

	var parts []string
	for _, out := range parsed.Output {
		for _, c := range out.Content {
			if strings.TrimSpace(c.Text) != "" {
				parts = append(parts, strings.TrimSpace(c.Text))
				continue
			}
			if strings.TrimSpace(c.Refusal) != "" {
				return "", errors.New(strings.TrimSpace(c.Refusal))
			}
		}
	}
	if len(parts) == 0 {
		if strings.TrimSpace(parsed.IncompleteDetails.Reason) != "" {
			return "", errors.New("empty response from responses api: " + parsed.IncompleteDetails.Reason)
		}
		snippet := strings.TrimSpace(string(respBody))
		if len(snippet) > 1000 {
			snippet = snippet[:1000]
		}
		return "", errors.New("empty response from responses api: " + snippet)
	}
	return strings.Join(parts, "\n"), nil
}

func (cli *Client) Responses(ctx context.Context, input string, modelOverride ...string) (string, error) {
	if cli == nil || cli.c == nil {
		return "", errors.New("openai client is nil")
	}

	model := cli.DefaultModel
	if len(modelOverride) > 0 && modelOverride[0] != "" {
		model = modelOverride[0]
	}
	if strings.TrimSpace(model) == "" {
		model = "gpt-5.2"
	}

	var lastErr error
	// attempts を 5 回に増やし、各リクエストにタイムアウトを設定
	for attempt := 1; attempt <= 5; attempt++ {
		ctxReq, cancel := context.WithTimeout(ctx, 60*time.Second)
		messageInput := []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]string{
					{
						"type": "input_text",
						"text": input,
					},
				},
			},
		}
		content, err := cli.callResponsesAPI(ctxReq, messageInput, model, nil, 600, true)
		if err != nil && strings.Contains(err.Error(), "empty response from responses api") {
			content, err = cli.callResponsesAPI(ctxReq, messageInput, model, nil, 600, false)
		}
		if err != nil && strings.Contains(err.Error(), "empty response from responses api") {
			content, err = cli.callResponsesAPI(ctxReq, input, model, nil, 600, false)
		}
		if err != nil && strings.Contains(err.Error(), "max_output_tokens") {
			content, err = cli.callResponsesAPI(ctxReq, messageInput, model, nil, 1200, true)
		}
		cancel()

		if err == nil && strings.TrimSpace(content) != "" {
			return strings.TrimSpace(content), nil
		}
		if err == nil {
			lastErr = errors.New("empty response from model")
			println("OpenAI API empty content (attempt", attempt, ")")
		} else {
			lastErr = err
			println("OpenAI API error (attempt", attempt, "):", err.Error())
		}

		// 指数バックオフ + ジッター
		backoff := time.Duration(1<<attempt) * time.Second
		jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
		time.Sleep(backoff + jitter)
	}

	if lastErr == nil {
		lastErr = errors.New("no response from model")
	}
	return "", lastErr
}

// ResponsesWithTemperature は system と user プロンプトを分けて、温度パラメータ付きでリクエストします
func (cli *Client) ResponsesWithTemperature(ctx context.Context, systemPrompt, userPrompt string, temperature float32, modelOverride ...string) (string, error) {
	if cli == nil || cli.c == nil {
		return "", errors.New("openai client is nil")
	}

	model := cli.DefaultModel
	if len(modelOverride) > 0 && modelOverride[0] != "" {
		model = modelOverride[0]
	}
	if strings.TrimSpace(model) == "" {
		model = "gpt-5.2"
	}

	var lastErr error
	for attempt := 1; attempt <= 5; attempt++ {
		ctxReq, cancel := context.WithTimeout(ctx, 60*time.Second)
		messageInput := []map[string]interface{}{
			{
				"role": "system",
				"content": []map[string]string{
					{
						"type": "input_text",
						"text": systemPrompt,
					},
				},
			},
			{
				"role": "user",
				"content": []map[string]string{
					{
						"type": "input_text",
						"text": userPrompt,
					},
				},
			},
		}
		content, err := cli.callResponsesAPI(ctxReq, messageInput, model, &temperature, 100, true)
		if err != nil && strings.Contains(err.Error(), "empty response from responses api") {
			content, err = cli.callResponsesAPI(ctxReq, messageInput, model, &temperature, 100, false)
		}
		if err != nil && strings.Contains(err.Error(), "empty response from responses api") {
			combinedPrompt := strings.TrimSpace(systemPrompt)
			if combinedPrompt != "" {
				combinedPrompt += "\n\n"
			}
			combinedPrompt += userPrompt
			content, err = cli.callResponsesAPI(ctxReq, combinedPrompt, model, &temperature, 100, false)
		}
		if err != nil && strings.Contains(err.Error(), "max_output_tokens") {
			content, err = cli.callResponsesAPI(ctxReq, messageInput, model, &temperature, 200, true)
		}
		cancel()

		if err == nil && strings.TrimSpace(content) != "" {
			return strings.TrimSpace(content), nil
		}
		if err == nil {
			lastErr = errors.New("empty response from model")
			println("OpenAI API empty content (attempt", attempt, ")")
		} else {
			lastErr = err
			println("OpenAI API error (attempt", attempt, "):", err.Error())
		}

		backoff := time.Duration(1<<attempt) * time.Second
		jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
		time.Sleep(backoff + jitter)
	}

	if lastErr == nil {
		lastErr = errors.New("no response from model")
	}
	return "", lastErr
}
