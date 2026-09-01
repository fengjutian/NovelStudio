package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"novelstudio/internal/project"
)

type ProjectStore struct{ DB *sql.DB }

var _ project.Store = ProjectStore{}

func (s ProjectStore) CreateNode(ctx context.Context, projectID string, input project.CreateNodeInput) (project.ContentNode, error) {
	if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.NodeType) == "" {
		return project.ContentNode{}, errors.New("title and nodeType are required")
	}
	if input.Position < 1 {
		_ = s.DB.QueryRowContext(ctx, `SELECT COALESCE(MAX(position),0)+1 FROM content_nodes WHERE project_id=? AND parent_id <=> ?`, projectID, input.ParentID).Scan(&input.Position)
	}
	item := project.ContentNode{ID: newID("nod"), ProjectID: projectID, ParentID: input.ParentID, NodeType: strings.ToUpper(input.NodeType), Title: strings.TrimSpace(input.Title), Position: input.Position, Metadata: input.Metadata}
	now := time.Now().UTC()
	_, err := s.DB.ExecContext(ctx, `INSERT INTO content_nodes(id,project_id,parent_id,node_type,title,position,metadata,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`, item.ID, projectID, input.ParentID, item.NodeType, item.Title, item.Position, metadataJSON(item.Metadata), now, now)
	return item, err
}
func (s ProjectStore) UpdateNode(ctx context.Context, id string, input project.UpdateNodeInput) (project.ContentNode, error) {
	var current project.ContentNode
	var parent sql.NullString
	var metadata []byte
	var documentID sql.NullString
	err := s.DB.QueryRowContext(ctx, `SELECT id,project_id,parent_id,document_id,node_type,title,position,metadata FROM content_nodes WHERE id=?`, id).Scan(&current.ID, &current.ProjectID, &parent, &documentID, &current.NodeType, &current.Title, &current.Position, &metadata)
	if errors.Is(err, sql.ErrNoRows) {
		return project.ContentNode{}, project.ErrNotFound
	}
	if err != nil {
		return project.ContentNode{}, err
	}
	if parent.Valid {
		current.ParentID = &parent.String
	}
	if documentID.Valid {
		current.DocumentID = &documentID.String
	}
	if strings.TrimSpace(input.Title) != "" {
		current.Title = strings.TrimSpace(input.Title)
	}
	if input.Position > 0 {
		current.Position = input.Position
	}
	if input.Metadata != nil {
		current.Metadata = input.Metadata
	} else if len(metadata) > 0 {
		_ = json.Unmarshal(metadata, &current.Metadata)
	}
	if input.DocumentID != nil {
		if strings.TrimSpace(*input.DocumentID) == "" {
			current.DocumentID = nil
		} else {
			current.DocumentID = input.DocumentID
		}
	}
	var documentValue any
	if current.DocumentID != nil {
		documentValue = *current.DocumentID
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE content_nodes SET title=?,position=?,metadata=?,document_id=?,updated_at=? WHERE id=?`, current.Title, current.Position, metadataJSON(current.Metadata), documentValue, time.Now().UTC(), id)
	return current, err
}
func (s ProjectStore) DeleteNode(ctx context.Context, id string) error {
	result, err := s.DB.ExecContext(ctx, `DELETE FROM content_nodes WHERE id=?`, id)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return project.ErrNotFound
	}
	return nil
}

func (s ProjectStore) List(ctx context.Context) ([]project.Project, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, name, project_type, description, status, created_at, updated_at FROM projects WHERE deleted_at IS NULL ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]project.Project, 0)
	for rows.Next() {
		var item project.Project
		if err := rows.Scan(&item.ID, &item.Name, &item.Type, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s ProjectStore) Get(ctx context.Context, id string) (project.Project, error) {
	var item project.Project
	err := s.DB.QueryRowContext(ctx, `SELECT id, name, project_type, description, status, created_at, updated_at FROM projects WHERE id = ? AND deleted_at IS NULL`, id).Scan(&item.ID, &item.Name, &item.Type, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return project.Project{}, project.ErrNotFound
	}
	return item, err
}

func (s ProjectStore) Create(ctx context.Context, input project.CreateInput) (project.Project, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return project.Project{}, errors.New("name is required")
	}
	var typeCount int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM content_types WHERE code=?`, input.Type).Scan(&typeCount); err != nil {
		return project.Project{}, err
	}
	if typeCount == 0 {
		return project.Project{}, errors.New("unsupported project type")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return project.Project{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	item := project.Project{ID: newID("prj"), Name: input.Name, Type: input.Type, Description: strings.TrimSpace(input.Description), Status: "DRAFT", CreatedAt: now, UpdatedAt: now}
	if _, err := tx.ExecContext(ctx, `INSERT INTO projects(id,name,project_type,description,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, item.ID, item.Name, item.Type, item.Description, item.Status, now, now); err != nil {
		return project.Project{}, err
	}
	rootType, childType := "SECTION", "CHAPTER"
	if input.Type == project.TypeMovieCommentary {
		rootType, childType = "ACT", "COMMENTARY_SEGMENT"
	} else if input.Type == project.TypeTVCommentary {
		rootType, childType = "SEASON", "EPISODE"
	} else if input.Type == project.TypeTechnicalDocument {
		rootType, childType = "MODULE", "SECTION"
	}
	rootID, childID := newID("nod"), newID("nod")
	if _, err := tx.ExecContext(ctx, `INSERT INTO content_nodes(id,project_id,parent_id,node_type,title,position,created_at,updated_at) VALUES(?,?,NULL,?,?,?,?,?)`, rootID, item.ID, rootType, "未命名结构", 1, now, now); err != nil {
		return project.Project{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO content_nodes(id,project_id,parent_id,node_type,title,position,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, childID, item.ID, rootID, childType, "第一部分", 1, now, now); err != nil {
		return project.Project{}, err
	}
	if err := tx.Commit(); err != nil {
		return project.Project{}, err
	}
	return item, nil
}

func (s ProjectStore) Update(ctx context.Context, id string, input project.UpdateInput) (project.Project, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return project.Project{}, errors.New("name is required")
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE projects SET name=?,description=?,updated_at=? WHERE id=? AND deleted_at IS NULL`, input.Name, strings.TrimSpace(input.Description), time.Now().UTC(), id)
	if err != nil {
		return project.Project{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return project.Project{}, err
	}
	if count == 0 {
		return project.Project{}, project.ErrNotFound
	}
	return s.Get(ctx, id)
}

func (s ProjectStore) ListContentTypes(ctx context.Context) ([]project.ContentType, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT code,name,icon,accent,description,prompt,created_at,updated_at FROM content_types ORDER BY created_at,code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []project.ContentType{}
	for rows.Next() {
		var item project.ContentType
		if err := rows.Scan(&item.Code, &item.Name, &item.Icon, &item.Accent, &item.Description, &item.Prompt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s ProjectStore) CreateContentType(ctx context.Context, input project.CreateContentTypeInput) (project.ContentType, error) {
	item, err := project.NormalizeContentType(input.Code, input.Name, input.Icon, input.Accent, input.Description, input.Prompt)
	if err != nil {
		return project.ContentType{}, err
	}
	now := time.Now().UTC()
	item.CreatedAt, item.UpdatedAt = now, now
	_, err = s.DB.ExecContext(ctx, `INSERT INTO content_types(code,name,icon,accent,description,prompt,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, item.Code, item.Name, item.Icon, item.Accent, item.Description, item.Prompt, now, now)
	return item, err
}

func (s ProjectStore) UpdateContentType(ctx context.Context, code project.Type, input project.UpdateContentTypeInput) (project.ContentType, error) {
	item, err := project.NormalizeContentType(code, input.Name, input.Icon, input.Accent, input.Description, input.Prompt)
	if err != nil {
		return project.ContentType{}, err
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE content_types SET name=?,icon=?,accent=?,description=?,prompt=?,updated_at=? WHERE code=?`, item.Name, item.Icon, item.Accent, item.Description, item.Prompt, time.Now().UTC(), code)
	if err != nil {
		return project.ContentType{}, err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return project.ContentType{}, project.ErrContentTypeNotFound
	}
	err = s.DB.QueryRowContext(ctx, `SELECT code,name,icon,accent,description,prompt,created_at,updated_at FROM content_types WHERE code=?`, code).Scan(&item.Code, &item.Name, &item.Icon, &item.Accent, &item.Description, &item.Prompt, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s ProjectStore) DeleteContentType(ctx context.Context, code project.Type) error {
	var used int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects WHERE project_type=?`, code).Scan(&used); err != nil {
		return err
	}
	if used > 0 {
		return project.ErrContentTypeInUse
	}
	result, err := s.DB.ExecContext(ctx, `DELETE FROM content_types WHERE code=?`, code)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return project.ErrContentTypeNotFound
	}
	return nil
}

func (s ProjectStore) Delete(ctx context.Context, id string) error {
	result, err := s.DB.ExecContext(ctx, `UPDATE projects SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, time.Now().UTC(), time.Now().UTC(), id)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return project.ErrNotFound
	}
	return nil
}

func (s ProjectStore) Tree(ctx context.Context, id string) ([]project.ContentNode, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id, project_id, parent_id, document_id, node_type, title, position, metadata FROM content_nodes WHERE project_id = ? ORDER BY parent_id, position`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]project.ContentNode, 0)
	for rows.Next() {
		var item project.ContentNode
		var parent sql.NullString
		var documentID sql.NullString
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &parent, &documentID, &item.NodeType, &item.Title, &item.Position, &metadata); err != nil {
			return nil, err
		}
		if parent.Valid {
			item.ParentID = &parent.String
		}
		if documentID.Valid {
			item.DocumentID = &documentID.String
		}
		if len(metadata) > 0 {
			_ = json.Unmarshal(metadata, &item.Metadata)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
