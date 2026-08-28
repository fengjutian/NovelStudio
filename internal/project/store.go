package project

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var ErrNotFound = errors.New("project not found")

type Store interface {
	List(context.Context) ([]Project, error)
	Get(context.Context, string) (Project, error)
	Create(context.Context, CreateInput) (Project, error)
	Delete(context.Context, string) error
	Tree(context.Context, string) ([]ContentNode, error)
	CreateNode(context.Context, string, CreateNodeInput) (ContentNode, error)
	UpdateNode(context.Context, string, UpdateNodeInput) (ContentNode, error)
	DeleteNode(context.Context, string) error
}

func (s *MemoryStore) CreateNode(_ context.Context, projectID string, input CreateNodeInput) (ContentNode, error) {
	if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.NodeType) == "" {
		return ContentNode{}, errors.New("title and nodeType are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[projectID]; !ok {
		return ContentNode{}, ErrNotFound
	}
	if input.Position < 1 {
		input.Position = len(s.nodes[projectID]) + 1
	}
	item := ContentNode{ID: s.nextID("nod"), ProjectID: projectID, ParentID: input.ParentID, NodeType: strings.ToUpper(input.NodeType), Title: strings.TrimSpace(input.Title), Position: input.Position, Metadata: input.Metadata}
	s.nodes[projectID] = append(s.nodes[projectID], item)
	return item, nil
}
func (s *MemoryStore) UpdateNode(_ context.Context, id string, input UpdateNodeInput) (ContentNode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for projectID, nodes := range s.nodes {
		for index, item := range nodes {
			if item.ID == id {
				if strings.TrimSpace(input.Title) != "" {
					item.Title = strings.TrimSpace(input.Title)
				}
				if input.Position > 0 {
					item.Position = input.Position
				}
			if input.Metadata != nil {
				item.Metadata = input.Metadata
			}
			if input.DocumentID != nil { item.DocumentID=input.DocumentID }
				s.nodes[projectID][index] = item
				return item, nil
			}
		}
	}
	return ContentNode{}, ErrNotFound
}
func (s *MemoryStore) DeleteNode(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	remove := map[string]bool{id: true}
	for {
		changed := false
		for _, nodes := range s.nodes {
			for _, node := range nodes {
				if node.ParentID != nil && remove[*node.ParentID] && !remove[node.ID] {
					remove[node.ID] = true
					changed = true
				}
			}
		}
		if !changed {
			break
		}
	}
	for projectID, nodes := range s.nodes {
		kept := nodes[:0]
		for _, node := range nodes {
			if remove[node.ID] {
				found = true
			} else {
				kept = append(kept, node)
			}
		}
		s.nodes[projectID] = kept
	}
	if !found {
		return ErrNotFound
	}
	return nil
}

type MemoryStore struct {
	mu       sync.RWMutex
	sequence atomic.Uint64
	projects map[string]Project
	nodes    map[string][]ContentNode
}

func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{projects: make(map[string]Project), nodes: make(map[string][]ContentNode)}
	s.seed()
	return s
}

func (s *MemoryStore) List(context.Context) ([]Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Project, 0, len(s.projects))
	for _, item := range s.projects {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return items, nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.projects[id]
	if !ok {
		return Project{}, ErrNotFound
	}
	return item, nil
}

func (s *MemoryStore) Create(_ context.Context, input CreateInput) (Project, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return Project{}, errors.New("name is required")
	}
	if !input.Type.Valid() {
		return Project{}, errors.New("unsupported project type")
	}
	now := time.Now().UTC()
	id := s.nextID("prj")
	item := Project{ID: id, Name: input.Name, Type: input.Type, Description: strings.TrimSpace(input.Description), Status: "DRAFT", CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	s.projects[id] = item
	s.nodes[id] = defaultTree(id, input.Type)
	s.mu.Unlock()
	return item, nil
}

func (s *MemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[id]; !ok {
		return ErrNotFound
	}
	delete(s.projects, id)
	delete(s.nodes, id)
	return nil
}

func (s *MemoryStore) Tree(_ context.Context, id string) ([]ContentNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.projects[id]; !ok {
		return nil, ErrNotFound
	}
	return append([]ContentNode(nil), s.nodes[id]...), nil
}

func (s *MemoryStore) nextID(prefix string) string {
	return fmt.Sprintf("%s_%06d", prefix, s.sequence.Add(1))
}

func (s *MemoryStore) seed() {
	_, _ = s.Create(context.Background(), CreateInput{Name: "长安风云", Type: TypeNovel, Description: "历史长篇小说创作项目"})
	_, _ = s.Create(context.Background(), CreateInput{Name: "银翼杀手电影解说", Type: TypeMovieCommentary, Description: "带时间码和事实引用的口播稿"})
	_, _ = s.Create(context.Background(), CreateInput{Name: "Content Studio API", Type: TypeTechnicalDocument, Description: "基于知识库校验的技术文档"})
}

func defaultTree(projectID string, projectType Type) []ContentNode {
	rootType, childType := "SECTION", "CHAPTER"
	if projectType == TypeMovieCommentary {
		rootType, childType = "ACT", "COMMENTARY_SEGMENT"
	}
	if projectType == TypeTechnicalDocument {
		rootType, childType = "MODULE", "SECTION"
	}
	rootID := projectID + "_root"
	return []ContentNode{
		{ID: rootID, ProjectID: projectID, NodeType: rootType, Title: "未命名结构", Position: 1},
		{ID: projectID + "_first", ProjectID: projectID, ParentID: &rootID, NodeType: childType, Title: "第一部分", Position: 1},
	}
}
