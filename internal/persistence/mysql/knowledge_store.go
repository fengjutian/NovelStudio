package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"novelstudio/internal/knowledge"
)

type KnowledgeStore struct{ DB *sql.DB }

var _ knowledge.Store = KnowledgeStore{}

func (s KnowledgeStore) CreateFileAsset(ctx context.Context, input knowledge.CreateFileAssetInput) (knowledge.FileAsset, error) {
	item := knowledge.FileAsset{ID: newID("fil"), ProjectID: input.ProjectID, Name: input.Name, Extension: input.Extension, MIMEType: input.MIMEType, Size: input.Size, Status: input.Status, StoragePath: input.StoragePath, SourceID: input.SourceID, CreatedAt: time.Now().UTC()}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO knowledge_files(id,project_id,name,extension,mime_type,size_bytes,status,storage_path,source_id,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, item.ID, item.ProjectID, item.Name, item.Extension, item.MIMEType, item.Size, item.Status, item.StoragePath, nullable(item.SourceID), item.CreatedAt)
	return item, err
}
func (s KnowledgeStore) ListFileAssets(ctx context.Context, projectID string) ([]knowledge.FileAsset, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,project_id,name,extension,mime_type,size_bytes,status,storage_path,COALESCE(source_id,''),created_at FROM knowledge_files WHERE (?='' OR project_id=?) ORDER BY created_at DESC`, projectID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []knowledge.FileAsset{}
	for rows.Next() {
		var item knowledge.FileAsset
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Name, &item.Extension, &item.MIMEType, &item.Size, &item.Status, &item.StoragePath, &item.SourceID, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (s KnowledgeStore) GetFileAsset(ctx context.Context, id string) (knowledge.FileAsset, error) {
	var item knowledge.FileAsset
	err := s.DB.QueryRowContext(ctx, `SELECT id,project_id,name,extension,mime_type,size_bytes,status,storage_path,COALESCE(source_id,''),created_at FROM knowledge_files WHERE id=?`, id).Scan(&item.ID, &item.ProjectID, &item.Name, &item.Extension, &item.MIMEType, &item.Size, &item.Status, &item.StoragePath, &item.SourceID, &item.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return knowledge.FileAsset{}, knowledge.ErrNotFound
	}
	return item, err
}
func (s KnowledgeStore) DeleteFileAsset(ctx context.Context, id string) (knowledge.FileAsset, error) {
	item, err := s.GetFileAsset(ctx, id)
	if err != nil {
		return knowledge.FileAsset{}, err
	}
	_, err = s.DB.ExecContext(ctx, `DELETE FROM knowledge_files WHERE id=?`, id)
	return item, err
}

func (s KnowledgeStore) CreateMemory(ctx context.Context, projectID string, input knowledge.CreateMemoryInput) (knowledge.MemoryEntry, error) {
	input.Type = strings.ToUpper(strings.TrimSpace(input.Type))
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || (input.Type != "CHARACTER" && input.Type != "PLACE" && input.Type != "TIMELINE" && input.Type != "PLOT" && input.Type != "FORESHADOW") {
		return knowledge.MemoryEntry{}, errors.New("valid type and name are required")
	}
	if input.Status == "" {
		input.Status = "ACTIVE"
	}
	now := time.Now().UTC()
	item := knowledge.MemoryEntry{ID: newID("mem"), ProjectID: projectID, Type: input.Type, Name: input.Name, Summary: strings.TrimSpace(input.Summary), Status: strings.ToUpper(input.Status), Attributes: input.Attributes, CreatedAt: now, UpdatedAt: now}
	raw, _ := json.Marshal(item.Attributes)
	_, err := s.DB.ExecContext(ctx, `INSERT INTO story_memories(id,project_id,memory_type,name,summary,status,attributes,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`, item.ID, item.ProjectID, item.Type, item.Name, item.Summary, item.Status, raw, now, now)
	return item, err
}
func (s KnowledgeStore) ListMemories(ctx context.Context, projectID, memoryType string) ([]knowledge.MemoryEntry, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,project_id,memory_type,name,summary,status,attributes,created_at,updated_at FROM story_memories WHERE project_id=? AND (?='' OR memory_type=?) ORDER BY updated_at DESC`, projectID, memoryType, memoryType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []knowledge.MemoryEntry{}
	for rows.Next() {
		var item knowledge.MemoryEntry
		var raw []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Type, &item.Name, &item.Summary, &item.Status, &raw, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(raw, &item.Attributes)
		items = append(items, item)
	}
	return items, rows.Err()
}
func (s KnowledgeStore) DeleteMemory(ctx context.Context, id string) error {
	result, err := s.DB.ExecContext(ctx, `DELETE FROM story_memories WHERE id=?`, id)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return knowledge.ErrNotFound
	}
	return nil
}

