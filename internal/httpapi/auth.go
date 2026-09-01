package httpapi

import (
	"errors"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"novelstudio/internal/auth"
)

const sessionCookie = "novelstudio_session"
const sessionLifetime = 30 * 24 * time.Hour

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Email = auth.NormalizeEmail(input.Email)
	if utf8.RuneCountInString(input.Name) < 2 || utf8.RuneCountInString(input.Name) > 80 {
		writeError(w, 422, "INVALID_NAME", "name must be 2 to 80 characters")
		return
	}
	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email || len(input.Email) > 254 {
		writeError(w, 422, "INVALID_EMAIL", "please enter a valid email address")
		return
	}
	if len(input.Password) < 8 || len(input.Password) > 128 {
		writeError(w, 422, "INVALID_PASSWORD", "password must be 8 to 128 characters")
		return
	}
	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		writeError(w, 500, "INTERNAL_ERROR", "could not secure password")
		return
	}
	user := auth.User{ID: auth.NewID("usr_"), Name: input.Name, Email: input.Email, CreatedAt: time.Now().UTC()}
	if err = a.auth.CreateUser(r.Context(), user, hash); errors.Is(err, auth.ErrEmailTaken) {
		writeError(w, 409, "EMAIL_TAKEN", "this email is already registered")
		return
	} else if err != nil {
		writeError(w, 500, "INTERNAL_ERROR", err.Error())
		return
	}
	if !a.startSession(w, r, user.ID) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": user})
}
func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	user, hash, err := a.auth.UserByEmail(r.Context(), input.Email)
	if err != nil || !auth.CheckPassword(hash, input.Password) {
		writeError(w, 401, "INVALID_CREDENTIALS", "email or password is incorrect")
		return
	}
	if !a.startSession(w, r, user.ID) {
		return
	}
	writeJSON(w, 200, map[string]any{"user": user})
}
func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		_ = a.auth.DeleteSession(r.Context(), auth.TokenHash(cookie.Value))
	}
	a.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
func (a *API) me(w http.ResponseWriter, r *http.Request) {
	user, ok := a.currentUser(r)
	if !ok {
		writeError(w, 401, "UNAUTHENTICATED", "please sign in")
		return
	}
	writeJSON(w, 200, map[string]any{"user": user})
}
func (a *API) startSession(w http.ResponseWriter, r *http.Request, userID string) bool {
	token, err := auth.NewToken()
	if err == nil {
		err = a.auth.CreateSession(r.Context(), auth.TokenHash(token), userID, time.Now().UTC().Add(sessionLifetime))
	}
	if err != nil {
		writeError(w, 500, "INTERNAL_ERROR", "could not create session")
		return false
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: os.Getenv("COOKIE_SECURE") == "true", SameSite: http.SameSiteLaxMode, MaxAge: int(sessionLifetime.Seconds())})
	return true
}
func (a *API) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Path: "/", HttpOnly: true, Secure: os.Getenv("COOKIE_SECURE") == "true", SameSite: http.SameSiteLaxMode, MaxAge: -1})
}
func (a *API) currentUser(r *http.Request) (auth.User, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return auth.User{}, false
	}
	user, err := a.auth.UserBySession(r.Context(), auth.TokenHash(cookie.Value), time.Now().UTC())
	return user, err == nil
}
func (a *API) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/healthz" || strings.HasPrefix(path, "/api/v1/auth/") {
			next.ServeHTTP(w, r)
			return
		}
		if _, ok := a.currentUser(r); !ok {
			writeError(w, 401, "UNAUTHENTICATED", "please sign in")
			return
		}
		next.ServeHTTP(w, r)
	})
}
