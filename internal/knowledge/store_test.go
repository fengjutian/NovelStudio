package knowledge_test

import (
	"context"
	"testing"

	"novelstudio/internal/knowledge"
)

func TestSearchPrefersAuthoritativeSource(t *testing.T) {
	store := knowledge.NewMemoryStore()
	_, chunks, err := store.CreateSource(context.Background(), knowledge.CreateSourceInput{ProjectID: "p1", Name: "参考博客", Authority: knowledge.AuthorityReference, Content: "创建任务需要 project_id 参数。"})
	if err != nil || len(chunks) != 1 {
		t.Fatalf("create reference: chunks=%d err=%v", len(chunks), err)
	}
	_, _, err = store.CreateSource(context.Background(), knowledge.CreateSourceInput{ProjectID: "p1", Name: "官方接口", Authority: knowledge.AuthorityOfficial, Content: "创建任务需要 project_id 参数。"})
	if err != nil {
		t.Fatal(err)
	}
	hits, err := store.Search(context.Background(), "p1", "project_id", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 || hits[0].Source.Authority != knowledge.AuthorityOfficial {
		t.Fatalf("unexpected ranking: %#v", hits)
	}
}

func TestSearchIsProjectScoped(t *testing.T) {
	store := knowledge.NewMemoryStore()
	_, _, _ = store.CreateSource(context.Background(), knowledge.CreateSourceInput{ProjectID: "p1", Name: "A", Content: "唯一知识"})
	hits, err := store.Search(context.Background(), "p2", "唯一知识", 5)
	if err != nil || len(hits) != 0 {
		t.Fatalf("hits=%d err=%v", len(hits), err)
	}
}

func TestMemoryLifecycleIsProjectScoped(t *testing.T) {
	store := knowledge.NewMemoryStore()
	created, err := store.CreateMemory(context.Background(), "p1", knowledge.CreateMemoryInput{Type: "CHARACTER", Name: "沈砚", Summary: "县衙书吏"})
	if err != nil {
		t.Fatal(err)
	}
	items, _ := store.ListMemories(context.Background(), "p1", "CHARACTER")
	other, _ := store.ListMemories(context.Background(), "p2", "")
	if len(items) != 1 || items[0].ID != created.ID || len(other) != 0 {
		t.Fatalf("items=%#v other=%#v", items, other)
	}
	if err := store.DeleteMemory(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	items, _ = store.ListMemories(context.Background(), "p1", "")
	if len(items) != 0 {
		t.Fatalf("memory was not deleted: %#v", items)
	}
}

func TestFileAssetLifecycleAndProjectFilter(t *testing.T) {
	store := knowledge.NewMemoryStore()
	first, err := store.CreateFileAsset(context.Background(), knowledge.CreateFileAssetInput{
		ProjectID: "p1", Name: "manual.pdf", Extension: ".pdf", MIMEType: "application/pdf",
		Size: 2048, Status: "STORED", StoragePath: "p1/manual.pdf",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateFileAsset(context.Background(), knowledge.CreateFileAssetInput{
		ProjectID: "p2", Name: "notes.md", Extension: ".md", Size: 128,
		Status: "INDEXED", StoragePath: "p2/notes.md", SourceID: "src-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	all, _ := store.ListFileAssets(context.Background(), "")
	projectFiles, _ := store.ListFileAssets(context.Background(), "p1")
	if len(all) != 2 || len(projectFiles) != 1 || projectFiles[0].ID != first.ID {
		t.Fatalf("all=%#v projectFiles=%#v", all, projectFiles)
	}
	got, err := store.GetFileAsset(context.Background(), first.ID)
	if err != nil || got.Name != "manual.pdf" {
		t.Fatalf("got=%#v err=%v", got, err)
	}
	deleted, err := store.DeleteFileAsset(context.Background(), first.ID)
	if err != nil || deleted.ID != first.ID {
		t.Fatalf("deleted=%#v err=%v", deleted, err)
	}
	if _, err := store.GetFileAsset(context.Background(), first.ID); err == nil {
		t.Fatal("deleted file asset is still readable")
	}
}
