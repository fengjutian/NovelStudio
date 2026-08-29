package knowledge

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

var ErrNotFound = errors.New("knowledge source not found")

type Store interface {
	ListSources(context.Context, string) ([]Source, error)
	CreateSource(context.Context, CreateSourceInput) (Source, []Chunk, error)
	DeleteSource(context.Context, string) error
	Search(context.Context, string, string, int) ([]SearchHit, error)
	CreateFacts(context.Context, string, []CreateFactInput) ([]Fact, error)
	ListFacts(context.Context, string) ([]Fact, error)
	UpdateFactStatus(context.Context, string, string) (Fact, error)
	CreateMemory(context.Context, string, CreateMemoryInput) (MemoryEntry, error)
	ListMemories(context.Context, string, string) ([]MemoryEntry, error)
	DeleteMemory(context.Context, string) error
	CreateFileAsset(context.Context, CreateFileAssetInput) (FileAsset, error)
	ListFileAssets(context.Context, string) ([]FileAsset, error)
	GetFileAsset(context.Context, string) (FileAsset, error)
	DeleteFileAsset(context.Context, string) (FileAsset, error)
}

type MemoryStore struct {
	mu        sync.RWMutex
	sequence  atomic.Uint64
	sources   map[string]Source
	projectID map[string]string
	chunks    map[string][]Chunk
	facts     map[string]Fact
	memories  map[string]MemoryEntry
	files     map[string]FileAsset
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{sources: make(map[string]Source), projectID: make(map[string]string), chunks: make(map[string][]Chunk), facts: make(map[string]Fact), memories: make(map[string]MemoryEntry), files: make(map[string]FileAsset)}
}

