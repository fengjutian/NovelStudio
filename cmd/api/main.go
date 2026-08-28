package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"novelstudio/internal/httpapi"
	"novelstudio/internal/project"
)

func main() {
	addr := env("HTTP_ADDR", ":8080")
	store := project.NewMemoryStore()
	handler := httpapi.New(store, slog.Default())

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	slog.Info("AI Content Studio API started", "addr", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
