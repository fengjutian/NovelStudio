package airun

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

func NewID() string {
	raw := make([]byte, 12)
	_, _ = rand.Read(raw)
	return "run_" + hex.EncodeToString(raw)
}

type Run struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"projectId,omitempty"`
	TaskID        string    `json:"taskId,omitempty"`
	Role          string    `json:"role"`
	Provider      string    `json:"provider"`
	Model         string    `json:"model"`
	PromptVersion string    `json:"promptVersion"`
	RequestID     string    `json:"requestId,omitempty"`
	InputTokens   int       `json:"inputTokens"`
	OutputTokens  int       `json:"outputTokens"`
	LatencyMs     int64     `json:"latencyMs"`
	Status        string    `json:"status"`
	Error         string    `json:"error,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

type Recorder interface {
	Record(context.Context, Run) error
	List(context.Context, string, int) ([]Run, error)
}

type MemoryRecorder struct {
	mu   sync.RWMutex
	runs []Run
}

func NewMemoryRecorder() *MemoryRecorder { return &MemoryRecorder{} }
func (r *MemoryRecorder) Record(_ context.Context, run Run) error {
	r.mu.Lock()
	r.runs = append(r.runs, run)
	r.mu.Unlock()
	return nil
}
func (r *MemoryRecorder) List(_ context.Context, projectID string, limit int) ([]Run, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit < 1 {
		limit = 50
	}
	items := make([]Run, 0, limit)
	for i := len(r.runs) - 1; i >= 0 && len(items) < limit; i-- {
		if projectID == "" || r.runs[i].ProjectID == projectID {
			items = append(items, r.runs[i])
		}
	}
	return items, nil
}
