package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"novelstudio/internal/airun"
	"novelstudio/internal/document"
	"novelstudio/internal/generation"
	"novelstudio/internal/knowledge"
	"novelstudio/internal/modelconfig"
	"novelstudio/internal/project"
	"novelstudio/internal/qualityhistory"
	"novelstudio/internal/task"
	"novelstudio/internal/validation"
	"novelstudio/internal/workflow"
)

type API struct {
	store     project.Store
	docs      document.Store
	knowledge knowledge.Store
	pipeline  *validation.Pipeline
	generator *generation.Service
	tasks     *task.Manager
	logger    *slog.Logger
	runs      airun.Recorder
	quality   qualityhistory.Store
	modelConfig modelconfig.Store
}

func New(store project.Store, logger *slog.Logger) http.Handler {
	return NewWithStores(store, document.NewMemoryStore(), knowledge.NewMemoryStore(), nil, logger)
}

func NewWithPipeline(store project.Store, pipeline *validation.Pipeline, logger *slog.Logger) http.Handler {
	return NewWithStores(store, document.NewMemoryStore(), knowledge.NewMemoryStore(), pipeline, logger)
}

func NewWithStores(store project.Store, docs document.Store, knowledgeStore knowledge.Store, pipeline *validation.Pipeline, logger *slog.Logger) http.Handler {
	return NewWithServices(store, docs, knowledgeStore, pipeline, nil, logger)
}

