package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAICompatibleConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

type OpenAICompatible struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewOpenAICompatible(config OpenAICompatibleConfig) (*OpenAICompatible, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		return nil, errors.New("LLM base URL is required")
	}
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &OpenAICompatible{baseURL: baseURL, apiKey: config.APIKey, client: &http.Client{Timeout: timeout}}, nil
}

func (p *OpenAICompatible) Generate(ctx context.Context, input GenerateRequest) (GenerateResponse, error) {
	if strings.TrimSpace(input.Model) == "" {
		return GenerateResponse{}, errors.New("model is required")
	}
	body := map[string]any{
		"model":       input.Model,
		"messages":    input.Messages,
		"temperature": input.Temperature,
	}
	if input.MaxTokens > 0 {
		body["max_tokens"] = input.MaxTokens
	}
	if input.ResponseSchema != nil {
		body["response_format"] = map[string]any{
			"type":        "json_schema",
			"json_schema": map[string]any{"name": "structured_response", "strict": true, "schema": input.ResponseSchema},
		}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("encode LLM request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("create LLM request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	response, err := p.client.Do(req)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("call LLM provider: %w", err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("read LLM response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return GenerateResponse{}, fmt.Errorf("LLM provider returned %d: %s", response.StatusCode, safeError(raw))
	}
	var decoded struct {
		ID      string `json:"id"`
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return GenerateResponse{}, fmt.Errorf("decode LLM response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return GenerateResponse{}, errors.New("LLM response contains no choices")
	}
	return GenerateResponse{Content: decoded.Choices[0].Message.Content, InputTokens: decoded.Usage.PromptTokens, OutputTokens: decoded.Usage.CompletionTokens, RequestID: decoded.ID}, nil
}

func (p *OpenAICompatible) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	response, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("LLM health check returned %d", response.StatusCode)
	}
	return nil
}

func safeError(raw []byte) string {
	text := strings.TrimSpace(string(raw))
	if len(text) > 500 {
		text = text[:500]
	}
	return text
}
