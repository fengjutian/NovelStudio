package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"novelstudio/internal/document"
	"novelstudio/internal/generation"
	"novelstudio/internal/httpapi"
	"novelstudio/internal/knowledge"
	"novelstudio/internal/llm"
	"novelstudio/internal/project"
	"novelstudio/internal/task"
	"novelstudio/internal/validation"
)

func TestProjectLifecycle(t *testing.T) {
	handler := httpapi.New(project.NewMemoryStore(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	body := []byte(`{"name":"API 指南","type":"TECHNICAL_DOCUMENT","description":"测试项目"}`)
	create := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewReader(body))
	create.Header.Set("Content-Type", "application/json")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", created.Code, created.Body.String())
	}
	var item project.Project
	if err := json.NewDecoder(created.Body).Decode(&item); err != nil {
		t.Fatal(err)
	}
	if item.Name != "API 指南" || item.Type != project.TypeTechnicalDocument {
		t.Fatalf("unexpected project: %#v", item)
	}

	tree := httptest.NewRecorder()
	handler.ServeHTTP(tree, httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+item.ID+"/tree", nil))
	if tree.Code != http.StatusOK {
		t.Fatalf("tree status = %d", tree.Code)
	}
}

func TestRejectsUnsupportedProjectType(t *testing.T) {
	handler := httpapi.New(project.NewMemoryStore(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(`{"name":"x","type":"UNKNOWN"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnprocessableEntity)
	}
}

func TestCanonCommitAndState(t *testing.T) {
	projects := project.NewMemoryStore()
	handler := httpapi.New(projects, slog.New(slog.NewTextHandler(io.Discard, nil)))
	items, _ := projects.List(context.Background())
	projectID := items[0].ID
	body := `{"baseCanonVersion":0,"chapterVersionId":"ver_1","approvedBy":"author","entities":[{"id":"char_1","type":"CHARACTER","name":"林川"}],"transitions":[{"id":"state_1","entityId":"char_1","field":"health","before":null,"after":"injured","storyTime":"day-1","source":{"documentVersionId":"ver_1","startOffset":1,"endOffset":4}}]}`
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/canon/commits", bytes.NewBufferString(body)))
	if response.Code != http.StatusCreated {
		t.Fatalf("commit status=%d body=%s", response.Code, response.Body.String())
	}

	state := httptest.NewRecorder()
	handler.ServeHTTP(state, httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID+"/canon", nil))
	if state.Code != http.StatusOK || !strings.Contains(state.Body.String(), `"health":"injured"`) {
		t.Fatalf("state status=%d body=%s", state.Code, state.Body.String())
	}

	conflict := httptest.NewRecorder()
	handler.ServeHTTP(conflict, httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/canon/commits", bytes.NewBufferString(body)))
	if conflict.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d body=%s", conflict.Code, conflict.Body.String())
	}
}

func TestChapterImpactAnalysisMarksTransitiveDependenciesStale(t *testing.T) {
	projects := project.NewMemoryStore()
	handler := httpapi.New(projects, slog.New(slog.NewTextHandler(io.Discard, nil)))
	items, _ := projects.List(context.Background())
	projectID := items[0].ID
	register := func(chapter, body string) {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID+"/chapter-versions/"+chapter+"/dependencies", bytes.NewBufferString(body)))
		if response.Code != http.StatusNoContent {
			t.Fatalf("register %s status=%d body=%s", chapter, response.Code, response.Body.String())
		}
	}
	register("ch7", `{"canonVersion":2,"factIds":["fact-injury"]}`)
	register("ch9", `{"canonVersion":3,"dependsOnChapterVersionIds":["ch7"]}`)
	register("ch12", `{"canonVersion":4,"dependsOnChapterVersionIds":["ch9"]}`)
	impact := httptest.NewRecorder()
	handler.ServeHTTP(impact, httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/impact-analysis", bytes.NewBufferString(`{"triggerCanonVersion":5,"changedFactIds":["fact-injury"],"markStale":true}`)))
	if impact.Code != http.StatusOK || !strings.Contains(impact.Body.String(), `"ch12"`) {
		t.Fatalf("impact status=%d body=%s", impact.Code, impact.Body.String())
	}
	stale := httptest.NewRecorder()
	handler.ServeHTTP(stale, httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID+"/stale-chapters", nil))
	if stale.Code != http.StatusOK || !strings.Contains(stale.Body.String(), `"total":3`) {
		t.Fatalf("stale status=%d body=%s", stale.Code, stale.Body.String())
	}
	resolved := httptest.NewRecorder()
	handler.ServeHTTP(resolved, httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/stale-chapters/ch12/resolve", bytes.NewBufferString(`{"status":"VALID","resolvedBy":"author","note":"review passed"}`)))
	if resolved.Code != http.StatusOK || !strings.Contains(resolved.Body.String(), `"status":"VALID"`) {
		t.Fatalf("resolve status=%d body=%s", resolved.Code, resolved.Body.String())
	}
}

func TestUpdateProject(t *testing.T) {
	handler := httpapi.New(project.NewMemoryStore(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	create := httptest.NewRecorder()
	handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(`{"name":"Old name","type":"NOVEL","description":"Old description"}`)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", create.Code, create.Body.String())
	}
	var item project.Project
	if err := json.NewDecoder(create.Body).Decode(&item); err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+item.ID, bytes.NewBufferString(`{"name":"New name","description":"New description"}`)))
	if response.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", response.Code, response.Body.String())
	}
	var updated project.Project
	if err := json.NewDecoder(response.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "New name" || updated.Description != "New description" || updated.Type != project.TypeNovel {
		t.Fatalf("updated = %#v", updated)
	}
}

func TestOutlineImportCreatesHierarchy(t *testing.T) {
	projects := project.NewMemoryStore()
	handler := httpapi.New(projects, slog.New(slog.NewTextHandler(io.Discard, nil)))
	items, _ := projects.List(context.Background())
	projectID := items[0].ID
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/outline-import", bytes.NewBufferString(`{"content":"# 第一卷\n## 第一章\n### 第一节","preview":false}`)))
	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	tree, err := projects.Tree(context.Background(), projectID)
	if err != nil || len(tree) < 5 {
		t.Fatalf("tree=%#v err=%v", tree, err)
	}
	created := tree[len(tree)-3:]
	if created[1].ParentID == nil || *created[1].ParentID != created[0].ID || created[2].ParentID == nil || *created[2].ParentID != created[1].ID {
		t.Fatalf("unexpected hierarchy: %#v", created)
	}
	repeated := httptest.NewRecorder()
	handler.ServeHTTP(repeated, httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/outline-import", bytes.NewBufferString(`{"content":"# 第一卷\n## 第一章\n### 第一节","preview":false}`)))
	if repeated.Code != http.StatusCreated {
		t.Fatalf("repeat status=%d body=%s", repeated.Code, repeated.Body.String())
	}
	afterRepeat, err := projects.Tree(context.Background(), projectID)
	if err != nil || len(afterRepeat) != len(tree) {
		t.Fatalf("repeated import created duplicates: before=%d after=%d err=%v", len(tree), len(afterRepeat), err)
	}
}

func TestDocumentAndKnowledgeLifecycle(t *testing.T) {
	handler := httpapi.New(project.NewMemoryStore(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil))
	var projects struct {
		Items []project.Project `json:"items"`
	}
	if err := json.NewDecoder(list.Body).Decode(&projects); err != nil {
		t.Fatal(err)
	}
	projectID := projects.Items[0].ID

	createDoc := httptest.NewRecorder()
	handler.ServeHTTP(createDoc, httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/documents", bytes.NewBufferString(`{"title":"第一章","content":"初稿"}`)))
	if createDoc.Code != http.StatusCreated {
		t.Fatalf("create document status=%d body=%s", createDoc.Code, createDoc.Body.String())
	}
	var created struct {
		Document projectDocument `json:"document"`
	}
	if err := json.NewDecoder(createDoc.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	version := httptest.NewRecorder()
	handler.ServeHTTP(version, httptest.NewRequest(http.MethodPost, "/api/v1/documents/"+created.Document.ID+"/versions", bytes.NewBufferString(`{"content":"修改稿","reason":"EDIT"}`)))
	if version.Code != http.StatusCreated {
		t.Fatalf("create version status=%d body=%s", version.Code, version.Body.String())
	}

	createSource := httptest.NewRecorder()
	handler.ServeHTTP(createSource, httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/knowledge/sources", bytes.NewBufferString(`{"name":"设定集","authority":"OFFICIAL","content":"沈砚是县衙书吏。"}`)))
	if createSource.Code != http.StatusCreated {
		t.Fatalf("create source status=%d body=%s", createSource.Code, createSource.Body.String())
	}
	search := httptest.NewRecorder()
	handler.ServeHTTP(search, httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID+"/knowledge/search?q=沈砚", nil))
	if search.Code != http.StatusOK || !bytes.Contains(search.Body.Bytes(), []byte("县衙书吏")) {
		t.Fatalf("search status=%d body=%s", search.Code, search.Body.String())
	}
}

func TestDeleteProjectRemovesKnowledgeFiles(t *testing.T) {
	projects := project.NewMemoryStore()
	knowledgeStore := knowledge.NewMemoryStore()
	handler := httpapi.NewWithStores(projects, document.NewMemoryStore(), knowledgeStore, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	items, _ := projects.List(context.Background())
	projectID := items[0].ID
	root := t.TempDir()
	t.Setenv("KNOWLEDGE_UPLOAD_DIR", root)
	relative := filepath.Join(projectID, "sample.txt")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, relative)), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, relative), []byte("sample"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := knowledgeStore.CreateFileAsset(context.Background(), knowledge.CreateFileAssetInput{ProjectID: projectID, Name: "sample.txt", Extension: ".txt", StoragePath: relative}); err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID, nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", response.Code, response.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, relative)); !os.IsNotExist(err) {
		t.Fatalf("uploaded file still exists: %v", err)
	}
	files, err := knowledgeStore.ListFileAssets(context.Background(), projectID)
	if err != nil || len(files) != 0 {
		t.Fatalf("knowledge files remain: %#v, %v", files, err)
	}
}

type projectDocument struct {
	ID string `json:"id"`
}

type fixedReviewer struct{}

func (fixedReviewer) Review(context.Context, validation.ReviewRequest) (validation.Result, validation.Run, error) {
	return validation.Result{Score: 91, Verdict: "PASS", Dimensions: map[string]int{"groundedness": 90}, Issues: []validation.Issue{}}, validation.Run{Status: "SUCCESS", Model: "test-model"}, nil
}

func TestAsyncValidationTask(t *testing.T) {
	projects := project.NewMemoryStore()
	pipeline := &validation.Pipeline{Validators: []validation.NamedReviewer{{Name: "A", Reviewer: fixedReviewer{}}}}
	handler := httpapi.NewWithStores(projects, document.NewMemoryStore(), knowledge.NewMemoryStore(), pipeline, slog.New(slog.NewTextHandler(io.Discard, nil)))
	list, _ := projects.List(context.Background())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+list[0].ID+"/validation-tasks", bytes.NewBufferString(`{"text":"verified text","task":"check"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("create task status=%d body=%s", response.Code, response.Body.String())
	}
	var created task.Task
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		get := httptest.NewRecorder()
		handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+created.ID, nil))
		var current task.Task
		if err := json.NewDecoder(get.Body).Decode(&current); err != nil {
			t.Fatal(err)
		}
		if current.Status == task.StatusSuccess {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("validation task did not complete")
}

type generationProvider struct{}

func (generationProvider) Generate(context.Context, llm.GenerateRequest) (llm.GenerateResponse, error) {
	return llm.GenerateResponse{Content: "# Generated document", RequestID: "generation-request"}, nil
}
func (generationProvider) HealthCheck(context.Context) error { return nil }

type resumableGenerationProvider struct {
	mu       sync.Mutex
	calls    int
	requests []llm.GenerateRequest
}

func (p *resumableGenerationProvider) Generate(_ context.Context, request llm.GenerateRequest) (llm.GenerateResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	p.requests = append(p.requests, request)
	if p.calls == 2 {
		return llm.GenerateResponse{}, errors.New("temporary provider failure")
	}
	return llm.GenerateResponse{Content: "generated body " + string(rune('0'+p.calls))}, nil
}
func (*resumableGenerationProvider) HealthCheck(context.Context) error { return nil }

func TestGenerationTaskCreatesDocument(t *testing.T) {
	projects := project.NewMemoryStore()
	documents := document.NewMemoryStore()
	knowledgeStore := knowledge.NewMemoryStore()
	generator := &generation.Service{ProviderName: "test", Provider: generationProvider{}, Models: map[generation.Operation]string{generation.OperationWrite: "writer"}}
	handler := httpapi.NewWithServices(projects, documents, knowledgeStore, nil, generator, slog.New(slog.NewTextHandler(io.Discard, nil)))
	list, _ := projects.List(context.Background())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+list[0].ID+"/generation-tasks", bytes.NewBufferString(`{"operation":"WRITE","instruction":"write a guide","title":"Guide"}`)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("create generation task status=%d body=%s", response.Code, response.Body.String())
	}
	var created task.Task
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		current := httptest.NewRecorder()
		handler.ServeHTTP(current, httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+created.ID, nil))
		var item task.Task
		_ = json.NewDecoder(current.Body).Decode(&item)
		if item.Status == task.StatusSuccess {
			items, err := documents.List(context.Background(), list[0].ID)
			if err != nil || len(items) != 1 || items[0].Title != "Guide" {
				t.Fatalf("generated documents=%#v err=%v", items, err)
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("generation task did not complete")
}

func TestBatchGenerationRetryKeepsCompletedDocuments(t *testing.T) {
	projects := project.NewMemoryStore()
	documents := document.NewMemoryStore()
	provider := &resumableGenerationProvider{}
	generator := &generation.Service{ProviderName: "test", Provider: provider, Models: map[generation.Operation]string{generation.OperationWrite: "writer"}}
	handler := httpapi.NewWithServices(projects, documents, knowledge.NewMemoryStore(), nil, generator, slog.New(slog.NewTextHandler(io.Discard, nil)))
	items, _ := projects.List(context.Background())
	projectID := items[0].ID
	nodes := make([]project.ContentNode, 0, 2)
	for index, title := range []string{"chapter one", "chapter two"} {
		node, err := projects.CreateNode(context.Background(), projectID, project.CreateNodeInput{NodeType: "CHAPTER", Title: title, Position: index + 1})
		if err != nil {
			t.Fatal(err)
		}
		doc, _, err := documents.Create(context.Background(), document.CreateInput{ProjectID: projectID, Title: title, Content: ""})
		if err != nil {
			t.Fatal(err)
		}
		documentID := doc.ID
		node, err = projects.UpdateNode(context.Background(), node.ID, project.UpdateNodeInput{Title: title, Position: index + 1, DocumentID: &documentID})
		if err != nil {
			t.Fatal(err)
		}
		nodes = append(nodes, node)
	}
	body, _ := json.Marshal(map[string]any{"nodeIds": []string{nodes[0].ID, nodes[1].ID}, "instruction": "write", "windowSize": 1})
	createdResponse := httptest.NewRecorder()
	handler.ServeHTTP(createdResponse, httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/batch-generation-tasks", bytes.NewReader(body)))
	var created task.Task
	if err := json.NewDecoder(createdResponse.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	waitForStatus := func(id string, wanted task.Status) task.Task {
		deadline := time.Now().Add(2 * time.Second)
		var last task.Task
		for time.Now().Before(deadline) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+id, nil))
			var current task.Task
			_ = json.NewDecoder(response.Body).Decode(&current)
			last = current
			if current.Status == wanted {
				return current
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatalf("task %s did not reach %s: status=%s error=%s", id, wanted, last.Status, last.Error)
		return task.Task{}
	}
	waitForStatus(created.ID, task.StatusFailed)
	versionsOne, _ := documents.Versions(context.Background(), *nodes[0].DocumentID)
	versionsTwo, _ := documents.Versions(context.Background(), *nodes[1].DocumentID)
	if len(versionsOne)+len(versionsTwo) != 3 || len(versionsOne) == len(versionsTwo) {
		t.Fatalf("versions after failure = %d, %d; exactly one document should be saved", len(versionsOne), len(versionsTwo))
	}
	retryResponse := httptest.NewRecorder()
	handler.ServeHTTP(retryResponse, httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+created.ID+"/retry", nil))
	var retried task.Task
	if err := json.NewDecoder(retryResponse.Body).Decode(&retried); err != nil {
		t.Fatal(err)
	}
	waitForStatus(retried.ID, task.StatusSuccess)
	versionsOne, _ = documents.Versions(context.Background(), *nodes[0].DocumentID)
	versionsTwo, _ = documents.Versions(context.Background(), *nodes[1].DocumentID)
	if len(versionsOne) != 2 || len(versionsTwo) != 2 {
		t.Fatalf("versions after retry = %d, %d", len(versionsOne), len(versionsTwo))
	}
	if provider.calls != 3 {
		t.Fatalf("provider calls = %d, want 3", provider.calls)
	}
	if !strings.Contains(provider.requests[2].Messages[1].Content, "generated body 1") || !strings.Contains(provider.requests[2].Messages[1].Content, "<previous-chapter>") {
		t.Fatal("retried second chapter did not receive the completed previous chapter as continuity context")
	}
	forceBody, _ := json.Marshal(map[string]any{"nodeIds": []string{nodes[0].ID, nodes[1].ID}, "instruction": "rewrite all", "windowSize": 1, "force": true})
	forceResponse := httptest.NewRecorder()
	handler.ServeHTTP(forceResponse, httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/batch-generation-tasks", bytes.NewReader(forceBody)))
	var forced task.Task
	if err := json.NewDecoder(forceResponse.Body).Decode(&forced); err != nil {
		t.Fatal(err)
	}
	waitForStatus(forced.ID, task.StatusSuccess)
	versionsOne, _ = documents.Versions(context.Background(), *nodes[0].DocumentID)
	versionsTwo, _ = documents.Versions(context.Background(), *nodes[1].DocumentID)
	if len(versionsOne) != 3 || len(versionsTwo) != 3 || provider.calls != 5 {
		t.Fatalf("forced regeneration versions = %d, %d; provider calls = %d", len(versionsOne), len(versionsTwo), provider.calls)
	}
}

func TestCommentaryGenerationAllowsMissingKnowledgeSource(t *testing.T) {
	projects := project.NewMemoryStore()
	documents := document.NewMemoryStore()
	generator := &generation.Service{ProviderName: "test", Provider: generationProvider{}, Models: map[generation.Operation]string{generation.OperationWrite: "writer"}}
	handler := httpapi.NewWithServices(projects, documents, knowledge.NewMemoryStore(), nil, generator, slog.New(slog.NewTextHandler(io.Discard, nil)))
	item, err := projects.Create(context.Background(), project.CreateInput{Name: "Known show", Type: project.TypeTVCommentary})
	if err != nil {
		t.Fatal(err)
	}
	singleResponse := httptest.NewRecorder()
	handler.ServeHTTP(singleResponse, httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+item.ID+"/generation-tasks", bytes.NewBufferString(`{"operation":"WRITE","instruction":"write","title":"Episode one"}`)))
	if singleResponse.Code != http.StatusAccepted {
		t.Fatalf("single generation status=%d body=%s", singleResponse.Code, singleResponse.Body.String())
	}
	node, err := projects.CreateNode(context.Background(), item.ID, project.CreateNodeInput{NodeType: "EPISODE", Title: "Episode one", Position: 1})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"nodeIds": []string{node.ID}, "instruction": "write", "windowSize": 1})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+item.ID+"/batch-generation-tasks", bytes.NewReader(body)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
