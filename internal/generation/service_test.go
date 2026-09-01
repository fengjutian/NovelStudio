package generation_test

import (
	"context"
	"strings"
	"testing"

	"novelstudio/internal/generation"
	"novelstudio/internal/llm"
)

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
