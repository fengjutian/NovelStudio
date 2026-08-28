package llm

import "context"

type Role string

const (
	RolePlanner   Role = "PLANNER"
	RoleWriter    Role = "WRITER"
	RolePolisher  Role = "POLISHER"
	RoleExtractor Role = "EXTRACTOR"
	RoleValidator Role = "VALIDATOR"
	RoleJudge     Role = "JUDGE"
	RoleRepair    Role = "REPAIR"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GenerateRequest struct {
	Model          string         `json:"model"`
	Messages       []Message      `json:"messages"`
	ResponseSchema map[string]any `json:"responseSchema,omitempty"`
	Temperature    float64        `json:"temperature"`
	MaxTokens      int            `json:"maxTokens"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type GenerateResponse struct {
	Content      string `json:"content"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
	RequestID    string `json:"requestId"`
}

type Provider interface {
	Generate(context.Context, GenerateRequest) (GenerateResponse, error)
	HealthCheck(context.Context) error
}

type Route struct {
	Role      Role       `json:"role"`
	Primary   ModelRef   `json:"primary"`
	Fallbacks []ModelRef `json:"fallbacks"`
}

type ModelRef struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}
