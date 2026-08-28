package knowledge

import "time"

type Authority string

const (
	AuthorityOfficial  Authority = "OFFICIAL"
	AuthorityVerified  Authority = "VERIFIED"
	AuthorityInternal  Authority = "INTERNAL"
	AuthorityReference Authority = "REFERENCE"
)

type Source struct {
	ID              string    `json:"id"`
	KnowledgeBaseID string    `json:"knowledgeBaseId"`
	Name            string    `json:"name"`
	SourceType      string    `json:"sourceType"`
	Version         string    `json:"version,omitempty"`
	Authority       Authority `json:"authority"`
	Status          string    `json:"status"`
	ContentHash     string    `json:"contentHash"`
	CreatedAt       time.Time `json:"createdAt"`
}

type CreateSourceInput struct {
	ProjectID  string    `json:"projectId"`
	Name       string    `json:"name"`
	SourceType string    `json:"sourceType"`
	Version    string    `json:"version"`
	Authority  Authority `json:"authority"`
	Content    string    `json:"content"`
}

type Chunk struct {
	ID        string    `json:"id"`
	SourceID  string    `json:"sourceId"`
	Position  int       `json:"position"`
	Content   string    `json:"content"`
	TokenHint int       `json:"tokenHint"`
	CreatedAt time.Time `json:"createdAt"`
}

type SearchHit struct {
	Chunk     Chunk   `json:"chunk"`
	Source    Source  `json:"source"`
	Score     float64 `json:"score"`
	MatchType string  `json:"matchType"`
}

type Fact struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"projectId"`
	Subject       string    `json:"subject"`
	Predicate     string    `json:"predicate"`
	Object        string    `json:"object"`
	SourceChunkID string    `json:"sourceChunkId"`
	Confidence    float64   `json:"confidence"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type CreateFactInput struct {
	Subject       string  `json:"subject"`
	Predicate     string  `json:"predicate"`
	Object        string  `json:"object"`
	SourceChunkID string  `json:"sourceChunkId"`
	Confidence    float64 `json:"confidence"`
}
