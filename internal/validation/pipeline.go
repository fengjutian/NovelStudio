package validation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"novelstudio/internal/airun"
	"novelstudio/internal/llm"
	"novelstudio/internal/promptcatalog"
)

type ReviewRequest struct {
	Text       string     `json:"text"`
	Task       string     `json:"task"`
	Evidence   []Evidence `json:"evidence"`
	Dimensions []string   `json:"dimensions"`
	ProjectID  string     `json:"projectId,omitempty"`
	TaskID     string     `json:"taskId,omitempty"`
}

type Evidence struct {
	ID        string `json:"id"`
	Source    string `json:"source"`
	Authority string `json:"authority"`
	Content   string `json:"content"`
}

type Reviewer interface {
	Review(context.Context, ReviewRequest) (Result, Run, error)
}

type NamedReviewer struct {
	Name     string
	Reviewer Reviewer
}

type Run struct {
	Role         string        `json:"role"`
	Provider     string        `json:"provider"`
	Model        string        `json:"model"`
	RequestID    string        `json:"requestId,omitempty"`
	InputTokens  int           `json:"inputTokens"`
	OutputTokens int           `json:"outputTokens"`
	Latency      time.Duration `json:"latencyMs"`
	Status       string        `json:"status"`
	Error        string        `json:"error,omitempty"`
}

type PipelineResult struct {
	Result        Result            `json:"result"`
	Reviews       map[string]Result `json:"reviews"`
	Runs          []Run             `json:"runs"`
	Disagreements int               `json:"disagreements"`
	Gate          Gate              `json:"gate"`
}

type Gate struct {
	Status  string   `json:"status"`
	Reasons []string `json:"reasons"`
}

type Pipeline struct {
	Validators []NamedReviewer
	Judge      *NamedReviewer
}

func (p Pipeline) Validate(ctx context.Context, request ReviewRequest) (PipelineResult, error) {
	if strings.TrimSpace(request.Text) == "" {
		return PipelineResult{}, errors.New("text is required")
	}
	if len(p.Validators) == 0 {
		return PipelineResult{}, errors.New("at least one validator is required")
	}
	reviews := make(map[string]Result, len(p.Validators))
	runs := make([]Run, 0, len(p.Validators)+1)
	all := make([]Result, 0, len(p.Validators))
	type reviewOutcome struct {
		name   string
		result Result
		run    Run
		err    error
	}
	outcomes := make(chan reviewOutcome, len(p.Validators))
	for _, validator := range p.Validators {
		go func(validator NamedReviewer) {
			result, run, err := validator.Reviewer.Review(ctx, request)
			run.Role = validator.Name
			outcomes <- reviewOutcome{name: validator.Name, result: result, run: run, err: err}
		}(validator)
	}
	for range p.Validators {
		outcome := <-outcomes
		runs = append(runs, outcome.run)
		if outcome.err != nil {
			return PipelineResult{Reviews: reviews, Runs: runs}, fmt.Errorf("validator %s: %w", outcome.name, outcome.err)
		}
		reviews[outcome.name] = outcome.result
		all = append(all, outcome.result)
	}
	merged, disputed := merge(all)
	if len(disputed) > 0 && p.Judge != nil {
		judgeRequest := request
		judgeRequest.Task += "\n请仅裁决以下有分歧的问题，权威证据优先：\n" + marshal(disputed)
		judged, run, err := p.Judge.Reviewer.Review(ctx, judgeRequest)
		run.Role = p.Judge.Name
		runs = append(runs, run)
		if err != nil {
			return PipelineResult{Reviews: reviews, Runs: runs, Disagreements: len(disputed)}, fmt.Errorf("judge: %w", err)
		}
		reviews[p.Judge.Name] = judged
		merged.Issues = append(merged.Issues, judged.Issues...)
	}
	merged.Score = aggregateScore(all, merged.Issues)
	merged.Verdict = verdict(merged.Score, merged.Issues)
	merged.Dimensions = aggregateDimensions(all)
	return PipelineResult{Result: merged, Reviews: reviews, Runs: runs, Disagreements: len(disputed), Gate: qualityGate(merged)}, nil
}

type ModelReviewer struct {
	ProviderName string
	Provider     llm.Provider
	Model        string
	Role         string
	Prompts      promptcatalog.Catalog
	Recorder     airun.Recorder
}

func (r ModelReviewer) Review(ctx context.Context, request ReviewRequest) (result Result, run Run, err error) {
	start := time.Now()
	run = Run{Provider: r.ProviderName, Model: r.Model, Status: "FAILED"}
	payload, err := json.Marshal(request)
	if err != nil {
		return result, run, err
	}
	promptName := "validator"
	if strings.EqualFold(r.Role, "judge") {
		promptName = "judge"
	}
	system, promptVersion, promptErr := r.Prompts.Load(promptName)
	if promptErr != nil {
		return result, run, promptErr
	}
	response, err := r.Provider.Generate(ctx, llm.GenerateRequest{
		Model: r.Model, Temperature: 0, MaxTokens: 4000, ResponseSchema: resultSchema(),
		Messages: []llm.Message{
			{Role: "system", Content: system},
			{Role: "user", Content: string(payload)},
		},
	})
	run.Latency = time.Since(start) / time.Millisecond
	if err != nil {
		run.Error = err.Error()
		r.record(ctx, request, promptVersion, response, time.Since(start).Milliseconds(), "FAILED", err.Error())
		return result, run, err
	}
	run.RequestID, run.InputTokens, run.OutputTokens = response.RequestID, response.InputTokens, response.OutputTokens
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		run.Error = "invalid structured response"
		return result, run, fmt.Errorf("decode structured validation: %w", err)
	}
	normalize(&result)
	run.Status = "SUCCESS"
	r.record(ctx, request, promptVersion, response, int64(run.Latency), "SUCCESS", "")
	return result, run, nil
}

