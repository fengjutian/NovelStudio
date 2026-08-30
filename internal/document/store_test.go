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

func TestStaleVersionIsRejected(t *testing.T) {
	store := document.NewMemoryStore()
	doc, first, _ := store.Create(context.Background(), document.CreateInput{ProjectID: "p1", Title: "Guide", Content: "v1"})
	_, err := store.CreateVersion(context.Background(), doc.ID, document.CreateVersionInput{Content: "v2", ExpectedVersionID: first.ID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateVersion(context.Background(), doc.ID, document.CreateVersionInput{Content: "stale edit", ExpectedVersionID: first.ID})
	if !errors.Is(err, document.ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
}

func TestListKeepsCreationOrderAfterEdit(t *testing.T) {
	store := document.NewMemoryStore()
	first, firstVersion, err := store.Create(context.Background(), document.CreateInput{ProjectID: "p1", Title: "第一章", Content: "初稿"})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := store.Create(context.Background(), document.CreateInput{ProjectID: "p1", Title: "第二章", Content: "初稿"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateVersion(context.Background(), first.ID, document.CreateVersionInput{Content: "修改稿", ExpectedVersionID: firstVersion.ID}); err != nil {
		t.Fatal(err)
	}
	items, err := store.List(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != first.ID || items[1].ID != second.ID {
		t.Fatalf("document order changed after edit: %#v", items)
	}
}
