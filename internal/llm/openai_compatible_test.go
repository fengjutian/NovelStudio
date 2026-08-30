package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"novelstudio/internal/llm"
)

func TestOpenAICompatibleGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("unexpected request: %s auth=%s", r.URL.Path, r.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["response_format"] == nil {
			t.Fatal("structured response format is missing")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"req-1","choices":[{"message":{"role":"assistant","content":"<think>internal reasoning</think>\n{\"score\":92}"}}],"usage":{"prompt_tokens":10,"completion_tokens":4}}`))
	}))
	defer server.Close()
	provider, err := llm.NewOpenAICompatible(llm.OpenAICompatibleConfig{BaseURL: server.URL + "/v1", APIKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Generate(context.Background(), llm.GenerateRequest{Model: "test", Messages: []llm.Message{{Role: "user", Content: "validate"}}, ResponseSchema: map[string]any{"type": "object"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.RequestID != "req-1" || result.InputTokens != 10 {
		t.Fatalf("unexpected response: %#v", result)
	}
	if result.Content != `{"score":92}` {
		t.Fatalf("reasoning content was not removed: %q", result.Content)
	}
}
