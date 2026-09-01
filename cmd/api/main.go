package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"novelstudio/internal/airun"
	"novelstudio/internal/auth"
	"novelstudio/internal/document"
	"novelstudio/internal/generation"
	"novelstudio/internal/httpapi"
	"novelstudio/internal/knowledge"
	"novelstudio/internal/llm"
	"novelstudio/internal/modelconfig"
	mysqlstore "novelstudio/internal/persistence/mysql"
	"novelstudio/internal/project"
	"novelstudio/internal/promptcatalog"
	"novelstudio/internal/qualityhistory"
	"novelstudio/internal/task"
	"novelstudio/internal/validation"
)

func main() {
	loadEnvFile(".env")
	addr := env("HTTP_ADDR", ":8080")
	applyLocalModelConfig()
	projectStore, documentStore, knowledgeStore, runRecorder, qualityStore, authStore, taskManager, closeStore := stores()
	defer closeStore()
	if timeout, err := time.ParseDuration(env("AI_TASK_TIMEOUT", "15m")); err == nil && timeout > 0 {
		taskManager.SetTimeout(timeout)
	} else {
		slog.Warn("invalid AI_TASK_TIMEOUT; using 15m", "value", os.Getenv("AI_TASK_TIMEOUT"))
	}
	pipeline := validationPipeline(runRecorder)
	generator := generationService(runRecorder)
	handler := httpapi.NewWithRuntimeAuth(projectStore, documentStore, knowledgeStore, pipeline, generator, taskManager, runRecorder, qualityStore, authStore, slog.Default())

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

func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("cannot read environment file", "path", path, "error", err)
		}
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	loaded := 0
	for scanner.Scan() {
		lineNumber++
		key, value, ok, err := parseEnvLine(scanner.Text())
		if err != nil {
			slog.Warn("invalid environment file entry", "path", path, "line", lineNumber, "error", err)
			continue
		}
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			slog.Warn("cannot set environment variable", "path", path, "line", lineNumber, "key", key, "error", err)
			continue
		}
		loaded++
	}
	if err := scanner.Err(); err != nil {
		slog.Warn("cannot read environment file", "path", path, "error", err)
		return
	}
	slog.Info("environment file loaded", "path", path, "variables", loaded)
}

func parseEnvLine(line string) (string, string, bool, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false, nil
	}
	line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
	key, value, found := strings.Cut(line, "=")
	if !found {
		return "", "", false, fmt.Errorf("missing =")
	}
	key = strings.TrimSpace(key)
	if key == "" || strings.ContainsAny(key, " \t") {
		return "", "", false, fmt.Errorf("invalid variable name")
	}
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", "", false, fmt.Errorf("invalid quoted value: %w", err)
		}
		value = unquoted
	} else if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		value = value[1 : len(value)-1]
	}
	return key, value, true, nil
}

func applyLocalModelConfig() {
	path := strings.TrimSpace(os.Getenv("MODEL_CONFIG_PATH"))
	if path == "" {
		path = ".local/model-config.json"
	}
	name, provider, err := (modelconfig.Store{Path: path}).Active()
	if err != nil {
		slog.Warn("cannot read local model config", "error", err)
		return
	}
	if !provider.Enabled {
		return
	}
	_ = os.Setenv("LLM_BASE_URL", provider.BaseURL)
	_ = os.Setenv("LLM_API_KEY", provider.APIKey)
	_ = os.Setenv("VALIDATOR_MODELS", provider.Model)
	_ = os.Setenv("JUDGE_MODEL", provider.Model)
	for _, key := range []string{"PLANNER_MODEL", "OUTLINER_MODEL", "WRITER_MODEL", "POLISHER_MODEL", "REPAIR_MODEL", "EXTRACTOR_MODEL"} {
		_ = os.Setenv(key, provider.Model)
	}
	slog.Info("local model configuration enabled", "provider", name, "model", provider.Model, "path", path)
}

func generationService(recorder airun.Recorder) *generation.Service {
	baseURL := strings.TrimSpace(os.Getenv("LLM_BASE_URL"))
	if baseURL == "" {
		return nil
	}
	provider, err := llm.NewOpenAICompatible(llm.OpenAICompatibleConfig{BaseURL: baseURL, APIKey: os.Getenv("LLM_API_KEY"), Timeout: modelRequestTimeout()})
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
		generation.OperationMemory:  first(strings.TrimSpace(os.Getenv("EXTRACTOR_MODEL")), writerModel),
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

func modelRequestTimeout() time.Duration {
	timeout, err := time.ParseDuration(env("LLM_REQUEST_TIMEOUT", "5m"))
	if err != nil || timeout <= 0 {
		slog.Warn("invalid LLM_REQUEST_TIMEOUT; using 5m", "value", os.Getenv("LLM_REQUEST_TIMEOUT"))
		return 5 * time.Minute
	}
	return timeout
}

func stores() (project.Store, document.Store, knowledge.Store, airun.Recorder, qualityhistory.Store, auth.Store, *task.Manager, func()) {
	dsn := strings.TrimSpace(os.Getenv("MYSQL_DSN"))
	if dsn == "" {
		slog.Warn("MYSQL_DSN is empty; using volatile in-memory storage")
		return project.NewMemoryStore(), document.NewMemoryStore(), knowledge.NewMemoryStore(), airun.NewMemoryRecorder(), qualityhistory.NewMemoryStore(), auth.NewMemoryStore(), task.NewManager(), func() {}
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
	return mysqlstore.ProjectStore{DB: db}, mysqlstore.DocumentStore{DB: db}, mysqlstore.KnowledgeStore{DB: db}, mysqlstore.AIRunRecorder{DB: db}, mysqlstore.QualityStore{DB: db}, mysqlstore.AuthStore{DB: db}, task.NewManagerWithRepository(mysqlstore.TaskRepository{DB: db}), func() { _ = db.Close() }
}

func validationPipeline(recorder airun.Recorder) *validation.Pipeline {
	baseURL := strings.TrimSpace(os.Getenv("LLM_BASE_URL"))
	models := split(os.Getenv("VALIDATOR_MODELS"))
	if baseURL == "" || len(models) == 0 {
		slog.Warn("model validation disabled; configure LLM_BASE_URL and VALIDATOR_MODELS")
		return nil
	}
	provider, err := llm.NewOpenAICompatible(llm.OpenAICompatibleConfig{BaseURL: baseURL, APIKey: os.Getenv("LLM_API_KEY"), Timeout: modelRequestTimeout()})
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