func (s KnowledgeStore) CreateFacts(ctx context.Context, projectID string, inputs []knowledge.CreateFactInput) ([]knowledge.Fact, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	items := []knowledge.Fact{}
	for _, input := range inputs {
		if strings.TrimSpace(input.Subject) == "" || strings.TrimSpace(input.Predicate) == "" || strings.TrimSpace(input.Object) == "" {
			continue
		}
		item := knowledge.Fact{ID: newID("fac"), ProjectID: projectID, Subject: input.Subject, Predicate: input.Predicate, Object: input.Object, SourceChunkID: input.SourceChunkID, Confidence: input.Confidence, Status: "PROPOSED", CreatedAt: now, UpdatedAt: now}
		_, err = tx.ExecContext(ctx, `INSERT INTO knowledge_facts(id,project_id,subject,predicate,object,source_version_id,confidence,status,created_at,updated_at) VALUES(?,?,?,?,?,NULL,?,'PROPOSED',?,?)`, item.ID, projectID, item.Subject, item.Predicate, item.Object, item.Confidence, now, now)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return items, nil
}
func (s KnowledgeStore) ListFacts(ctx context.Context, projectID string) ([]knowledge.Fact, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,project_id,subject,predicate,object,COALESCE(source_version_id,''),confidence,status,created_at,updated_at FROM knowledge_facts WHERE project_id=? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []knowledge.Fact{}
	for rows.Next() {
		var item knowledge.Fact
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Subject, &item.Predicate, &item.Object, &item.SourceChunkID, &item.Confidence, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (s KnowledgeStore) UpdateFactStatus(ctx context.Context, id, status string) (knowledge.Fact, error) {
	if status != "CONFIRMED" && status != "REJECTED" && status != "SUPERSEDED" {
		return knowledge.Fact{}, errors.New("invalid fact status")
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE knowledge_facts SET status=?,updated_at=? WHERE id=?`, status, time.Now().UTC(), id)
	if err != nil {
		return knowledge.Fact{}, err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return knowledge.Fact{}, knowledge.ErrNotFound
	}
	var item knowledge.Fact
	err = s.DB.QueryRowContext(ctx, `SELECT id,project_id,subject,predicate,object,COALESCE(source_version_id,''),confidence,status,created_at,updated_at FROM knowledge_facts WHERE id=?`, id).Scan(&item.ID, &item.ProjectID, &item.Subject, &item.Predicate, &item.Object, &item.SourceChunkID, &item.Confidence, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s KnowledgeStore) ListSources(ctx context.Context, projectID string) ([]knowledge.Source, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,knowledge_base_id,name,source_type,version,authority,status,content_hash,created_at FROM knowledge_sources WHERE project_id=? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]knowledge.Source, 0)
	for rows.Next() {
		var item knowledge.Source
		if err := rows.Scan(&item.ID, &item.KnowledgeBaseID, &item.Name, &item.SourceType, &item.Version, &item.Authority, &item.Status, &item.ContentHash, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s KnowledgeStore) CreateSource(ctx context.Context, input knowledge.CreateSourceInput) (knowledge.Source, []knowledge.Chunk, error) {
	input.Name, input.Content = strings.TrimSpace(input.Name), strings.TrimSpace(input.Content)
	if input.ProjectID == "" || input.Name == "" || input.Content == "" {
		return knowledge.Source{}, nil, errors.New("projectId, name and content are required")
	}
	if input.Authority == "" {
		input.Authority = knowledge.AuthorityReference
	}
	sourceType := strings.ToUpper(strings.TrimSpace(input.SourceType))
	if sourceType == "" {
		sourceType = "TEXT"
	}
	now := time.Now().UTC()
	item := knowledge.Source{ID: newID("src"), KnowledgeBaseID: "kb_" + input.ProjectID, Name: input.Name, SourceType: sourceType, Version: strings.TrimSpace(input.Version), Authority: input.Authority, Status: "READY", ContentHash: contentHash(input.Content), CreatedAt: now}
	chunks := buildChunks(item.ID, input.Content, now)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return knowledge.Source{}, nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO knowledge_sources(id,project_id,knowledge_base_id,name,source_type,version,authority,status,content_hash,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, item.ID, input.ProjectID, item.KnowledgeBaseID, item.Name, item.SourceType, item.Version, item.Authority, item.Status, item.ContentHash, now); err != nil {
		return knowledge.Source{}, nil, err
	}
	statement, err := tx.PrepareContext(ctx, `INSERT INTO knowledge_chunks(id,source_id,position,content,token_hint,created_at) VALUES(?,?,?,?,?,?)`)
	if err != nil {
		return knowledge.Source{}, nil, err
	}
	defer statement.Close()
	for _, chunk := range chunks {
		if _, err := statement.ExecContext(ctx, chunk.ID, chunk.SourceID, chunk.Position, chunk.Content, chunk.TokenHint, chunk.CreatedAt); err != nil {
			return knowledge.Source{}, nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return knowledge.Source{}, nil, err
	}
	return item, chunks, nil
}

func (s KnowledgeStore) Search(ctx context.Context, projectID, query string, limit int) ([]knowledge.SearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("query is required")
	}
	if limit < 1 || limit > 20 {
		limit = 5
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT c.id,c.source_id,c.position,c.content,c.token_hint,c.created_at,
       s.id,s.knowledge_base_id,s.name,s.source_type,s.version,s.authority,s.status,s.content_hash,s.created_at,
       CASE s.authority WHEN 'OFFICIAL' THEN 4 WHEN 'VERIFIED' THEN 3 WHEN 'INTERNAL' THEN 2 ELSE 1 END authority_rank
FROM knowledge_chunks c JOIN knowledge_sources s ON s.id=c.source_id
WHERE s.project_id=? AND c.content LIKE CONCAT('%',?,'%')
ORDER BY authority_rank DESC, c.position ASC LIMIT ?`, projectID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]knowledge.SearchHit, 0)
	for rows.Next() {
		var hit knowledge.SearchHit
		var rank int
		if err := rows.Scan(&hit.Chunk.ID, &hit.Chunk.SourceID, &hit.Chunk.Position, &hit.Chunk.Content, &hit.Chunk.TokenHint, &hit.Chunk.CreatedAt, &hit.Source.ID, &hit.Source.KnowledgeBaseID, &hit.Source.Name, &hit.Source.SourceType, &hit.Source.Version, &hit.Source.Authority, &hit.Source.Status, &hit.Source.ContentHash, &hit.Source.CreatedAt, &rank); err != nil {
			return nil, err
		}
		hit.Score, hit.MatchType = 1, "KEYWORD"
		items = append(items, hit)
	}
	return items, rows.Err()
}

func buildChunks(sourceID, content string, now time.Time) []knowledge.Chunk {
	paragraphs := strings.FieldsFunc(content, func(r rune) bool { return r == '\n' || r == '\r' })
	chunks := make([]knowledge.Chunk, 0)
	var buffer strings.Builder
	flush := func() {
		text := strings.TrimSpace(buffer.String())
		if text != "" {
			chunks = append(chunks, knowledge.Chunk{ID: newID("chk"), SourceID: sourceID, Position: len(chunks) + 1, Content: text, TokenHint: utf8.RuneCountInString(text), CreatedAt: now})
		}
		buffer.Reset()
	}
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		if buffer.Len() > 0 && buffer.Len()+len(paragraph) > 800 {
			flush()
		}
		if buffer.Len() > 0 {
			buffer.WriteByte('\n')
		}
		buffer.WriteString(paragraph)
	}
	flush()
	return chunks
}
