package workflow_test

import (
	"context"
	"testing"

	"novelstudio/internal/generation"
	"novelstudio/internal/llm"
	"novelstudio/internal/validation"
	"novelstudio/internal/workflow"
)

type provider struct{ calls int }

func (p *provider) Generate(context.Context, llm.GenerateRequest) (llm.GenerateResponse, error) {
	p.calls++
	if p.calls == 1 {
		return llm.GenerateResponse{Content: "draft"}, nil
	}
	return llm.GenerateResponse{Content: "repaired"}, nil
}
func (*provider) HealthCheck(context.Context) error { return nil }

type reviewer struct{ calls int }

func (r *reviewer) Review(context.Context, validation.ReviewRequest) (validation.Result, validation.Run, error) {
	r.calls++
	if r.calls == 1 {
		return validation.Result{Score: 50, Issues: []validation.Issue{{Type: "fact", Severity: validation.SeverityCritical, Claim: "bad"}}}, validation.Run{}, nil
	}
	return validation.Result{Score: 95, Issues: []validation.Issue{}}, validation.Run{}, nil
}

func TestQualityWorkflowRepairsAndRevalidates(t *testing.T) {
	p, r := &provider{}, &reviewer{}
	generator := &generation.Service{ProviderName: "test", Provider: p, Models: map[generation.Operation]string{generation.OperationWrite: "w", generation.OperationRepair: "r"}}
	validator := &validation.Pipeline{Validators: []validation.NamedReviewer{{Name: "v", Reviewer: r}}}
	flow := workflow.QualityWorkflow{Generator: generator, Validator: validator, MaxRepairs: 2}
	result, err := flow.Run(context.Background(), generation.Request{Operation: generation.OperationWrite, Instruction: "write"}, validation.ReviewRequest{}, func(int, string) {})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "repaired" || result.Validation.Gate.Status != "PASS" || len(result.Repairs) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
}
