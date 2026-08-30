package document

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
)

var (
	ErrNotFound     = errors.New("document not found")
	ErrVersionFound = errors.New("document version not found")
	ErrNoChange     = errors.New("content has not changed")
	ErrConflict     = errors.New("document was changed by another editor")
)

type Store interface {
	List(context.Context, string) ([]Document, error)
	Get(context.Context, string) (Document, error)
	Create(context.Context, CreateInput) (Document, Version, error)
	Delete(context.Context, string) error
	Versions(context.Context, string) ([]Version, error)
	CreateVersion(context.Context, string, CreateVersionInput) (Version, error)
	Restore(context.Context, string, string) (Version, error)
}

type MemoryStore struct {
	mu        sync.RWMutex
	sequence  atomic.Uint64
	documents map[string]Document
	versions  map[string][]Version
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{documents: make(map[string]Document), versions: make(map[string][]Version)}
}

func (s *MemoryStore) List(_ context.Context, projectID string) ([]Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Document, 0)
	for _, item := range s.documents {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return items, nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.documents[id]
	if !ok {
		return Document{}, ErrNotFound
	}
	return item, nil
}

func (s *MemoryStore) Create(_ context.Context, input CreateInput) (Document, Version, error) {
	input.Title = strings.TrimSpace(input.Title)
	if strings.TrimSpace(input.ProjectID) == "" || input.Title == "" {
		return Document{}, Version{}, errors.New("projectId and title are required")
	}
	now := time.Now().UTC()
	documentID := s.nextID("doc")
	item := Document{ID: documentID, ProjectID: input.ProjectID, Title: input.Title, CreatedAt: now, UpdatedAt: now}
	version := newVersion(s.nextID("ver"), item.ID, nil, 1, input.Content, "INITIAL", "HUMAN", now)
	item.CurrentVersionID = version.ID
	item.VersionCount = 1
	s.mu.Lock()
	s.documents[item.ID] = item
	s.versions[item.ID] = []Version{version}
	s.mu.Unlock()
	return item, version, nil
}

func (s *MemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.documents[id]; !ok {
		return ErrNotFound
	}
	delete(s.documents, id)
	delete(s.versions, id)
	return nil
}

func (s *MemoryStore) Versions(_ context.Context, documentID string) ([]Version, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.documents[documentID]; !ok {
		return nil, ErrNotFound
	}
	items := append([]Version(nil), s.versions[documentID]...)
	sort.Slice(items, func(i, j int) bool { return items[i].VersionNumber > items[j].VersionNumber })
	return items, nil
}

func (s *MemoryStore) CreateVersion(_ context.Context, documentID string, input CreateVersionInput) (Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.documents[documentID]
	if !ok {
		return Version{}, ErrNotFound
	}
	items := s.versions[documentID]
	latest := items[len(items)-1]
	if input.ExpectedVersionID != "" && input.ExpectedVersionID != latest.ID {
		return Version{}, ErrConflict
	}
	if hash(input.Content) == latest.ContentHash {
		return Version{}, ErrNoChange
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = "EDIT"
	}
	authorType := strings.ToUpper(strings.TrimSpace(input.AuthorType))
	if authorType == "" {
		authorType = "HUMAN"
	}
	version := newVersion(s.nextID("ver"), documentID, &latest.ID, latest.VersionNumber+1, input.Content, reason, authorType, time.Now().UTC())
	s.versions[documentID] = append(items, version)
	item.CurrentVersionID = version.ID
	item.VersionCount++
	item.UpdatedAt = version.CreatedAt
	s.documents[documentID] = item
	return version, nil
}

func (s *MemoryStore) Restore(ctx context.Context, documentID, versionID string) (Version, error) {
	s.mu.RLock()
	var target *Version
	for i := range s.versions[documentID] {
		if s.versions[documentID][i].ID == versionID {
			copy := s.versions[documentID][i]
			target = &copy
			break
		}
	}
	s.mu.RUnlock()
	if target == nil {
		return Version{}, ErrVersionFound
	}
	return s.CreateVersion(ctx, documentID, CreateVersionInput{Content: target.Content, Reason: "RESTORE:" + target.ID, AuthorType: "HUMAN"})
}

func (s *MemoryStore) nextID(prefix string) string {
	return fmt.Sprintf("%s_%06d", prefix, s.sequence.Add(1))
}

func newVersion(id, documentID string, parentID *string, number int, content, reason, authorType string, now time.Time) Version {
	return Version{ID: id, DocumentID: documentID, ParentVersionID: parentID, VersionNumber: number, Content: content, ContentHash: hash(content), Reason: reason, AuthorType: authorType, CreatedAt: now}
}

func hash(content string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
}
