package document

import "time"

type Document struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"projectId"`
	Title            string    `json:"title"`
	CurrentVersionID string    `json:"currentVersionId,omitempty"`
	VersionCount     int       `json:"versionCount"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type Version struct {
	ID              string    `json:"id"`
	DocumentID      string    `json:"documentId"`
	ParentVersionID *string   `json:"parentVersionId,omitempty"`
	VersionNumber   int       `json:"versionNumber"`
	Content         string    `json:"content"`
	ContentHash     string    `json:"contentHash"`
	Reason          string    `json:"reason"`
	AuthorType      string    `json:"authorType"`
	CreatedAt       time.Time `json:"createdAt"`
}

type CreateInput struct {
	ProjectID string `json:"projectId"`
	Title     string `json:"title"`
	Content   string `json:"content"`
}

type CreateVersionInput struct {
	Content    string `json:"content"`
	Reason     string `json:"reason"`
	AuthorType string `json:"authorType"`
}
