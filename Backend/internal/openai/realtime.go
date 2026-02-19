package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

type RealtimeSessionRequest struct {
	Type             string                 `json:"type,omitempty"`
	Model            string                 `json:"model"`
	OutputModalities []string               `json:"output_modalities,omitempty"`
	Instructions     string                 `json:"instructions,omitempty"`
	Audio            map[string]interface{} `json:"audio,omitempty"`
	MaxOutputTokens  interface{}            `json:"max_output_tokens,omitempty"`
}

type RealtimeClientSecretRequest struct {
	Session RealtimeSessionRequest `json:"session"`
}

type RealtimeClientSecretResponse struct {
	Value     string `json:"value"`
	ExpiresAt int64  `json:"expires_at"`
}

func (cli *Client) CreateRealtimeClientSecret(ctx context.Context, session RealtimeSessionRequest) (*RealtimeClientSecretResponse, error) {
	if cli.apiKey == "" {
		return nil, errors.New("openai api key is not set")
	}
	payload := RealtimeClientSecretRequest{Session: session}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/realtime/client_secrets", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cli.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New(string(respBody))
	}

	var parsed RealtimeClientSecretResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if parsed.Value == "" {
		return nil, errors.New("missing value in realtime client secret response")
	}
	return &parsed, nil
}
