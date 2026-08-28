package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"novelstudio/internal/httpapi"
	"novelstudio/internal/llm"
	"novelstudio/internal/project"
	"novelstudio/internal/validation"
)

func main() {
	addr := env("HTTP_ADDR", ":8080")
	store := project.NewMemoryStore()
	pipeline := validationPipeline()
	handler := httpapi.NewWithPipeline(store, pipeline, slog.Default())

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	slog.Info("AI Content Studio API started", "addr", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func validationPipeline() *validation.Pipeline {
	baseURL := strings.TrimSpace(os.Getenv("LLM_BASE_URL"))
	models := split(os.Getenv("VALIDATOR_MODELS"))
	if baseURL == "" || len(models) == 0 {
		slog.Warn("model validation disabled; configure LLM_BASE_URL and VALIDATOR_MODELS")
		return nil
	}
	provider, err := llm.NewOpenAICompatible(llm.OpenAICompatibleConfig{BaseURL: baseURL, APIKey: os.Getenv("LLM_API_KEY")})
	if err != nil {
		slog.Error("invalid LLM configuration", "error", err)
		return nil
	}
	pipeline := &validation.Pipeline{}
	for index, model := range models {
		pipeline.Validators = append(pipeline.Validators, validation.NamedReviewer{Name: fmt.Sprintf("VALIDATOR_%d", index+1), Reviewer: validation.ModelReviewer{ProviderName: "openai-compatible", Provider: provider, Model: model, Role: "validator"}})
	}
	if judgeModel := strings.TrimSpace(os.Getenv("JUDGE_MODEL")); judgeModel != "" {
		pipeline.Judge = &validation.NamedReviewer{Name: "JUDGE", Reviewer: validation.ModelReviewer{ProviderName: "openai-compatible", Provider: provider, Model: judgeModel, Role: "judge"}}
	}
	return pipeline
}

func split(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