func (s *MemoryStore) CreateFileAsset(_ context.Context, input CreateFileAssetInput) (FileAsset, error) {
	if strings.TrimSpace(input.ProjectID) == "" || strings.TrimSpace(input.Name) == "" {
		return FileAsset{}, errors.New("projectId and name are required")
	}
	item := FileAsset{ID: s.nextID("fil"), ProjectID: input.ProjectID, Name: input.Name, Extension: input.Extension, MIMEType: input.MIMEType, Size: input.Size, Status: input.Status, StoragePath: input.StoragePath, SourceID: input.SourceID, CreatedAt: time.Now().UTC()}
	s.mu.Lock()
	s.files[item.ID] = item
	s.mu.Unlock()
	return item, nil
}
func (s *MemoryStore) ListFileAssets(_ context.Context, projectID string) ([]FileAsset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []FileAsset{}
	for _, item := range s.files {
		if projectID == "" || item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}
func (s *MemoryStore) GetFileAsset(_ context.Context, id string) (FileAsset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.files[id]
	if !ok {
		return FileAsset{}, ErrNotFound
	}
	return item, nil
}
func (s *MemoryStore) DeleteFileAsset(_ context.Context, id string) (FileAsset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.files[id]
	if !ok {
		return FileAsset{}, ErrNotFound
	}
	delete(s.files, id)
	return item, nil
}

func (s *MemoryStore) DeleteSource(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sources[id]; !ok {
		return ErrNotFound
	}
	delete(s.sources, id)
	delete(s.projectID, id)
	delete(s.chunks, id)
	return nil
}

func (s *MemoryStore) CreateMemory(_ context.Context, projectID string, input CreateMemoryInput) (MemoryEntry, error) {
	input.Type = strings.ToUpper(strings.TrimSpace(input.Type))
	input.Name = strings.TrimSpace(input.Name)
	if !validMemoryType(input.Type) || input.Name == "" {
		return MemoryEntry{}, errors.New("valid type and name are required")
	}
	now := time.Now().UTC()
	if input.Status == "" {
		input.Status = "ACTIVE"
	}
	item := MemoryEntry{ID: s.nextID("mem"), ProjectID: projectID, Type: input.Type, Name: input.Name, Summary: strings.TrimSpace(input.Summary), Status: strings.ToUpper(input.Status), Attributes: input.Attributes, CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	s.memories[item.ID] = item
	s.mu.Unlock()
	return item, nil
}
func (s *MemoryStore) ListMemories(_ context.Context, projectID, memoryType string) ([]MemoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	memoryType = strings.ToUpper(memoryType)
	items := []MemoryEntry{}
	for _, item := range s.memories {
		if item.ProjectID == projectID && (memoryType == "" || item.Type == memoryType) {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return items, nil
}
func (s *MemoryStore) DeleteMemory(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.memories[id]; !ok {
		return ErrNotFound
	}
	delete(s.memories, id)
	return nil
}
func validMemoryType(value string) bool {
	return value == "CHARACTER" || value == "PLACE" || value == "TIMELINE" || value == "PLOT" || value == "FORESHADOW"
}

func (s *MemoryStore) CreateFacts(_ context.Context, projectID string, inputs []CreateFactInput) ([]Fact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	items := make([]Fact, 0, len(inputs))
	for _, input := range inputs {
		if strings.TrimSpace(input.Subject) == "" || strings.TrimSpace(input.Predicate) == "" || strings.TrimSpace(input.Object) == "" {
			continue
		}
		item := Fact{ID: s.nextID("fac"), ProjectID: projectID, Subject: input.Subject, Predicate: input.Predicate, Object: input.Object, SourceChunkID: input.SourceChunkID, Confidence: input.Confidence, Status: "PROPOSED", CreatedAt: now, UpdatedAt: now}
		s.facts[item.ID] = item
		items = append(items, item)
	}
	return items, nil
}
func (s *MemoryStore) ListFacts(_ context.Context, projectID string) ([]Fact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []Fact{}
	for _, item := range s.facts {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items, nil
}
func (s *MemoryStore) UpdateFactStatus(_ context.Context, id, status string) (Fact, error) {
	if status != "CONFIRMED" && status != "REJECTED" && status != "SUPERSEDED" {
		return Fact{}, errors.New("invalid fact status")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.facts[id]
	if !ok {
		return Fact{}, ErrNotFound
	}
	item.Status = status
	item.UpdatedAt = time.Now().UTC()
	s.facts[id] = item
	return item, nil
}

func (s *MemoryStore) ListSources(_ context.Context, projectID string) ([]Source, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Source, 0)
	for id, item := range s.sources {
		if s.projectID[id] == projectID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *MemoryStore) CreateSource(_ context.Context, input CreateSourceInput) (Source, []Chunk, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Content = strings.TrimSpace(input.Content)
	if input.ProjectID == "" || input.Name == "" || input.Content == "" {
		return Source{}, nil, errors.New("projectId, name and content are required")
	}
	if input.Authority == "" {
		input.Authority = AuthorityReference
	}
	id := s.nextID("src")
	now := time.Now().UTC()
	source := Source{ID: id, KnowledgeBaseID: "kb_" + input.ProjectID, Name: input.Name, SourceType: fallback(input.SourceType, "TEXT"), Version: strings.TrimSpace(input.Version), Authority: input.Authority, Status: "READY", ContentHash: hash(input.Content), CreatedAt: now}
	chunks := split(s, id, input.Content, now)
	s.mu.Lock()
	s.sources[id] = source
	s.projectID[id] = input.ProjectID
	s.chunks[id] = chunks
	s.mu.Unlock()
	return source, chunks, nil
}

func (s *MemoryStore) Search(_ context.Context, projectID, query string, limit int) ([]SearchHit, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, errors.New("query is required")
	}
	if limit < 1 || limit > 20 {
		limit = 5
	}
	terms := strings.Fields(query)
	s.mu.RLock()
	defer s.mu.RUnlock()
	var hits []SearchHit
	for sourceID, chunks := range s.chunks {
		if s.projectID[sourceID] != projectID {
			continue
		}
		for _, chunk := range chunks {
			text := strings.ToLower(chunk.Content)
			matched := 0
			for _, term := range terms {
				if strings.Contains(text, term) {
					matched++
				}
			}
			if matched > 0 {
				hits = append(hits, SearchHit{Chunk: chunk, Source: s.sources[sourceID], Score: float64(matched) / float64(len(terms)), MatchType: "KEYWORD"})
			}
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return authorityRank(hits[i].Source.Authority) > authorityRank(hits[j].Source.Authority)
		}
		return hits[i].Score > hits[j].Score
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

func split(s *MemoryStore, sourceID, content string, now time.Time) []Chunk {
	paragraphs := strings.FieldsFunc(content, func(r rune) bool { return r == '\n' || r == '\r' })
	chunks := make([]Chunk, 0)
	var buffer strings.Builder
	flush := func() {
		text := strings.TrimSpace(buffer.String())
		if text == "" {
			return
		}
		chunks = append(chunks, Chunk{ID: s.nextID("chk"), SourceID: sourceID, Position: len(chunks) + 1, Content: text, TokenHint: utf8.RuneCountInString(text), CreatedAt: now})
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
			buffer.WriteString("\n")
		}
		buffer.WriteString(paragraph)
	}
	flush()
	return chunks
}

func (s *MemoryStore) nextID(prefix string) string {
	return fmt.Sprintf("%s_%06d", prefix, s.sequence.Add(1))
}
func hash(content string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(content))) }
func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return strings.ToUpper(strings.TrimSpace(value))
}
func authorityRank(value Authority) int {
	switch value {
	case AuthorityOfficial:
		return 4
	case AuthorityVerified:
		return 3
	case AuthorityInternal:
		return 2
	default:
		return 1
	}
}
