package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"novelstudio/internal/document"
)

type DocumentStore struct{ DB *sql.DB }

var _ document.Store = DocumentStore{}

func (s DocumentStore) List(ctx context.Context, projectID string) ([]document.Document, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,project_id,title,COALESCE(current_version_id,''),version_count,created_at,updated_at FROM documents WHERE project_id=? AND deleted_at IS NULL ORDER BY updated_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]document.Document, 0)
	for rows.Next() {
		var item document.Document
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Title, &item.CurrentVersionID, &item.VersionCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s DocumentStore) Get(ctx context.Context, id string) (document.Document, error) {
	var item document.Document
	err := s.DB.QueryRowContext(ctx, `SELECT id,project_id,title,COALESCE(current_version_id,''),version_count,created_at,updated_at FROM documents WHERE id=? AND deleted_at IS NULL`, id).Scan(&item.ID, &item.ProjectID, &item.Title, &item.CurrentVersionID, &item.VersionCount, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return document.Document{}, document.ErrNotFound
	}
	return item, err
}

func (s DocumentStore) Create(ctx context.Context, input document.CreateInput) (document.Document, document.Version, error) {
	input.Title = strings.TrimSpace(input.Title)
	if input.ProjectID == "" || input.Title == "" {
		return document.Document{}, document.Version{}, errors.New("projectId and title are required")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return document.Document{}, document.Version{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	item := document.Document{ID: newID("doc"), ProjectID: input.ProjectID, Title: input.Title, VersionCount: 1, CreatedAt: now, UpdatedAt: now}
	version := document.Version{ID: newID("ver"), DocumentID: item.ID, VersionNumber: 1, Content: input.Content, ContentHash: contentHash(input.Content), Reason: "INITIAL", AuthorType: "HUMAN", CreatedAt: now}
	item.CurrentVersionID = version.ID
	if _, err := tx.ExecContext(ctx, `INSERT INTO documents(id,project_id,title,current_version_id,version_count,created_at,updated_at) VALUES(?,?,?,NULL,0,?,?)`, item.ID, item.ProjectID, item.Title, now, now); err != nil {
		return document.Document{}, document.Version{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO document_versions(id,document_id,parent_version_id,version_number,content,content_hash,reason,author_type,created_at) VALUES(?,?,NULL,?,?,?,?,?,?)`, version.ID, item.ID, 1, version.Content, version.ContentHash, version.Reason, version.AuthorType, now); err != nil {
		return document.Document{}, document.Version{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE documents SET current_version_id=?,version_count=1 WHERE id=?`, version.ID, item.ID); err != nil {
		return document.Document{}, document.Version{}, err
	}
	if err := tx.Commit(); err != nil {
		return document.Document{}, document.Version{}, err
	}
	return item, version, nil
}

func (s DocumentStore) Versions(ctx context.Context, documentID string) ([]document.Version, error) {
	if _, err := s.Get(ctx, documentID); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id,document_id,parent_version_id,version_number,content,content_hash,reason,author_type,created_at FROM document_versions WHERE document_id=? ORDER BY version_number DESC`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]document.Version, 0)
	for rows.Next() {
		item, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s DocumentStore) CreateVersion(ctx context.Context, documentID string, input document.CreateVersionInput) (document.Version, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return document.Version{}, err
	}
	defer tx.Rollback()
	var currentID string
	var number int
	err = tx.QueryRowContext(ctx, `SELECT current_version_id,version_count FROM documents WHERE id=? AND deleted_at IS NULL FOR UPDATE`, documentID).Scan(&currentID, &number)
	if errors.Is(err, sql.ErrNoRows) {
		return document.Version{}, document.ErrNotFound
	}
	if err != nil {
		return document.Version{}, err
	}
	if input.ExpectedVersionID != "" && input.ExpectedVersionID != currentID {
		return document.Version{}, document.ErrConflict
	}
	var currentHash string
	if err := tx.QueryRowContext(ctx, `SELECT content_hash FROM document_versions WHERE id=?`, currentID).Scan(&currentHash); err != nil {
		return document.Version{}, err
	}
	if currentHash == contentHash(input.Content) {
		return document.Version{}, document.ErrNoChange
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = "EDIT"
	}
	author := strings.ToUpper(strings.TrimSpace(input.AuthorType))
	if author == "" {
		author = "HUMAN"
	}
	now := time.Now().UTC()
	version := document.Version{ID: newID("ver"), DocumentID: documentID, ParentVersionID: &currentID, VersionNumber: number + 1, Content: input.Content, ContentHash: contentHash(input.Content), Reason: reason, AuthorType: author, CreatedAt: now}
	if _, err := tx.ExecContext(ctx, `INSERT INTO document_versions(id,document_id,parent_version_id,version_number,content,content_hash,reason,author_type,created_at) VALUES(?,?,?,?,?,?,?,?,?)`, version.ID, documentID, currentID, version.VersionNumber, version.Content, version.ContentHash, version.Reason, version.AuthorType, now); err != nil {
		return document.Version{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE documents SET current_version_id=?,version_count=?,updated_at=? WHERE id=?`, version.ID, version.VersionNumber, now, documentID); err != nil {
		return document.Version{}, err
	}
	if err := tx.Commit(); err != nil {
		return document.Version{}, err
	}
	return version, nil
}

func (s DocumentStore) Restore(ctx context.Context, documentID, versionID string) (document.Version, error) {
	var content string
	err := s.DB.QueryRowContext(ctx, `SELECT content FROM document_versions WHERE id=? AND document_id=?`, versionID, documentID).Scan(&content)
	if errors.Is(err, sql.ErrNoRows) {
		return document.Version{}, document.ErrVersionFound
	}
	if err != nil {
		return document.Version{}, err
	}
	return s.CreateVersion(ctx, documentID, document.CreateVersionInput{Content: content, Reason: "RESTORE:" + versionID, AuthorType: "HUMAN"})
}

type rowScanner interface{ Scan(...any) error }

func scanVersion(row rowScanner) (document.Version, error) {
	var item document.Version
	var parent sql.NullString
	err := row.Scan(&item.ID, &item.DocumentID, &parent, &item.VersionNumber, &item.Content, &item.ContentHash, &item.Reason, &item.AuthorType, &item.CreatedAt)
	if parent.Valid {
		item.ParentVersionID = &parent.String
	}
	return item, err
}
