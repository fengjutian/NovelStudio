package document_test

import (
	"context"
	"errors"
	"testing"

	"novelstudio/internal/document"
)

func TestVersionsAreImmutableAndRestoreCreatesVersion(t *testing.T) {
	store := document.NewMemoryStore()
	doc, first, err := store.Create(context.Background(), document.CreateInput{ProjectID: "p1", Title: "指南", Content: "第一版"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateVersion(context.Background(), doc.ID, document.CreateVersionInput{Content: "第二版", Reason: "EDIT"})
	if err != nil {
		t.Fatal(err)
	}
	restored, err := store.Restore(context.Background(), doc.ID, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.VersionNumber != 3 || restored.Content != "第一版" || restored.ParentVersionID == nil || *restored.ParentVersionID != second.ID {
		t.Fatalf("unexpected restored version: %#v", restored)
	}
	versions, _ := store.Versions(context.Background(), doc.ID)
	if len(versions) != 3 || versions[2].Content != "第一版" {
		t.Fatalf("versions were mutated: %#v", versions)
	}
}

func TestUnchangedContentIsRejected(t *testing.T) {
	store := document.NewMemoryStore()
	doc, _, _ := store.Create(context.Background(), document.CreateInput{ProjectID: "p1", Title: "指南", Content: "相同"})
	_, err := store.CreateVersion(context.Background(), doc.ID, document.CreateVersionInput{Content: "相同"})
	if !errors.Is(err, document.ErrNoChange) {
		t.Fatalf("error = %v, want ErrNoChange", err)
	}
}
