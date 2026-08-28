package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"novelstudio/internal/airun"
	"novelstudio/internal/document"
	"novelstudio/internal/generation"
	"novelstudio/internal/httpapi"
	"novelstudio/internal/knowledge"
	"novelstudio/internal/llm"
	mysqlstore "novelstudio/internal/persistence/mysql"
	"novelstudio/internal/project"
	"novelstudio/internal/promptcatalog"
	"novelstudio/internal/qualityhistory"
	"novelstudio/internal/task"
	"novelstudio/internal/validation"
)

func main() {
	addr := env("HTTP_ADDR", ":8080")
	projectStore, documentStore, knowledgeStore, runRecorder, qualityStore, taskManager, closeStore := stores()
	defer closeStore()
	if timeout, err := time.ParseDuration(env("AI_TASK_TIMEOUT", "15m")); err == nil && timeout > 0 {
		taskManager.SetTimeout(timeout)
	} else {
		slog.Warn("invalid AI_TASK_TIMEOUT; using 15m", "value", os.Getenv("AI_TASK_TIMEOUT"))
	}
	pipeline := validationPipeline(runRecorder)
	generator := generationService(runRecorder)
	handler := httpapi.NewWithRuntime(projectStore, documentStore, knowledgeStore, pipeline, generator, taskManager, runRecorder, qualityStore, slog.Default())

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		// SSE task events may remain open for the lifetime of an AI task.
		// Individual model calls enforce their own timeout.
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("AI Content Studio API started", "addr", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func generationService(recorder airun.Recorder) *generation.Service {
	baseURL := strings.TrimSpace(os.Getenv("LLM_BASE_URL"))
	if baseURL == "" {
		return nil
	}
	provider, err := llm.NewOpenAICompatible(llm.OpenAICompatibleConfig{BaseURL: baseURL, APIKey: os.Getenv("LLM_API_KEY")})
	if err != nil {
		slog.Error("invalid generation model configuration", "error", err)
		return nil
	}
	writerModel := strings.TrimSpace(os.Getenv("WRITER_MODEL"))
	models := map[generation.Operation]string{
		generation.OperationPlan:    first(strings.TrimSpace(os.Getenv("PLANNER_MODEL")), writerModel),
		generation.OperationOutline: first(strings.TrimSpace(os.Getenv("OUTLINER_MODEL")), writerModel),
		generation.OperationWrite:   writerModel,
		generation.OperationPolish:  first(strings.TrimSpace(os.Getenv("POLISHER_MODEL")), writerModel),
		generation.OperationRepair:  first(strings.TrimSpace(os.Getenv("REPAIR_MODEL")), writerModel),
		generation.OperationExtract: first(strings.TrimSpace(os.Getenv("EXTRACTOR_MODEL")), writerModel),
	}
	configured := false
	for _, model := range models {
		configured = configured || model != ""
	}
	if !configured {
		slog.Warn("content generation disabled; configure WRITER_MODEL or role-specific models")
		return nil
	}
	return &generation.Service{ProviderName: "openai-compatible", Provider: provider, Models: models, Recorder: recorder, Prompts: promptcatalog.Catalog{Dir: os.Getenv("PROMPT_DIR")}}
}

func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func stores() (project.Store, document.Store, knowledge.Store, airun.Recorder, qualityhistory.Store, *task.Manager, func()) {
	dsn := strings.TrimSpace(os.Getenv("MYSQL_DSN"))
	if dsn == "" {
		slog.Warn("MYSQL_DSN is empty; using volatile in-memory storage")
		return project.NewMemoryStore(), document.NewMemoryStore(), knowledge.NewMemoryStore(), airun.NewMemoryRecorder(), qualityhistory.NewMemoryStore(), task.NewManager(), func() {}
	}
	db, err := mysqlstore.Open(context.Background(), dsn)
	if err != nil {
		slog.Error("cannot connect to MySQL", "error", err)
		os.Exit(1)
	}
	if err := mysqlstore.Migrate(context.Background(), db); err != nil {
		db.Close()
		slog.Error("cannot migrate MySQL", "error", err)
		os.Exit(1)
	}
	slog.Info("MySQL persistence enabled")
	return mysqlstore.ProjectStore{DB: db}, mysqlstore.DocumentStore{DB: db}, mysqlstore.KnowledgeStore{DB: db}, mysqlstore.AIRunRecorder{DB: db}, mysqlstore.QualityStore{DB: db}, task.NewManagerWithRepository(mysqlstore.TaskRepository{DB: db}), func() { _ = db.Close() }
}

func validationPipeline(recorder airun.Recorder) *validation.Pipeline {
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
		pipeline.Validators = append(pipeline.Validators, validation.NamedReviewer{Name: fmt.Sprintf("VALIDATOR_%d", index+1), Reviewer: validation.ModelReviewer{ProviderName: "openai-compatible", Provider: provider, Model: model, Role: "validator", Recorder: recorder}})
	}
	if judgeModel := strings.TrimSpace(os.Getenv("JUDGE_MODEL")); judgeModel != "" {
		pipeline.Judge = &validation.NamedReviewer{Name: "JUDGE", Reviewer: validation.ModelReviewer{ProviderName: "openai-compatible", Provider: provider, Model: judgeModel, Role: "judge", Recorder: recorder}}
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
