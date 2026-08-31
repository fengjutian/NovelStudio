package project_test

import (
	"context"
	"errors"
	"testing"

	"novelstudio/internal/project"
)

func TestUpdateProject(t *testing.T) {
	store := project.NewMemoryStore()
	created, err := store.Create(context.Background(), project.CreateInput{Name: "旧名称", Type: project.TypeNovel, Description: "旧说明"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.Update(context.Background(), created.ID, project.UpdateInput{Name: " 新名称 ", Description: " 新说明 "})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "新名称" || updated.Description != "新说明" || updated.Type != created.Type {
		t.Fatalf("updated = %#v", updated)
	}
	if _, err := store.Update(context.Background(), created.ID, project.UpdateInput{}); err == nil {
		t.Fatal("empty name should fail")
	}
	if _, err := store.Update(context.Background(), "missing", project.UpdateInput{Name: "x"}); !errors.Is(err, project.ErrNotFound) {
		t.Fatalf("missing project error = %v", err)
	}
}
