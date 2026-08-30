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
var ErrContentTypeNotFound = errors.New("content type not found")
var ErrContentTypeInUse = errors.New("content type is used by projects")

type Store interface {
	List(context.Context) ([]Project, error)
	Get(context.Context, string) (Project, error)
	Create(context.Context, CreateInput) (Project, error)
	Delete(context.Context, string) error
	Tree(context.Context, string) ([]ContentNode, error)
	CreateNode(context.Context, string, CreateNodeInput) (ContentNode, error)
	UpdateNode(context.Context, string, UpdateNodeInput) (ContentNode, error)
	DeleteNode(context.Context, string) error
	ListContentTypes(context.Context) ([]ContentType, error)
	CreateContentType(context.Context, CreateContentTypeInput) (ContentType, error)
	UpdateContentType(context.Context, Type, UpdateContentTypeInput) (ContentType, error)
	DeleteContentType(context.Context, Type) error
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
				if input.DocumentID != nil {
					item.DocumentID = input.DocumentID
				}
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
	types    map[Type]ContentType
}

func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{projects: make(map[string]Project), nodes: make(map[string][]ContentNode), types: make(map[Type]ContentType)}
	s.seedContentTypes()
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
	s.mu.RLock()
	_, typeExists := s.types[input.Type]
	s.mu.RUnlock()
	if !typeExists {
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

func (s *MemoryStore) ListContentTypes(_ context.Context) ([]ContentType, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]ContentType, 0, len(s.types))
	for _, item := range s.types {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

func NormalizeContentType(code Type, name, icon, accent, description string) (ContentType, error) {
	code = Type(strings.ToUpper(strings.TrimSpace(string(code))))
	name, icon, accent = strings.TrimSpace(name), strings.TrimSpace(icon), strings.ToLower(strings.TrimSpace(accent))
	if code == "" || name == "" {
		return ContentType{}, errors.New("code and name are required")
	}
	for _, r := range code {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return ContentType{}, errors.New("code only supports A-Z, 0-9 and underscore")
		}
	}
	if len(code) > 40 || len(name) > 80 {
		return ContentType{}, errors.New("code or name is too long")
	}
	if icon == "" {
		icon = string([]rune(name)[0])
	}
	if accent == "" {
		accent = "amber"
	}
	return ContentType{Code: code, Name: name, Icon: icon, Accent: accent, Description: strings.TrimSpace(description)}, nil
}

func (s *MemoryStore) CreateContentType(_ context.Context, input CreateContentTypeInput) (ContentType, error) {
	item, err := NormalizeContentType(input.Code, input.Name, input.Icon, input.Accent, input.Description)
	if err != nil {
		return ContentType{}, err
	}
	now := time.Now().UTC()
	item.CreatedAt, item.UpdatedAt = now, now
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.types[item.Code]; exists {
		return ContentType{}, errors.New("content type code already exists")
	}
	s.types[item.Code] = item
	return item, nil
}

func (s *MemoryStore) UpdateContentType(_ context.Context, code Type, input UpdateContentTypeInput) (ContentType, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.types[code]
	if !ok {
		return ContentType{}, ErrContentTypeNotFound
	}
	item, err := NormalizeContentType(code, input.Name, input.Icon, input.Accent, input.Description)
	if err != nil {
		return ContentType{}, err
	}
	item.CreatedAt, item.UpdatedAt = current.CreatedAt, time.Now().UTC()
	s.types[code] = item
	return item, nil
}

func (s *MemoryStore) DeleteContentType(_ context.Context, code Type) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.types[code]; !ok {
		return ErrContentTypeNotFound
	}
	for _, item := range s.projects {
		if item.Type == code {
			return ErrContentTypeInUse
		}
	}
	delete(s.types, code)
	return nil
}

func (s *MemoryStore) seedContentTypes() {
	now := time.Now().UTC()
	items := []ContentType{{Code: TypeNovel, Name: "小说", Icon: "文", Accent: "amber", Description: "小说与长篇叙事"}, {Code: TypeMovieCommentary, Name: "电影解说", Icon: "映", Accent: "blue", Description: "电影、剧集解说稿"}, {Code: TypeTechnicalDocument, Name: "技术文档", Icon: "术", Accent: "green", Description: "产品与技术资料"}}
	for index := range items {
		items[index].CreatedAt = now.Add(time.Duration(index) * time.Microsecond)
		items[index].UpdatedAt = items[index].CreatedAt
		s.types[items[index].Code] = items[index]
	}
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
