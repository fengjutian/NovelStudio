package canon

import (
	"context"
	"testing"
)

func TestImpactAnalysisPropagatesThroughChapters(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	_ = s.RegisterDependencies(ctx, ChapterDependencies{ProjectID: "p", ChapterVersionID: "ch7", CanonVersion: 2, FactIDs: []string{"fact-injury"}})
	_ = s.RegisterDependencies(ctx, ChapterDependencies{ProjectID: "p", ChapterVersionID: "ch9", CanonVersion: 3, DependsOnChapterVersionIDs: []string{"ch7"}})
	_ = s.RegisterDependencies(ctx, ChapterDependencies{ProjectID: "p", ChapterVersionID: "ch12", CanonVersion: 4, DependsOnChapterVersionIDs: []string{"ch9"}})
	got, err := s.AnalyzeImpact(ctx, ImpactRequest{ProjectID: "p", ChangedFactIDs: []string{"fact-injury"}, TriggerCanonVersion: 5, MarkStale: true})
	if err != nil || len(got.Affected) != 3 {
		t.Fatalf("got=%#v err=%v", got, err)
	}
	stale, _ := s.ListStale(ctx, "p", "main")
	if len(stale) != 3 {
		t.Fatalf("stale=%#v", stale)
	}
	if len(got.Affected[2].Paths) == 0 {
		t.Fatalf("missing propagation path: %#v", got.Affected)
	}
	resolved, err := s.ResolveStale(ctx, "p", "main", "ch12", ResolveStaleInput{Status: "VALID", ResolvedBy: "author", Note: "review passed"})
	if err != nil || resolved.ResolvedAt == nil || resolved.Status != "VALID" {
		t.Fatalf("resolved=%#v err=%v", resolved, err)
	}
}
