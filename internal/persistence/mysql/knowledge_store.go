package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"novelstudio/internal/knowledge"
)

type KnowledgeStore struct{ DB *sql.DB }

var _ knowledge.Store = KnowledgeStore{}

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