func NewWithServices(store project.Store, docs document.Store, knowledgeStore knowledge.Store, pipeline *validation.Pipeline, generator *generation.Service, logger *slog.Logger) http.Handler {
	return NewWithRuntime(store, docs, knowledgeStore, pipeline, generator, task.NewManager(), nil, nil, logger)
}
func NewWithRuntime(store project.Store, docs document.Store, knowledgeStore knowledge.Store, pipeline *validation.Pipeline, generator *generation.Service, tasks *task.Manager, runs airun.Recorder, quality qualityhistory.Store, logger *slog.Logger) http.Handler {
	if tasks == nil {
		tasks = task.NewManager()
	}
	configPath:=strings.TrimSpace(os.Getenv("MODEL_CONFIG_PATH"));if configPath==""{configPath=filepath.Join(".local","model-config.json")}
	a := &API{store: store, docs: docs, knowledge: knowledgeStore, pipeline: pipeline, generator: generator, tasks: tasks, runs: runs, quality: quality, modelConfig:modelconfig.Store{Path:configPath}, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /api/v1/project-types", a.projectTypes)
	mux.HandleFunc("GET /api/v1/projects", a.listProjects)
	mux.HandleFunc("GET /api/v1/dashboard/stats", a.dashboardStats)
	mux.HandleFunc("POST /api/v1/projects", a.createProject)
	mux.HandleFunc("GET /api/v1/projects/{id}", a.getProject)
	mux.HandleFunc("DELETE /api/v1/projects/{id}", a.deleteProject)
	mux.HandleFunc("GET /api/v1/projects/{id}/export.md", a.exportProjectMarkdown)
	mux.HandleFunc("GET /api/v1/projects/{id}/tree", a.getTree)
	mux.HandleFunc("POST /api/v1/projects/{id}/nodes", a.createNode)
	mux.HandleFunc("POST /api/v1/projects/{id}/outline-import", a.importOutline)
	mux.HandleFunc("PUT /api/v1/nodes/{id}", a.updateNode)
	mux.HandleFunc("DELETE /api/v1/nodes/{id}", a.deleteNode)
	mux.HandleFunc("GET /api/v1/projects/{id}/documents", a.listDocuments)
	mux.HandleFunc("POST /api/v1/projects/{id}/documents", a.createDocument)
	mux.HandleFunc("GET /api/v1/documents/{id}", a.getDocument)
	mux.HandleFunc("GET /api/v1/documents/{id}/versions", a.listVersions)
	mux.HandleFunc("GET /api/v1/documents/{id}/diff", a.diffVersions)
	mux.HandleFunc("POST /api/v1/documents/{id}/versions", a.createVersion)
	mux.HandleFunc("POST /api/v1/documents/{id}/versions/{versionId}/restore", a.restoreVersion)
	mux.HandleFunc("GET /api/v1/projects/{id}/knowledge/sources", a.listKnowledgeSources)
	mux.HandleFunc("POST /api/v1/projects/{id}/knowledge/sources", a.createKnowledgeSource)
	mux.HandleFunc("POST /api/v1/projects/{id}/knowledge/files", a.uploadKnowledgeFile)
	mux.HandleFunc("GET /api/v1/projects/{id}/knowledge/search", a.searchKnowledge)
	mux.HandleFunc("GET /api/v1/projects/{id}/knowledge/facts", a.listFacts)
	mux.HandleFunc("GET /api/v1/projects/{id}/memories", a.listMemories)
	mux.HandleFunc("POST /api/v1/projects/{id}/memories", a.createMemory)
	mux.HandleFunc("DELETE /api/v1/memories/{id}", a.deleteMemory)
	mux.HandleFunc("POST /api/v1/projects/{id}/fact-extraction-tasks", a.createFactExtractionTask)
	mux.HandleFunc("PUT /api/v1/facts/{id}/status", a.updateFactStatus)
	mux.HandleFunc("GET /api/v1/models/status", a.modelStatus)
	mux.HandleFunc("GET /api/v1/models/local-config", a.getLocalModelConfig)
	mux.HandleFunc("PUT /api/v1/models/local-config", a.updateLocalModelConfig)
	mux.HandleFunc("GET /api/v1/projects/{id}/ai-runs", a.listAIRuns)
	mux.HandleFunc("GET /api/v1/projects/{id}/quality-results", a.listQualityResults)
	mux.HandleFunc("POST /api/v1/projects/{id}/validate", a.validateText)
	mux.HandleFunc("GET /api/v1/projects/{id}/tasks", a.listTasks)
	mux.HandleFunc("GET /api/v1/tasks", a.listAllTasks)
	mux.HandleFunc("POST /api/v1/projects/{id}/validation-tasks", a.createValidationTask)
	mux.HandleFunc("POST /api/v1/projects/{id}/generation-tasks", a.createGenerationTask)
	mux.HandleFunc("POST /api/v1/projects/{id}/quality-generation-tasks", a.createQualityGenerationTask)
	mux.HandleFunc("POST /api/v1/projects/{id}/batch-generation-tasks", a.createBatchGenerationTask)
	mux.HandleFunc("GET /api/v1/tasks/{id}", a.getTask)
	mux.HandleFunc("POST /api/v1/tasks/{id}/cancel", a.cancelTask)
	mux.HandleFunc("POST /api/v1/tasks/{id}/retry", a.retryTask)
	mux.HandleFunc("GET /api/v1/tasks/{id}/events", a.taskEvents)
	return recoverer(logger, requestLogger(logger, cors(mux)))
}

var htmlTags = regexp.MustCompile(`<[^>]+>`)

func (a *API) uploadKnowledgeFile(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if _, err := a.store.Get(r.Context(), projectID); err != nil {
		a.handleStoreError(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_FILE", "file exceeds 10 MB or multipart form is invalid")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_FILE", "file field is required")
		return
	}
	defer file.Close()
	extension := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".txt": true, ".md": true, ".markdown": true, ".json": true, ".csv": true, ".html": true, ".htm": true}
	if !allowed[extension] {
		writeError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_FILE", "supported types: txt, md, json, csv, html")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(file, 10<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_FILE", err.Error())
		return
	}
	content := string(raw)
	if extension == ".html" || extension == ".htm" {
		content = html.UnescapeString(htmlTags.ReplaceAllString(content, " "))
	}
	authority := knowledge.Authority(strings.ToUpper(r.FormValue("authority")))
	if authority == "" {
		authority = knowledge.AuthorityReference
	}
	source, chunks, err := a.knowledge.CreateSource(r.Context(), knowledge.CreateSourceInput{ProjectID: projectID, Name: header.Filename, SourceType: strings.TrimPrefix(strings.ToUpper(extension), "."), Version: r.FormValue("version"), Authority: authority, Content: content})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "IMPORT_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"source": source, "chunkCount": len(chunks)})
}

