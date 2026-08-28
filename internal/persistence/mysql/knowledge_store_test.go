package mysql

import (
	"strings"
	"testing"
	"time"
)

func TestBuildChunksPreservesParagraphs(t *testing.T) {
	content := strings.Repeat("甲", 500) + "\n" + strings.Repeat("乙", 500)
	chunks := buildChunks("source", content, time.Now())
	if len(chunks) != 2 {
		t.Fatalf("chunk count = %d, want 2", len(chunks))
	}
	if chunks[0].Position != 1 || chunks[1].Position != 2 {
		t.Fatalf("unexpected positions: %#v", chunks)
	}
}
