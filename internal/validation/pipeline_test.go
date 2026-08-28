package validation_test

import (
	"context"
	"testing"

	"novelstudio/internal/validation"
)

type reviewer struct {
	result validation.Result
	calls  *int
}

func (r reviewer) Review(context.Context, validation.ReviewRequest) (validation.Result, validation.Run, error) {
	if r.calls != nil {
		*r.calls = *r.calls + 1
	}
	return r.result, validation.Run{Status: "SUCCESS"}, nil
}

func TestPipelineConsensusCriticalIssueFailsGate(t *testing.T) {
	issue := validation.Issue{ID: "a", Type: "factual_conflict", Severity: validation.SeverityCritical, Claim: "batch delete is supported", Confidence: .9}
	pipeline := validation.Pipeline{Validators: []validation.NamedReviewer{
		{Name: "A", Reviewer: reviewer{result: validation.Result{Score: 90, Dimensions: map[string]int{"groundedness": 90}, Issues: []validation.Issue{issue}}}},
		{Name: "B", Reviewer: reviewer{result: validation.Result{Score: 85, Dimensions: map[string]int{"groundedness": 80}, Issues: []validation.Issue{issue}}}},
	}}
	result, err := pipeline.Validate(context.Background(), validation.ReviewRequest{Text: "batch delete is supported"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Gate.Status != "FAIL" || result.Result.Score != 60 || len(result.Result.Issues) != 1 {
		t.Fatalf("unexpected pipeline result: %#v", result)
	}
}

func TestPipelineUsesJudgeOnlyForMajorDisagreement(t *testing.T) {
	judgeCalls := 0
	issue := validation.Issue{ID: "a", Type: "unsupported_claim", Severity: validation.SeverityMajor, Claim: "authentication required", Confidence: .8}
	pipeline := validation.Pipeline{
		Validators: []validation.NamedReviewer{
			{Name: "A", Reviewer: reviewer{result: validation.Result{Score: 75, Issues: []validation.Issue{issue}}}},
			{Name: "B", Reviewer: reviewer{result: validation.Result{Score: 95, Issues: []validation.Issue{}}}},
		},
		Judge: &validation.NamedReviewer{
			Name: "JUDGE",
			Reviewer: reviewer{
				calls:  &judgeCalls,
				result: validation.Result{Score: 90, Issues: []validation.Issue{issue}},
			},
		},
	}
	result, err := pipeline.Validate(context.Background(), validation.ReviewRequest{Text: "authentication required"})
	if err != nil {
		t.Fatal(err)
	}
	if judgeCalls != 1 || result.Disagreements != 1 || len(result.Result.Issues) != 1 {
		t.Fatalf("judgeCalls=%d result=%#v", judgeCalls, result)
	}
}

func TestMinorDisagreementDoesNotSpendJudgeCall(t *testing.T) {
	judgeCalls := 0
	minor := validation.Issue{Type: "style", Severity: validation.SeverityMinor, Claim: "long sentence"}
	pipeline := validation.Pipeline{
		Validators: []validation.NamedReviewer{
			{Name: "A", Reviewer: reviewer{result: validation.Result{Score: 88, Issues: []validation.Issue{minor}}}},
			{Name: "B", Reviewer: reviewer{result: validation.Result{Score: 92}}},
		},
		Judge: &validation.NamedReviewer{Name: "JUDGE", Reviewer: reviewer{calls: &judgeCalls}},
	}
	_, err := pipeline.Validate(context.Background(), validation.ReviewRequest{Text: "test text"})
	if err != nil {
		t.Fatal(err)
	}
	if judgeCalls != 0 {
		t.Fatalf("judge called %d times", judgeCalls)
	}
}
