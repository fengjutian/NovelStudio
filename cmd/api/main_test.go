package main

import (
	"context"
	"testing"

	"novelstudio/internal/auth"
)

func TestParseEnvLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantKey   string
		wantValue string
		wantOK    bool
		wantErr   bool
	}{
		{name: "plain", line: "MYSQL_DSN=user:pass@tcp(localhost:3306)/db?parseTime=true", wantKey: "MYSQL_DSN", wantValue: "user:pass@tcp(localhost:3306)/db?parseTime=true", wantOK: true},
		{name: "double quoted", line: `HTTP_ADDR=":9090"`, wantKey: "HTTP_ADDR", wantValue: ":9090", wantOK: true},
		{name: "single quoted", line: "LLM_API_KEY='secret value'", wantKey: "LLM_API_KEY", wantValue: "secret value", wantOK: true},
		{name: "export", line: "export AI_TASK_TIMEOUT=10m", wantKey: "AI_TASK_TIMEOUT", wantValue: "10m", wantOK: true},
		{name: "comment", line: " # ignored", wantOK: false},
		{name: "invalid", line: "NOT_AN_ASSIGNMENT", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value, ok, err := parseEnvLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseEnvLine() error = %v, wantErr %v", err, tt.wantErr)
			}
			if key != tt.wantKey || value != tt.wantValue || ok != tt.wantOK {
				t.Fatalf("parseEnvLine() = (%q, %q, %v), want (%q, %q, %v)", key, value, ok, tt.wantKey, tt.wantValue, tt.wantOK)
			}
		})
	}
}

func TestSeedDefaultUser(t *testing.T) {
	t.Setenv("DEFAULT_USER_NAME", "Admin")
	t.Setenv("DEFAULT_USER_EMAIL", "admin@example.com")
	t.Setenv("DEFAULT_USER_PASSWORD", "initial-password")
	store := auth.NewMemoryStore()

	if err := seedDefaultUser(store); err != nil {
		t.Fatal(err)
	}
	user, hash, err := store.UserByEmail(context.Background(), "ADMIN@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.Name != "Admin" || !auth.CheckPassword(hash, "initial-password") {
		t.Fatalf("unexpected seeded user: %#v", user)
	}

	t.Setenv("DEFAULT_USER_PASSWORD", "replacement-password")
	if err := seedDefaultUser(store); err != nil {
		t.Fatal(err)
	}
	_, hash, err = store.UserByEmail(context.Background(), "admin@example.com")
	if err != nil || !auth.CheckPassword(hash, "initial-password") {
		t.Fatal("existing default user's password was overwritten")
	}
}
