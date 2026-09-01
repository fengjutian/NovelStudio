package httpapi_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"novelstudio/internal/auth"
	"novelstudio/internal/document"
	"novelstudio/internal/httpapi"
	"novelstudio/internal/knowledge"
	"novelstudio/internal/project"
	"novelstudio/internal/task"
)

func authHandler() http.Handler {
	return httpapi.NewWithRuntimeAuth(project.NewMemoryStore(), document.NewMemoryStore(), knowledge.NewMemoryStore(), nil, nil, task.NewManager(), nil, nil, auth.NewMemoryStore(), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestRegisterLoginAndProtectedRoutes(t *testing.T) {
	h := authHandler()
	unauthorized := httptest.NewRecorder()
	h.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unprotected route: %d", unauthorized.Code)
	}
	register := httptest.NewRecorder()
	h.ServeHTTP(register, httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{"name":"Writer","email":"writer@example.com","password":"long-password"}`)))
	if register.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", register.Code, register.Body.String())
	}
	cookies := register.Result().Cookies()
	if len(cookies) == 0 || !cookies[0].HttpOnly {
		t.Fatal("missing secure session cookie")
	}
	meRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meRequest.AddCookie(cookies[0])
	me := httptest.NewRecorder()
	h.ServeHTTP(me, meRequest)
	if me.Code != http.StatusOK {
		t.Fatalf("me: %d", me.Code)
	}
	duplicate := httptest.NewRecorder()
	h.ServeHTTP(duplicate, httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{"name":"Other","email":"WRITER@example.com","password":"long-password"}`)))
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate: %d", duplicate.Code)
	}
	login := httptest.NewRecorder()
	h.ServeHTTP(login, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"writer@example.com","password":"long-password"}`)))
	if login.Code != http.StatusOK {
		t.Fatalf("login: %d %s", login.Code, login.Body.String())
	}
}
