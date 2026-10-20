package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func registerUser(t *testing.T, h http.Handler, name, email string) *http.Cookie {
	t.Helper()
	response := httptest.NewRecorder()
	body := fmt.Sprintf(`{"name":%q,"email":%q,"password":"long-password"}`, name, email)
	h.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body)))
	if response.Code != http.StatusCreated {
		t.Fatalf("register %s: %d %s", email, response.Code, response.Body.String())
	}
	return response.Result().Cookies()[0]
}

func authenticatedRequest(method, target string, body io.Reader, cookie *http.Cookie) *http.Request {
	request := httptest.NewRequest(method, target, body)
	request.AddCookie(cookie)
	return request
}

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

func TestProjectsAreIsolatedByUser(t *testing.T) {
	h := authHandler()
	first := registerUser(t, h, "Writer One", "one@example.com")
	second := registerUser(t, h, "Writer Two", "two@example.com")

	created := httptest.NewRecorder()
	h.ServeHTTP(created, authenticatedRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(`{"name":"Private","type":"NOVEL"}`), first))
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	var item project.Project
	if err := json.NewDecoder(created.Body).Decode(&item); err != nil {
		t.Fatal(err)
	}

	list := httptest.NewRecorder()
	h.ServeHTTP(list, authenticatedRequest(http.MethodGet, "/api/v1/projects", nil, second))
	if list.Code != http.StatusOK || bytes.Contains(list.Body.Bytes(), []byte(item.ID)) {
		t.Fatalf("second user can list first user's project: %d %s", list.Code, list.Body.String())
	}
	get := httptest.NewRecorder()
	h.ServeHTTP(get, authenticatedRequest(http.MethodGet, "/api/v1/projects/"+item.ID, nil, second))
	if get.Code != http.StatusNotFound {
		t.Fatalf("second user can read first user's project: %d %s", get.Code, get.Body.String())
	}
}
