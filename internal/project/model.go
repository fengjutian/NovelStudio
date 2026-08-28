package project

import "time"

type Type string

const (
	TypeNovel             Type = "NOVEL"
	TypeMovieCommentary   Type = "MOVIE_COMMENTARY"
	TypeTechnicalDocument Type = "TECHNICAL_DOCUMENT"
)

func (t Type) Valid() bool {
	return t == TypeNovel || t == TypeMovieCommentary || t == TypeTechnicalDocument
}

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        Type      `json:"type"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateInput struct {
	Name        string `json:"name"`
	Type        Type   `json:"type"`
	Description string `json:"description"`
}

type ContentNode struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"projectId"`
	ParentID  *string        `json:"parentId,omitempty"`
	DocumentID *string      `json:"documentId,omitempty"`
	NodeType  string         `json:"nodeType"`
	Title     string         `json:"title"`
	Position  int            `json:"position"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}
type CreateNodeInput struct {
	ParentID *string        `json:"parentId"`
	NodeType string         `json:"nodeType"`
	Title    string         `json:"title"`
	Position int            `json:"position"`
	Metadata map[string]any `json:"metadata"`
}
type UpdateNodeInput struct {
	Title    string         `json:"title"`
	Position int            `json:"position"`
	Metadata map[string]any `json:"metadata"`
	DocumentID *string `json:"documentId"`
}
