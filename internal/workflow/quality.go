package workflow

import (
	"context"
	"encoding/json"
	"errors"

	"novelstudio/internal/generation"
	"novelstudio/internal/validation"
)

type QualityWorkflow struct {
	Generator  *generation.Service
	Validator  *validation.Pipeline
	MaxRepairs int
}

type Result struct {
	Content    string                    `json:"content"`
	Generation generation.Result         `json:"generation"`
	Validation validation.PipelineResult `json:"validation"`
	Repairs    []generation.Result       `json:"repairs"`
	Attempts   int                       `json:"attempts"`
}

func (w QualityWorkflow) Run(ctx context.Context, request generation.Request, review validation.ReviewRequest, progress func(int, string)) (Result, error) {
	if w.Generator == nil || w.Validator == nil {
		return Result{}, errors.New("generation and validation services are required")
	}
	generated, err := w.Generator.Generate(ctx, request)
	if err != nil {
		return Result{}, err
	}
	result := Result{Content: generated.Content, Generation: generated, Repairs: []generation.Result{}}
	maxRepairs := w.MaxRepairs
	if maxRepairs <= 0 || maxRepairs > 3 {
		maxRepairs = 2
	}
	for attempt := 0; ; attempt++ {
		progress(45+attempt*20, "多模型质量校验")
		review.Text = result.Content
		checked, err := w.Validator.Validate(ctx, review)
		if err != nil {
			return result, err
		}
		result.Validation, result.Attempts = checked, attempt+1
		if checked.Gate.Status != "FAIL" || attempt >= maxRepairs {
			return result, nil
		}
		progress(55+attempt*20, "Repair Agent 正在修复失败项")
		issues, _ := json.Marshal(checked.Result.Issues)
		repairRequest := request
		repairRequest.Operation = generation.OperationRepair
		repairRequest.Content = result.Content
		repairRequest.Instruction = "修复以下质量问题并输出完整正文：" + string(issues)
		repaired, err := w.Generator.Generate(ctx, repairRequest)
		if err != nil {
			return result, err
		}
		result.Content = repaired.Content
		result.Repairs = append(result.Repairs, repaired)
	}
}
