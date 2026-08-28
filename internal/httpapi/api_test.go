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