func (a *API) createBatchGenerationTask(w http.ResponseWriter, r *http.Request) {
	if a.generator == nil || !a.generator.Configured(generation.OperationWrite) {
		writeError(w, http.StatusServiceUnavailable, "MODEL_NOT_CONFIGURED", "writer model is not configured")
		return
	}
	projectItem, err := a.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		a.handleStoreError(w, err)
		return
	}
	var input struct {
		NodeIDs        []string `json:"nodeIds"`
		Instruction    string   `json:"instruction"`
		KnowledgeQuery string   `json:"knowledgeQuery"`
		WindowSize     int      `json:"windowSize"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if len(input.NodeIDs) == 0 || strings.TrimSpace(input.Instruction) == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "nodeIds and instruction are required")
		return
	}
	if input.WindowSize < 1 {
		input.WindowSize = 2
	}
	if input.WindowSize > 3 {
		input.WindowSize = 3
	}
	tree, err := a.store.Tree(r.Context(), projectItem.ID)
	if err != nil {
		a.handleStoreError(w, err)
		return
	}
	byID := map[string]project.ContentNode{}
	for _, node := range tree {
		byID[node.ID] = node
	}
	nodes := make([]project.ContentNode, 0, len(input.NodeIDs))
	for _, id := range input.NodeIDs {
		node, ok := byID[id]
		if !ok {
			writeError(w, http.StatusUnprocessableEntity, "INVALID_NODE", "content node not found: "+id)
			return
		}
		nodes = append(nodes, node)
	}
	evidence := a.memoryEvidence(r.Context(), projectItem.ID)
	if strings.TrimSpace(input.KnowledgeQuery) != "" {
		hits, searchErr := a.knowledge.Search(r.Context(), projectItem.ID, input.KnowledgeQuery, 8)
		if searchErr != nil {
			writeError(w, http.StatusBadRequest, "INVALID_QUERY", searchErr.Error())
			return
		}
		for _, hit := range hits {
			evidence = append(evidence, generation.Evidence{ID: hit.Chunk.ID, Source: hit.Source.Name, Authority: string(hit.Source.Authority), Content: hit.Chunk.Content})
		}
	}
	item := a.tasks.Create(projectItem.ID, "BATCH_GENERATE", func(ctx context.Context, progress func(int, string)) (any, error) {
		outputs := make([]generation.Result, len(nodes))
		semaphore := make(chan struct{}, input.WindowSize)
		errorsChannel := make(chan error, len(nodes))
		var wait sync.WaitGroup
		for index, node := range nodes {
			wait.Add(1)
			go func(index int, node project.ContentNode) {
				defer wait.Done()
				select {
				case semaphore <- struct{}{}:
				case <-ctx.Done():
					errorsChannel <- ctx.Err()
					return
				}
				defer func() { <-semaphore }()
				result, generateErr := a.generator.Generate(ctx, generation.Request{Operation: generation.OperationWrite, ProjectType: string(projectItem.Type), ProjectID: projectItem.ID, Instruction: input.Instruction + "\n当前内容节点：" + node.Title, Evidence: evidence})
				if generateErr != nil {
					errorsChannel <- generateErr
					return
				}
				outputs[index] = result
				progress(10+(index+1)*60/len(nodes), "已生成 "+node.Title)
			}(index, node)
		}
		wait.Wait()
		close(errorsChannel)
		for batchErr := range errorsChannel {
			if batchErr != nil {
				return nil, batchErr
			}
		}
		documents := make([]document.Document, 0, len(nodes))
		for index, node := range nodes {
			doc, _, saveErr := a.docs.Create(ctx, document.CreateInput{ProjectID: projectItem.ID, Title: node.Title, Content: outputs[index].Content})
			if saveErr != nil {
				return nil, saveErr
			}
			documents = append(documents, doc)
			documentID := doc.ID
			if _, updateErr := a.store.UpdateNode(ctx, node.ID, project.UpdateNodeInput{Title: node.Title, Position: node.Position, Metadata: node.Metadata, DocumentID: &documentID}); updateErr != nil {
				return nil, updateErr
			}
			progress(75+(index+1)*20/len(nodes), "已保存 "+node.Title)
		}
		return map[string]any{"documents": documents, "generations": outputs}, nil
	})
	writeJSON(w, http.StatusAccepted, item)
}
func (a *API) createNode(w http.ResponseWriter, r *http.Request) {
	var input project.CreateNodeInput
	if !decodeJSON(w, r, &input) {
		return
	}
	item, err := a.store.CreateNode(r.Context(), r.PathValue("id"), input)
	if err != nil {
		a.handleStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) importOutline(w http.ResponseWriter, r *http.Request) {
	projectItem, err := a.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		a.handleStoreError(w, err)
		return
	}
	var input struct {
		Content  string `json:"content"`
		ParentID string `json:"parentId"`
		Preview  bool   `json:"preview"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	items, err := project.ParseOutline(input.Content, projectItem.Type)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_OUTLINE", err.Error())
		return
	}
	if input.Preview {
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
		return
	}
	tree, treeErr := a.store.Tree(r.Context(), projectItem.ID)
	if treeErr != nil {
		a.handleStoreError(w, treeErr)
		return
	}
	found := input.ParentID == ""
	positions := map[string]int{}
	for _, node := range tree {
		if node.ID == input.ParentID {
			found = true
		}
		parent := ""
		if node.ParentID != nil {
			parent = *node.ParentID
		}
		if node.Position > positions[parent] {
			positions[parent] = node.Position
		}
	}
	if !found {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_PARENT", "parent node not found in project")
		return
	}
	created := make([]project.ContentNode, 0, len(items))
	parents := map[int]string{}
	for _, outlineItem := range items {
		parentID := input.ParentID
		for level := outlineItem.Level - 1; level >= 1; level-- {
			if value := parents[level]; value != "" {
				parentID = value
				break
			}
		}
		positions[parentID]++
		var parent *string
		if parentID != "" {
			value := parentID
			parent = &value
		}
		node, createErr := a.store.CreateNode(r.Context(), projectItem.ID, project.CreateNodeInput{ParentID: parent, NodeType: outlineItem.NodeType, Title: outlineItem.Title, Position: positions[parentID], Metadata: map[string]any{"outlineLevel": outlineItem.Level}})
		if createErr != nil {
			writeError(w, http.StatusUnprocessableEntity, "IMPORT_FAILED", createErr.Error())
			return
		}
		created = append(created, node)
		parents[outlineItem.Level] = node.ID
		for level := outlineItem.Level + 1; level <= 6; level++ {
			delete(parents, level)
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"items": created, "total": len(created)})
}
func (a *API) updateNode(w http.ResponseWriter, r *http.Request) {
	var input project.UpdateNodeInput
	if !decodeJSON(w, r, &input) {
		return
	}
	item, err := a.store.UpdateNode(r.Context(), r.PathValue("id"), input)
	if err != nil {
		a.handleStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (a *API) deleteNode(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DeleteNode(r.Context(), r.PathValue("id")); err != nil {
		a.handleStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) listFacts(w http.ResponseWriter, r *http.Request) {
	items, err := a.knowledge.ListFacts(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}
func (a *API) listMemories(w http.ResponseWriter, r *http.Request) {
	items, err := a.knowledge.ListMemories(r.Context(), r.PathValue("id"), strings.ToUpper(r.URL.Query().Get("type")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}
func (a *API) createMemory(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if _, err := a.store.Get(r.Context(), projectID); err != nil {
		a.handleStoreError(w, err)
		return
	}
	var input knowledge.CreateMemoryInput
	if !decodeJSON(w, r, &input) {
		return
	}
	item, err := a.knowledge.CreateMemory(r.Context(), projectID, input)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
func (a *API) deleteMemory(w http.ResponseWriter, r *http.Request) {
	if err := a.knowledge.DeleteMemory(r.Context(), r.PathValue("id")); err != nil {
		if errors.Is(err, knowledge.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) memoryEvidence(ctx context.Context, projectID string) []generation.Evidence {
	items, err := a.knowledge.ListMemories(ctx, projectID, "")
	if err != nil {
		return nil
	}
	if len(items) > 30 {
		items = items[:30]
	}
	evidence := make([]generation.Evidence, 0, len(items))
	for _, item := range items {
		evidence = append(evidence, generation.Evidence{ID: item.ID, Source: "Story Memory / " + item.Type + " / " + item.Name, Authority: "INTERNAL", Content: item.Summary})
	}
	return evidence
}
func (a *API) updateFactStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	item, err := a.knowledge.UpdateFactStatus(r.Context(), r.PathValue("id"), strings.ToUpper(input.Status))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (a *API) createFactExtractionTask(w http.ResponseWriter, r *http.Request) {
	if a.generator == nil || !a.generator.Configured(generation.OperationExtract) {
		writeError(w, http.StatusServiceUnavailable, "MODEL_NOT_CONFIGURED", "extractor model is not configured")
		return
	}
	projectItem, err := a.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		a.handleStoreError(w, err)
		return
	}
	var input struct {
		Content    string `json:"content"`
		DocumentID string `json:"documentId"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Content == "" && input.DocumentID != "" {
		versions, versionErr := a.docs.Versions(r.Context(), input.DocumentID)
		if versionErr != nil || len(versions) == 0 {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "document content not found")
			return
		}
		input.Content = versions[0].Content
	}
	if strings.TrimSpace(input.Content) == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "content or documentId is required")
		return
	}
	item := a.tasks.Create(projectItem.ID, "FACT_EXTRACT", func(ctx context.Context, progress func(int, string)) (any, error) {
		progress(20, "Extractor 正在抽取结构化事实")
		generated, generateErr := a.generator.Generate(ctx, generation.Request{Operation: generation.OperationExtract, ProjectType: string(projectItem.Type), ProjectID: projectItem.ID, Instruction: "抽取正文中的明确事实", Content: input.Content})
		if generateErr != nil {
			return nil, generateErr
		}
		var decoded struct {
			Facts []knowledge.CreateFactInput `json:"facts"`
		}
		raw := strings.TrimSpace(generated.Content)
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimSuffix(raw, "```")
		if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &decoded); err != nil {
			return nil, fmt.Errorf("decode extracted facts: %w", err)
		}
		facts, saveErr := a.knowledge.CreateFacts(ctx, projectItem.ID, decoded.Facts)
		if saveErr != nil {
			return nil, saveErr
		}
		progress(90, "事实已保存为待审核状态")
		return map[string]any{"facts": facts, "generation": generated}, nil
	})
	writeJSON(w, http.StatusAccepted, item)
}

func (a *API) createQualityGenerationTask(w http.ResponseWriter, r *http.Request) {
	if a.generator == nil || a.pipeline == nil || !a.generator.Configured(generation.OperationRepair) {
		writeError(w, http.StatusServiceUnavailable, "WORKFLOW_NOT_CONFIGURED", "writer, repair and validation models are required")
		return
	}
	projectItem, err := a.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		a.handleStoreError(w, err)
		return
	}
	var input struct {
		Instruction    string `json:"instruction"`
		Title          string `json:"title"`
		KnowledgeQuery string `json:"knowledgeQuery"`
		MaxRepairs     int    `json:"maxRepairs"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if strings.TrimSpace(input.Instruction) == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "instruction is required")
		return
	}
	genRequest := generation.Request{Operation: generation.OperationWrite, ProjectType: string(projectItem.Type), ProjectID: projectItem.ID, Instruction: input.Instruction}
	genRequest.Evidence = append(genRequest.Evidence, a.memoryEvidence(r.Context(), projectItem.ID)...)
	reviewRequest := validation.ReviewRequest{ProjectID: projectItem.ID, Task: "校验生成内容的事实依据、一致性、完整性、术语和风格", Dimensions: []string{"groundedness", "consistency", "completeness", "terminology", "style"}}
	if strings.TrimSpace(input.KnowledgeQuery) != "" {
		hits, searchErr := a.knowledge.Search(r.Context(), projectItem.ID, input.KnowledgeQuery, 8)
		if searchErr != nil {
			writeError(w, http.StatusBadRequest, "INVALID_QUERY", searchErr.Error())
			return
		}
		for _, hit := range hits {
			genRequest.Evidence = append(genRequest.Evidence, generation.Evidence{ID: hit.Chunk.ID, Source: hit.Source.Name, Authority: string(hit.Source.Authority), Content: hit.Chunk.Content})
			reviewRequest.Evidence = append(reviewRequest.Evidence, validation.Evidence{ID: hit.Chunk.ID, Source: hit.Source.Name, Authority: string(hit.Source.Authority), Content: hit.Chunk.Content})
		}
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = "AI 质量生成正文"
	}
	flow := workflow.QualityWorkflow{Generator: a.generator, Validator: a.pipeline, MaxRepairs: input.MaxRepairs}
	item := a.tasks.Create(projectItem.ID, "QUALITY_GENERATE", func(ctx context.Context, progress func(int, string)) (any, error) {
		progress(10, "开始生成初稿")
		result, runErr := flow.Run(ctx, genRequest, reviewRequest, progress)
		if runErr != nil {
			return nil, runErr
		}
		progress(90, "保存质量门禁结果")
		doc, version, saveErr := a.docs.Create(ctx, document.CreateInput{ProjectID: projectItem.ID, Title: title, Content: result.Content})
		if saveErr != nil {
			return nil, saveErr
		}
		a.saveQuality(ctx, projectItem.ID, doc.ID, version.ID, result.Content, result.Validation)
		return map[string]any{"workflow": result, "document": doc, "version": version}, nil
	})
	writeJSON(w, http.StatusAccepted, item)
}

func (a *API) modelStatus(w http.ResponseWriter, _ *http.Request) {
	configured := a.pipeline != nil && len(a.pipeline.Validators) > 0
	validatorCount := 0
	judgeConfigured := false
	generationOperations := []string{}
	if a.pipeline != nil {
		validatorCount = len(a.pipeline.Validators)
		judgeConfigured = a.pipeline.Judge != nil
	}
	if a.generator != nil {
		for _, operation := range []generation.Operation{generation.OperationPlan, generation.OperationOutline, generation.OperationWrite, generation.OperationPolish} {
			if a.generator.Configured(operation) {
				generationOperations = append(generationOperations, string(operation))
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"configured": configured, "validatorCount": validatorCount, "judgeConfigured": judgeConfigured, "generationOperations": generationOperations})
}

func(a *API)getLocalModelConfig(w http.ResponseWriter,_ *http.Request){config,err:=a.modelConfig.Public();if err!=nil{writeError(w,http.StatusInternalServerError,"CONFIG_READ_FAILED",err.Error());return};writeJSON(w,http.StatusOK,config)}
func(a *API)updateLocalModelConfig(w http.ResponseWriter,r *http.Request){var input modelconfig.UpdateInput;if !decodeJSON(w,r,&input){return};config,err:=a.modelConfig.Save(input);if err!=nil{writeError(w,http.StatusUnprocessableEntity,"CONFIG_SAVE_FAILED",err.Error());return};writeJSON(w,http.StatusOK,map[string]any{"config":config,"restartRequired":true})}

func (a *API) listAIRuns(w http.ResponseWriter, r *http.Request) {
	if a.runs == nil {
		writeJSON(w, http.StatusOK, map[string]any{"items": []any{}, "total": 0})
		return
	}
	items, err := a.runs.List(r.Context(), r.PathValue("id"), 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	input, output := 0, 0
	latency := int64(0)
	for _, item := range items {
		input += item.InputTokens
		output += item.OutputTokens
		latency += item.LatencyMs
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items), "stats": map[string]any{"inputTokens": input, "outputTokens": output, "latencyMs": latency}})
}
func (a *API) listQualityResults(w http.ResponseWriter, r *http.Request) {
	if a.quality == nil {
		writeJSON(w, http.StatusOK, map[string]any{"items": []any{}, "total": 0})
		return
	}
	items, err := a.quality.List(r.Context(), r.PathValue("id"), 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}
func (a *API) saveQuality(ctx context.Context, projectID, documentID, versionID, text string, result validation.PipelineResult) {
	if a.quality == nil {
		return
	}
	_ = a.quality.Save(ctx, qualityhistory.Record{ID: qualityhistory.NewID(), ProjectID: projectID, DocumentID: documentID, VersionID: versionID, TextHash: fmt.Sprintf("%x", sha256.Sum256([]byte(text))), Score: result.Result.Score, Verdict: result.Result.Verdict, GateStatus: result.Gate.Status, Result: result, CreatedAt: time.Now().UTC()})
}

func (a *API) createGenerationTask(w http.ResponseWriter, r *http.Request) {
	if a.generator == nil {
		writeError(w, http.StatusServiceUnavailable, "MODEL_NOT_CONFIGURED", "generation models are not configured")
		return
	}
	projectItem, err := a.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		a.handleStoreError(w, err)
		return
	}
	var input struct {
		Operation      generation.Operation `json:"operation"`
		Instruction    string               `json:"instruction"`
		Title          string               `json:"title"`
		DocumentID     string               `json:"documentId"`
		Content        string               `json:"content"`
		KnowledgeQuery string               `json:"knowledgeQuery"`
		Save           *bool                `json:"save"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if !input.Operation.Valid() || strings.TrimSpace(input.Instruction) == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "valid operation and instruction are required")
		return
	}
	if !a.generator.Configured(input.Operation) {
		writeError(w, http.StatusServiceUnavailable, "MODEL_NOT_CONFIGURED", "model is not configured for "+string(input.Operation))
		return
	}
	request := generation.Request{Operation: input.Operation, ProjectType: string(projectItem.Type), ProjectID: projectItem.ID, Instruction: input.Instruction, Content: input.Content}
	request.Evidence = append(request.Evidence, a.memoryEvidence(r.Context(), projectItem.ID)...)
	if strings.TrimSpace(input.KnowledgeQuery) != "" {
		hits, searchErr := a.knowledge.Search(r.Context(), projectItem.ID, input.KnowledgeQuery, 8)
		if searchErr != nil {
			writeError(w, http.StatusBadRequest, "INVALID_QUERY", searchErr.Error())
			return
		}
		for _, hit := range hits {
			request.Evidence = append(request.Evidence, generation.Evidence{ID: hit.Chunk.ID, Source: hit.Source.Name, Authority: string(hit.Source.Authority), Content: hit.Chunk.Content})
		}
	}
	expectedVersionID := ""
	if input.DocumentID != "" {
		doc, getErr := a.docs.Get(r.Context(), input.DocumentID)
		if getErr != nil || doc.ProjectID != projectItem.ID {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "document not found in project")
			return
		}
		expectedVersionID = doc.CurrentVersionID
		if request.Content == "" {
			versions, versionErr := a.docs.Versions(r.Context(), doc.ID)
			if versionErr != nil || len(versions) == 0 {
				writeError(w, http.StatusUnprocessableEntity, "DOCUMENT_EMPTY", "document has no content version")
				return
			}
			request.Content = versions[0].Content
		}
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = operationTitle(input.Operation)
	}
	item := a.tasks.Create(projectItem.ID, string(input.Operation), func(ctx context.Context, progress func(int, string)) (any, error) {
		progress(10, "知识上下文构建完成")
		progress(20, string(input.Operation)+" Agent 正在生成")
		generated, generateErr := a.generator.Generate(ctx, request)
		if generateErr != nil {
			return nil, generateErr
		}
		progress(85, "模型输出完成，正在创建文档版本")
		if input.Save != nil && !*input.Save {
			return map[string]any{"generation": generated}, nil
		}
		if input.DocumentID == "" {
			doc, version, createErr := a.docs.Create(ctx, document.CreateInput{ProjectID: projectItem.ID, Title: title, Content: generated.Content})
			if createErr != nil {
				return nil, createErr
			}
			return map[string]any{"generation": generated, "document": doc, "version": version}, nil
		}
		version, versionErr := a.docs.CreateVersion(ctx, input.DocumentID, document.CreateVersionInput{Content: generated.Content, Reason: "AI_" + string(input.Operation), AuthorType: "AI", ExpectedVersionID: expectedVersionID})
		if versionErr != nil {
			return nil, versionErr
		}
		return map[string]any{"generation": generated, "documentId": input.DocumentID, "version": version}, nil
	})
	writeJSON(w, http.StatusAccepted, item)
}

func operationTitle(operation generation.Operation) string {
	switch operation {
	case generation.OperationPlan:
		return "AI 内容策划"
	case generation.OperationOutline:
		return "AI 内容目录"
	case generation.OperationPolish:
		return "AI 润色稿"
	default:
		return "AI 生成正文"
	}
}

func (a *API) validateText(w http.ResponseWriter, r *http.Request) {
	if a.pipeline == nil {
		writeError(w, http.StatusServiceUnavailable, "MODEL_NOT_CONFIGURED", "validation models are not configured")
		return
	}
	request, ok := a.decodeReviewRequest(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	result, err := a.pipeline.Validate(r.Context(), request)
	if err != nil {
		a.logger.Error("validation pipeline failed", "projectId", r.PathValue("id"), "error", err)
		writeError(w, http.StatusBadGateway, "VALIDATION_FAILED", err.Error())
		return
	}
	a.saveQuality(r.Context(), r.PathValue("id"), "", "", request.Text, result)
	writeJSON(w, http.StatusOK, result)
}

func (a *API) createValidationTask(w http.ResponseWriter, r *http.Request) {
	if a.pipeline == nil {
		writeError(w, http.StatusServiceUnavailable, "MODEL_NOT_CONFIGURED", "validation models are not configured")
		return
	}
	projectID := r.PathValue("id")
	request, ok := a.decodeReviewRequest(w, r, projectID)
	if !ok {
		return
	}
	item := a.tasks.Create(projectID, "TEXT_VALIDATE", func(ctx context.Context, progress func(int, string)) (any, error) {
		progress(10, "知识证据准备完成")
		progress(20, "Validator 开始独立校验")
		result, err := a.pipeline.Validate(ctx, request)
		if err != nil {
			return nil, err
		}
		a.saveQuality(ctx, projectID, "", "", request.Text, result)
		progress(90, "质量门禁计算完成")
		return result, nil
	})
	writeJSON(w, http.StatusAccepted, item)
}

func (a *API) decodeReviewRequest(w http.ResponseWriter, r *http.Request, projectID string) (validation.ReviewRequest, bool) {
	if _, err := a.store.Get(r.Context(), projectID); err != nil {
		a.handleStoreError(w, err)
		return validation.ReviewRequest{}, false
	}
	var input struct {
		Text           string   `json:"text"`
		Task           string   `json:"task"`
		KnowledgeQuery string   `json:"knowledgeQuery"`
		Dimensions     []string `json:"dimensions"`
	}
	if !decodeJSON(w, r, &input) {
		return validation.ReviewRequest{}, false
	}
	request := validation.ReviewRequest{Text: input.Text, Task: input.Task, Dimensions: input.Dimensions, ProjectID: projectID}
	if strings.TrimSpace(input.KnowledgeQuery) != "" {
		hits, err := a.knowledge.Search(r.Context(), projectID, input.KnowledgeQuery, 8)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_QUERY", err.Error())
			return validation.ReviewRequest{}, false
		}
		for _, hit := range hits {
			request.Evidence = append(request.Evidence, validation.Evidence{ID: hit.Chunk.ID, Source: hit.Source.Name, Authority: string(hit.Source.Authority), Content: hit.Chunk.Content})
		}
	}
	return request, true
}

func (a *API) listTasks(w http.ResponseWriter, r *http.Request) {
	items := a.tasks.List(r.PathValue("id"))
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func (a *API) listAllTasks(w http.ResponseWriter, _ *http.Request) {
	items := a.tasks.List("")
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func (a *API) getTask(w http.ResponseWriter, r *http.Request) {
	item, err := a.tasks.Get(r.PathValue("id"))
	if err != nil {
		a.handleTaskError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) cancelTask(w http.ResponseWriter, r *http.Request) {
	if err := a.tasks.Cancel(r.PathValue("id")); err != nil {
		a.handleTaskError(w, err)
		return
	}
	item, _ := a.tasks.Get(r.PathValue("id"))
	writeJSON(w, http.StatusOK, item)
}

func (a *API) retryTask(w http.ResponseWriter, r *http.Request) {
	item, err := a.tasks.Retry(r.PathValue("id"))
	if err != nil {
		a.handleTaskError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, item)
}

func (a *API) taskEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "SSE_UNSUPPORTED", "streaming is unsupported")
		return
	}
	lastID, _ := strconv.ParseUint(r.Header.Get("Last-Event-ID"), 10, 64)
	if queryID, err := strconv.ParseUint(r.URL.Query().Get("after"), 10, 64); err == nil && queryID > lastID {
		lastID = queryID
	}
	stream, unsubscribe, err := a.tasks.Subscribe(r.PathValue("id"))
	if err != nil {
		a.handleTaskError(w, err)
		return
	}
	defer unsubscribe()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	send := func(event task.Event) {
		raw, _ := json.Marshal(event)
		_, _ = fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Type, raw)
		flusher.Flush()
		lastID = event.ID
	}
	replay, _ := a.tasks.EventsSince(r.PathValue("id"), lastID)
	for _, event := range replay {
		send(event)
	}
	if item, _ := a.tasks.Get(r.PathValue("id")); item.Status == task.StatusSuccess || item.Status == task.StatusFailed || item.Status == task.StatusCancelled {
		return
	}
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-stream:
			if event.ID > lastID {
				send(event)
			}
			if event.Type == "task.completed" || event.Type == "task.failed" || event.Type == "task.cancelled" {
				return
			}
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (a *API) handleTaskError(w http.ResponseWriter, err error) {
	if errors.Is(err, task.ErrNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	if errors.Is(err, task.ErrNotRetryable) {
		writeError(w, http.StatusConflict, "TASK_NOT_RETRYABLE", "任务仍在运行、已成功，或服务重启后缺少原执行上下文")
		return
	}
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
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

func (a *API) dashboardStats(w http.ResponseWriter, r *http.Request) {
	projects, err := a.store.List(r.Context())
	if err != nil { writeError(w,http.StatusInternalServerError,"INTERNAL_ERROR",err.Error());return }
	documentCount,sourceCount,pendingIssues,qualityTotal,qualityScore:=0,0,0,0,0
	for _,item:=range projects{
		if docs,docErr:=a.docs.List(r.Context(),item.ID);docErr==nil{documentCount+=len(docs)}
		if sources,sourceErr:=a.knowledge.ListSources(r.Context(),item.ID);sourceErr==nil{sourceCount+=len(sources)}
		if a.quality!=nil{if results,resultErr:=a.quality.List(r.Context(),item.ID,200);resultErr==nil{for _,result:=range results{qualityTotal++;qualityScore+=result.Score;for _,issue:=range result.Result.Result.Issues{if issue.Severity=="CRITICAL"||issue.Severity=="MAJOR"{pendingIssues++}}}}}
	}
	runningTasks:=0
	for _,item:=range a.tasks.List(""){if item.Status==task.StatusPending||item.Status==task.StatusRunning{runningTasks++}}
	average:=0;if qualityTotal>0{average=qualityScore/qualityTotal}
	writeJSON(w,http.StatusOK,map[string]any{"projects":len(projects),"documents":documentCount,"knowledgeSources":sourceCount,"pendingIssues":pendingIssues,"runningTasks":runningTasks,"averageQualityScore":average})
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

func (a *API) diffVersions(w http.ResponseWriter, r *http.Request) {
	versions, err := a.docs.Versions(r.Context(), r.PathValue("id"))
	if err != nil {
		a.handleDocumentError(w, err)
		return
	}
	byID := make(map[string]document.Version, len(versions))
	for _, version := range versions {
		byID[version.ID] = version
	}
	from, fromOK := byID[r.URL.Query().Get("from")]
	to, toOK := byID[r.URL.Query().Get("to")]
	if !fromOK || !toOK {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "document version not found")
		return
	}
	writeJSON(w, http.StatusOK, document.Diff(from, to))
}

func (a *API) exportProjectMarkdown(w http.ResponseWriter, r *http.Request) {
	item, err := a.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		a.handleStoreError(w, err)
		return
	}
	documents, err := a.docs.List(r.Context(), item.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	tree, _ := a.store.Tree(r.Context(), item.ID)
	documentByID := make(map[string]document.Document, len(documents))
	for _, doc := range documents {
		documentByID[doc.ID] = doc
	}
	ordered := make([]document.Document, 0, len(documents))
	seen := map[string]bool{}
	for _, node := range tree {
		if node.DocumentID != nil && !seen[*node.DocumentID] {
			if doc, ok := documentByID[*node.DocumentID]; ok {
				ordered = append(ordered, doc)
				seen[doc.ID] = true
			}
		}
	}
	for _, doc := range documents {
		if !seen[doc.ID] {
			ordered = append(ordered, doc)
		}
	}
	var output strings.Builder
	fmt.Fprintf(&output, "# %s\n\n%s\n", item.Name, item.Description)
	for _, doc := range ordered {
		versions, versionErr := a.docs.Versions(r.Context(), doc.ID)
		if versionErr != nil || len(versions) == 0 {
			continue
		}
		fmt.Fprintf(&output, "\n---\n\n# %s\n\n%s\n", doc.Title, versions[0].Content)
	}
	filename := regexp.MustCompile(`[^a-zA-Z0-9_-]+`).ReplaceAllString(item.Name, "_")
	if filename == "" {
		filename = "content-export"
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`.md"`)
	_, _ = io.WriteString(w, output.String())
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
	case errors.Is(err, document.ErrConflict):
		writeError(w, http.StatusConflict, "VERSION_CONFLICT", err.Error())
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
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
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
