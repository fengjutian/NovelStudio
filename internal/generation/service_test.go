package generation_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"novelstudio/internal/generation"
	"novelstudio/internal/llm"
)

type limitingProvider struct {
	active    atomic.Int64
	maxActive atomic.Int64
	release   chan struct{}
}

func (p *limitingProvider) Generate(ctx context.Context, _ llm.GenerateRequest) (llm.GenerateResponse, error) {
	active := p.active.Add(1)
	defer p.active.Add(-1)
	for {
		maximum := p.maxActive.Load()
		if active <= maximum || p.maxActive.CompareAndSwap(maximum, active) {
			break
		}
	}
	select {
	case <-p.release:
		return llm.GenerateResponse{Content: "generated"}, nil
	case <-ctx.Done():
		return llm.GenerateResponse{}, ctx.Err()
	}
}

func (*limitingProvider) HealthCheck(context.Context) error { return nil }

func TestGenerateLimitsGlobalModelConcurrency(t *testing.T) {
	fake := &limitingProvider{release: make(chan struct{})}
	service := generation.Service{
		ProviderName: "test",
		Provider:     fake,
		Models:       map[generation.Operation]string{generation.OperationWrite: "writer-model"},
		Slots:        make(chan struct{}, 2),
	}
	done := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func() {
			_, err := service.Generate(context.Background(), generation.Request{Operation: generation.OperationWrite, ProjectType: "NOVEL", Instruction: "write"})
			done <- err
		}()
	}
	deadline := time.Now().Add(time.Second)
	for fake.active.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if active := fake.active.Load(); active != 2 {
		t.Fatalf("active model calls = %d, want 2", active)
	}
	time.Sleep(20 * time.Millisecond)
	if maximum := fake.maxActive.Load(); maximum != 2 {
		t.Fatalf("maximum model concurrency = %d, want 2", maximum)
	}
	close(fake.release)
	for i := 0; i < 3; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}

type provider struct{ request llm.GenerateRequest }

func (p *provider) Generate(_ context.Context, request llm.GenerateRequest) (llm.GenerateResponse, error) {
	p.request = request
	return llm.GenerateResponse{Content: "# Generated", InputTokens: 12, OutputTokens: 4, RequestID: "req"}, nil
}
func (*provider) HealthCheck(context.Context) error { return nil }

func TestGenerateUsesOperationModelAndEvidence(t *testing.T) {
	fake := &provider{}
	service := generation.Service{ProviderName: "test", Provider: fake, Models: map[generation.Operation]string{generation.OperationWrite: "writer-model"}}
	result, err := service.Generate(context.Background(), generation.Request{Operation: generation.OperationWrite, ProjectType: "TECHNICAL_DOCUMENT", TypePrompt: "Do not invent API parameters.", Instruction: "Write API docs", Evidence: []generation.Evidence{{ID: "c1", Source: "API", Authority: "OFFICIAL", Content: "project_id is required"}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Model != "writer-model" || result.PromptVersion != "v1" || len(result.EvidenceIDs) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if !strings.Contains(fake.request.Messages[1].Content, "project_id is required") {
		t.Fatal("evidence was not included in the prompt")
	}
	if !strings.Contains(fake.request.Messages[0].Content, "Do not invent API parameters.") {
		t.Fatal("content type prompt was not included in the system prompt")
	}
}
