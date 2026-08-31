package project

import "time"

type Type string

const (
	TypeNovel             Type = "NOVEL"
	TypeMovieCommentary   Type = "MOVIE_COMMENTARY"
	TypeTechnicalDocument Type = "TECHNICAL_DOCUMENT"
)

func (t Type) Valid() bool {
	return t != ""
}

type ContentType struct {
	Code        Type      `json:"code"`
	Name        string    `json:"name"`
	Icon        string    `json:"icon"`
	Accent      string    `json:"accent"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateContentTypeInput struct {
	Code        Type   `json:"code"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Accent      string `json:"accent"`
	Description string `json:"description"`
}

type UpdateContentTypeInput struct {
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Accent      string `json:"accent"`
	Description string `json:"description"`
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

type UpdateInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ContentNode struct {
	ID         string         `json:"id"`
	ProjectID  string         `json:"projectId"`
	ParentID   *string        `json:"parentId,omitempty"`
	DocumentID *string        `json:"documentId,omitempty"`
	NodeType   string         `json:"nodeType"`
	Title      string         `json:"title"`
	Position   int            `json:"position"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}
type CreateNodeInput struct {
	ParentID *string        `json:"parentId"`
	NodeType string         `json:"nodeType"`
	Title    string         `json:"title"`
	Position int            `json:"position"`
	Metadata map[string]any `json:"metadata"`
}
type UpdateNodeInput struct {
	Title      string         `json:"title"`
	Position   int            `json:"position"`
	Metadata   map[string]any `json:"metadata"`
	DocumentID *string        `json:"documentId"`
}
