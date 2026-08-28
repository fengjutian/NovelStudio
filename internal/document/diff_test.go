package document_test

import (
	"novelstudio/internal/document"
	"testing"
)

func TestDiffReportsAddedAndDeletedLines(t *testing.T) {
	result := document.Diff(document.Version{ID: "v1", Content: "one\ntwo\nthree"}, document.Version{ID: "v2", Content: "one\nsecond\nthree\nfour"})
	if result.Added != 2 || result.Deleted != 1 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if result.FromVersionID != "v1" || result.ToVersionID != "v2" {
		t.Fatalf("unexpected versions: %+v", result)
	}
}
