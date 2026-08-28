package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"novelstudio/internal/document"
	"novelstudio/internal/knowledge"
	"novelstudio/internal/project"
	"novelstudio/internal/validation"
)

type API struct {
	store     project.Store
	docs      document.Store
	knowledge knowledge.Store
	pipeline  *validation.Pipeline
	logger    *slog.Logger
}

func New(store project.Store, logger *slog.Logger) http.Handler {
	return NewWithStores(store, document.NewMemoryStore(), knowledge.NewMemoryStore(), nil, logger)
}

func NewWithPipeline(store project.Store, pipeline *validation.Pipeline, logger *slog.Logger) http.Handler {
	return NewWithStores(store, document.NewMemoryStore(), knowledge.NewMemoryStore(), pipeline, logger)
}

func NewWithStores(store project.Store, docs document.Store, knowledgeStore knowledge.Store, pipeline *validation.Pipeline, logger *slog.Logger) http.Handler {
	a := &API{store: store, docs: docs, knowledge: knowledgeStore, pipeline: pipeline, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /api/v1/project-types", a.projectTypes)
	mux.HandleFunc("GET /api/v1/projects", a.listProjects)
	mux.HandleFunc("POST /api/v1/projects", a.createProject)
	mux.HandleFunc("GET /api/v1/projects/{id}", a.getProject)
	mux.HandleFunc("DELETE /api/v1/projects/{id}", a.deleteProject)
	mux.HandleFunc("GET /api/v1/projects/{id}/tree", a.getTree)
	mux.HandleFunc("GET /api/v1/projects/{id}/documents", a.listDocuments)
	mux.HandleFunc("POST /api/v1/projects/{id}/documents", a.createDocument)
	mux.HandleFunc("GET /api/v1/documents/{id}", a.getDocument)
	mux.HandleFunc("GET /api/v1/documents/{id}/versions", a.listVersions)
	mux.HandleFunc("POST /api/v1/documents/{id}/versions", a.createVersion)
	mux.HandleFunc("POST /api/v1/documents/{id}/versions/{versionId}/restore", a.restoreVersion)
	mux.HandleFunc("GET /api/v1/projects/{id}/knowledge/sources", a.listKnowledgeSources)
	mux.HandleFunc("POST /api/v1/projects/{id}/knowledge/sources", a.createKnowledgeSource)
	mux.HandleFunc("GET /api/v1/projects/{id}/knowledge/search", a.searchKnowledge)
	mux.HandleFunc("GET /api/v1/models/status", a.modelStatus)
	mux.HandleFunc("POST /api/v1/projects/{id}/validate", a.validateText)
	return recoverer(logger, requestLogger(logger, cors(mux)))
}

func (a *API) modelStatus(w http.ResponseWriter, _ *http.Request) {
	configured := a.pipeline != nil && len(a.pipeline.Validators) > 0
	validatorCount := 0
	judgeConfigured := false
	if a.pipeline != nil {
		validatorCount = len(a.pipeline.Validators)
		judgeConfigured = a.pipeline.Judge != nil
	}
	writeJSON(w, http.StatusOK, map[string]any{"configured": configured, "validatorCount": validatorCount, "judgeConfigured": judgeConfigured})
}

func (a *API) validateText(w http.ResponseWriter, r *http.Request) {
	if a.pipeline == nil {
		writeError(w, http.StatusServiceUnavailable, "MODEL_NOT_CONFIGURED", "validation models are not configured")
		return
	}
	projectID := r.PathValue("id")
	if _, err := a.store.Get(r.Context(), projectID); err != nil {
		a.handleStoreError(w, err)
		return
	}
	var input struct {
		Text           string   `json:"text"`
		Task           string   `json:"task"`
		KnowledgeQuery string   `json:"knowledgeQuery"`
		Dimensions     []string `json:"dimensions"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	request := validation.ReviewRequest{Text: input.Text, Task: input.Task, Dimensions: input.Dimensions}
	if strings.TrimSpace(input.KnowledgeQuery) != "" {
		hits, err := a.knowledge.Search(r.Context(), projectID, input.KnowledgeQuery, 8)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_QUERY", err.Error())
			return
		}
		for _, hit := range hits {
			request.Evidence = append(request.Evidence, validation.Evidence{ID: hit.Chunk.ID, Source: hit.Source.Name, Authority: string(hit.Source.Authority), Content: hit.Chunk.Content})
		}
	}
	result, err := a.pipeline.Validate(r.Context(), request)
	if err != nil {
		a.logger.Error("validation pipeline failed", "projectId", projectID, "error", err)
		writeError(w, http.StatusBadGateway, "VALIDATION_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "ai-content-studio"})
}

func (a *API) projectTypes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, []map[string]string{
		{"value": string(project.TypeNovel), "label": "小说"},
		{"value": string(project.TypeMovieCommentary), "label": "电影解说"},
		{"value": string(project.TypeTechnicalDocument), "label": "技术文档"},
	})
}

func (a *API) listProjects(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func (a *API) createProject(w http.ResponseWriter, r *http.Request) {
	var input project.CreateInput
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON request")
		return
	}
	item, err := a.store.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) getProject(w http.ResponseWriter, r *http.Request) {
	item, err := a.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		a.handleStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) deleteProject(w http.ResponseWriter, r *http.Request) {
	if err := a.store.Delete(r.Context(), r.PathValue("id")); err != nil {
		a.handleStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) getTree(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.Tree(r.Context(), r.PathValue("id"))
	if err != nil {
		a.handleStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) listDocuments(w http.ResponseWriter, r *http.Request) {
	if _, err := a.store.Get(r.Context(), r.PathValue("id")); err != nil {
		a.handleStoreError(w, err)
		return
	}
	items, err := a.docs.List(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func (a *API) createDocument(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if _, err := a.store.Get(r.Context(), projectID); err != nil {
		a.handleStoreError(w, err)
		return
	}
	var input document.CreateInput
	if !decodeJSON(w, r, &input) {
		return
	}
	input.ProjectID = projectID
	item, version, err := a.docs.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"document": item, "version": version})
}

func (a *API) getDocument(w http.ResponseWriter, r *http.Request) {
	item, err := a.docs.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		a.handleDocumentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) listVersions(w http.ResponseWriter, r *http.Request) {
	items, err := a.docs.Versions(r.Context(), r.PathValue("id"))
	if err != nil {
		a.handleDocumentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func (a *API) createVersion(w http.ResponseWriter, r *http.Request) {
	var input document.CreateVersionInput
	if !decodeJSON(w, r, &input) {
		return
	}
	item, err := a.docs.CreateVersion(r.Context(), r.PathValue("id"), input)
	if err != nil {
		a.handleDocumentError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) restoreVersion(w http.ResponseWriter, r *http.Request) {
	item, err := a.docs.Restore(r.Context(), r.PathValue("id"), r.PathValue("versionId"))
	if err != nil {
		a.handleDocumentError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) listKnowledgeSources(w http.ResponseWriter, r *http.Request) {
	items, err := a.knowledge.ListSources(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func (a *API) createKnowledgeSource(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if _, err := a.store.Get(r.Context(), projectID); err != nil {
		a.handleStoreError(w, err)
		return
	}
	var input knowledge.CreateSourceInput
	if !decodeJSON(w, r, &input) {
		return
	}
	input.ProjectID = projectID
	source, chunks, err := a.knowledge.CreateSource(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"source": source, "chunks": chunks})
}

func (a *API) searchKnowledge(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := a.knowledge.Search(r.Context(), r.PathValue("id"), r.URL.Query().Get("q"), limit)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func (a *API) handleStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, project.ErrNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
}

func (a *API) handleDocumentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, document.ErrNotFound), errors.Is(err, document.ErrVersionFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, document.ErrNoChange):
		writeError(w, http.StatusConflict, "NO_CHANGE", err.Error())
	default:
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", err.Error())
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON request")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:5173" || strings.HasPrefix(origin, "http://127.0.0.1:5173") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("http request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func recoverer(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if value := recover(); value != nil {
				logger.Error("panic recovered", "value", value)
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
