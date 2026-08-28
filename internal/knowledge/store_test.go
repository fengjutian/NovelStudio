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