func (r ModelReviewer) record(ctx context.Context, request ReviewRequest, promptVersion string, response llm.GenerateResponse, latency int64, status, errorText string) {
	if r.Recorder == nil {
		return
	}
	_ = r.Recorder.Record(ctx, airun.Run{ID: airun.NewID(), ProjectID: request.ProjectID, TaskID: request.TaskID, Role: strings.ToUpper(r.Role), Provider: r.ProviderName, Model: r.Model, PromptVersion: promptVersion, RequestID: response.RequestID, InputTokens: response.InputTokens, OutputTokens: response.OutputTokens, LatencyMs: latency, Status: status, Error: errorText, CreatedAt: time.Now().UTC()})
}

func merge(results []Result) (Result, []Issue) {
	counts := make(map[string]int)
	issues := make(map[string]Issue)
	for _, result := range results {
		seen := make(map[string]bool)
		for _, issue := range result.Issues {
			key := issueKey(issue)
			if !seen[key] {
				counts[key]++
				seen[key] = true
			}
			if previous, ok := issues[key]; !ok || issue.Confidence > previous.Confidence {
				issues[key] = issue
			}
		}
	}
	consensus := make([]Issue, 0)
	disputed := make([]Issue, 0)
	threshold := 2
	if len(results) == 1 {
		threshold = 1
	}
	for key, issue := range issues {
		if counts[key] >= threshold {
			consensus = append(consensus, issue)
		} else if issue.Severity == SeverityCritical || issue.Severity == SeverityMajor {
			disputed = append(disputed, issue)
		} else {
			consensus = append(consensus, issue)
		}
	}
	return Result{Issues: consensus}, disputed
}

func aggregateScore(results []Result, issues []Issue) int {
	total := 0
	for _, result := range results {
		total += result.Score
	}
	score := total / len(results)
	for _, issue := range issues {
		if issue.Severity == SeverityCritical && score > 60 {
			score = 60
		}
	}
	return score
}

func aggregateDimensions(results []Result) map[string]int {
	totals, counts := map[string]int{}, map[string]int{}
	for _, result := range results {
		for name, score := range result.Dimensions {
			totals[name] += score
			counts[name]++
		}
	}
	for name := range totals {
		totals[name] /= counts[name]
	}
	return totals
}

func qualityGate(result Result) Gate {
	gate := Gate{Status: "PASS", Reasons: []string{}}
	for _, issue := range result.Issues {
		if issue.Severity == SeverityCritical {
			gate.Status = "FAIL"
			gate.Reasons = append(gate.Reasons, "存在关键问题："+issue.Type)
		}
	}
	if result.Score < 70 {
		gate.Status = "FAIL"
		gate.Reasons = append(gate.Reasons, "综合评分低于 70")
	} else if result.Score < 80 && gate.Status == "PASS" {
		gate.Status = "WARNING"
		gate.Reasons = append(gate.Reasons, "综合评分低于 80")
	}
	return gate
}

func verdict(score int, issues []Issue) string {
	return qualityGate(Result{Score: score, Issues: issues}).Status
}
func issueKey(issue Issue) string {
	return strings.ToLower(strings.TrimSpace(issue.Type) + "|" + strings.TrimSpace(issue.Claim))
}
func marshal(value any) string { raw, _ := json.Marshal(value); return string(raw) }
func normalize(result *Result) {
	if result.Score < 0 {
		result.Score = 0
	}
	if result.Score > 100 {
		result.Score = 100
	}
	if result.Dimensions == nil {
		result.Dimensions = map[string]int{}
	}
	if result.Issues == nil {
		result.Issues = []Issue{}
	}
}

func resultSchema() map[string]any {
	return map[string]any{
		"type": "object", "additionalProperties": false,
		"required": []string{"score", "verdict", "dimensions", "issues"},
		"properties": map[string]any{
			"score":   map[string]any{"type": "integer", "minimum": 0, "maximum": 100},
			"verdict": map[string]any{"type": "string", "enum": []string{"PASS", "WARNING", "FAIL"}},
			"dimensions": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"groundedness", "consistency", "completeness", "terminology", "style"},
				"properties": map[string]any{
					"groundedness": map[string]any{"type": "integer", "minimum": 0, "maximum": 100},
					"consistency":  map[string]any{"type": "integer", "minimum": 0, "maximum": 100},
					"completeness": map[string]any{"type": "integer", "minimum": 0, "maximum": 100},
					"terminology":  map[string]any{"type": "integer", "minimum": 0, "maximum": 100},
					"style":        map[string]any{"type": "integer", "minimum": 0, "maximum": 100},
				},
			},
			"issues": map[string]any{"type": "array", "items": map[string]any{
				"type": "object", "additionalProperties": false,
				"required": []string{"id", "type", "severity", "claim", "explanation", "suggestedFix", "evidenceIds", "confidence", "startCharacter", "endCharacter"},
				"properties": map[string]any{
					"id": map[string]any{"type": "string"}, "type": map[string]any{"type": "string"}, "severity": map[string]any{"type": "string", "enum": []string{"CRITICAL", "MAJOR", "MINOR"}}, "claim": map[string]any{"type": "string"}, "explanation": map[string]any{"type": "string"}, "suggestedFix": map[string]any{"type": "string"}, "evidenceIds": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}, "confidence": map[string]any{"type": "number"}, "startCharacter": map[string]any{"type": "integer"}, "endCharacter": map[string]any{"type": "integer"},
				},
			}},
		},
	}
}
