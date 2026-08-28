package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"novelstudio/internal/httpapi"
	"novelstudio/internal/project"
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

type projectDocument struct {
	ID string `json:"id"`
}
