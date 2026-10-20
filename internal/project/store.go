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

	"novelstudio/internal/auth"
)

var ErrNotFound = errors.New("project not found")
var ErrContentTypeNotFound = errors.New("content type not found")
var ErrContentTypeInUse = errors.New("content type is used by projects")

type Store interface {
	List(context.Context) ([]Project, error)
	Get(context.Context, string) (Project, error)
	Create(context.Context, CreateInput) (Project, error)
	Update(context.Context, string, UpdateInput) (Project, error)
	Delete(context.Context, string) error
	Tree(context.Context, string) ([]ContentNode, error)
	CreateNode(context.Context, string, CreateNodeInput) (ContentNode, error)
	UpdateNode(context.Context, string, UpdateNodeInput) (ContentNode, error)
	DeleteNode(context.Context, string) error
	ProjectIDForNode(context.Context, string) (string, error)
	ClaimUnowned(context.Context, string) error
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
func (s *MemoryStore) ProjectIDForNode(_ context.Context, id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for projectID, nodes := range s.nodes {
		for _, node := range nodes {
			if node.ID == id {
				return projectID, nil
			}
		}
	}
	return "", ErrNotFound
}
func (s *MemoryStore) ClaimUnowned(_ context.Context, ownerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, item := range s.projects {
		if item.OwnerID == "" {
			item.OwnerID = ownerID
			s.projects[id] = item
		}
	}
	return nil
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

func (s *MemoryStore) List(ctx context.Context) ([]Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Project, 0, len(s.projects))
	user, scoped := auth.UserFromContext(ctx)
	for _, item := range s.projects {
		if scoped && item.OwnerID != user.ID {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return items, nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.projects[id]
	user, scoped := auth.UserFromContext(ctx)
	if !ok || scoped && item.OwnerID != user.ID {
		return Project{}, ErrNotFound
	}
	return item, nil
}

func (s *MemoryStore) Create(ctx context.Context, input CreateInput) (Project, error) {
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
	if user, ok := auth.UserFromContext(ctx); ok {
		item.OwnerID = user.ID
	}
	s.mu.Lock()
	s.projects[id] = item
	s.nodes[id] = defaultTree(id, input.Type)
	s.mu.Unlock()
	return item, nil
}

func (s *MemoryStore) Update(_ context.Context, id string, input UpdateInput) (Project, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return Project{}, errors.New("name is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.projects[id]
	if !ok {
		return Project{}, ErrNotFound
	}
	item.Name = input.Name
	item.Description = strings.TrimSpace(input.Description)
	item.UpdatedAt = time.Now().UTC()
	s.projects[id] = item
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

func NormalizeContentType(code Type, name, icon, accent, description, roles, prompt string) (ContentType, error) {
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
	if len(prompt) > 12000 {
		return ContentType{}, errors.New("prompt is too long")
	}
	if len(roles) > 6000 {
		return ContentType{}, errors.New("roles is too long")
	}
	if icon == "" {
		icon = string([]rune(name)[0])
	}
	if accent == "" {
		accent = "amber"
	}
	return ContentType{Code: code, Name: name, Icon: icon, Accent: accent, Description: strings.TrimSpace(description), Roles: strings.TrimSpace(roles), Prompt: strings.TrimSpace(prompt)}, nil
}

func (s *MemoryStore) CreateContentType(_ context.Context, input CreateContentTypeInput) (ContentType, error) {
	item, err := NormalizeContentType(input.Code, input.Name, input.Icon, input.Accent, input.Description, input.Roles, input.Prompt)
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
	item, err := NormalizeContentType(code, input.Name, input.Icon, input.Accent, input.Description, input.Roles, input.Prompt)
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
	items := []ContentType{
		{Code: TypeNovel, Name: "小说", Icon: "文", Accent: "amber", Description: "小说与长篇叙事", Roles: "主角：明确欲望、缺陷、外部目标与内在需求，经历可验证的人物弧光。\n关键配角：每人拥有独立立场、行动能力和与主角的关系变化。\n对立角色：目标合理且能持续施压，不作为纯粹工具人。\n关系要求：记录称谓、利益、情感、秘密与冲突的变化，避免人物失忆、性格突变和能力越界。", Prompt: "你正在创作可持续连载的小说。先确认题材、时代、世界规则、叙事视角、目标读者和篇幅；围绕人物欲望与阻力构建主线、支线和阶段性转折。每个场景都应具备目标、冲突、变化与后果，使用动作、环境、对话和感官细节呈现信息，避免用总结代替戏剧过程。跨章节维护时间线、地点、人物状态、物品、承诺、伏笔和因果链；伏笔必须有铺垫与回收。语言符合人物身份与时代背景，控制视角漂移、重复解释、空泛抒情和提纲腔。资料不足时保留不确定性，不擅自创造关键事实。输出应结构清晰、节奏有张弛，并与既有正文无缝衔接。"},
		{Code: TypeMovieCommentary, Name: "电影解说", Icon: "映", Accent: "blue", Description: "电影解说与影评口播稿", Roles: "创作人格：你是一名从业多年的专业电影解说人和资深影迷，熟悉类型片结构、视听语言、人物弧光、悬念设计与口播节奏。\n核心专长：能迅速抓住影片最有传播力的冲突、反转和情绪爆点；擅长把复杂剧情讲得清楚、紧凑、有画面感，并从人物选择中提炼主题。\n表达风格：聪明、幽默、自然，可使用克制的吐槽、反差和包袱，但幽默必须服务剧情，不低俗、不抢戏，也不破坏悲剧或严肃场面的情绪。\n解说立场：像一位懂电影又会讲故事的朋友，观点鲜明但不居高临下；明确区分影片事实、合理分析和个人评论。\n人物处理：人物首次出场时交代身份和关系，持续追踪其目标、关键选择、关系变化和人物弧光，称谓始终一致。\n事实边界：只依据提供的影片资料，不虚构镜头、台词、幕后信息、人物动机或主创观点。", Prompt: "你正在撰写适合配音和剪辑的电影解说稿。以影片可核验内容为基础，设计开场钩子、背景建立、冲突升级、关键转折、高潮、结局与主题回收。解说必须对应明确的剧情节点，句子简洁、口语自然、节奏紧凑，长短句交替并为画面留出空间。幽默应来自剧情反差、人物行为或精准观察，密度适中，不强行造梗。区分影片事实、合理概括和作者评论；不得虚构镜头、台词、人物动机、主创信息或幕后资料。重要人物首次出现时交代身份和关系，后续称谓保持一致。评论应由剧情证据支撑，避免流水账、过度煽情、重复复述和泛化价值判断。输出完整 Markdown 成稿，不写制作说明或思维过程。"},
		{Code: TypeTVCommentary, Name: "电视剧解说", Icon: "剧", Accent: "violet", Description: "电视剧、网剧与连续剧分集解说稿", Roles: "解说者：保持统一口吻，负责前情承接、当集推进和悬念收束。\n常驻人物：维护身份、阵营、关系、目标、已知信息与成长变化。\n单元人物：说明其在当集的功能、诉求和与主线的连接。\n对立角色：持续追踪策略、资源和威胁升级，避免跨集行为失真。", Prompt: "你正在撰写连续剧分集解说稿。严格区分季、集、时间线和剧情单元，先承接必要前情，再聚焦当集核心冲突、人物选择、信息揭示和结尾悬念。维护人物关系、阵营、伤病、物品、秘密、已知信息和未回收伏笔，禁止把后续剧情提前写入当前集。语言应适合口播和连续更新，节奏清晰，转场自然，每一段都推动剧情或深化人物。只依据提供的剧集资料，不虚构台词、镜头、集数、幕后信息和角色动机；资料冲突时显式采用更可靠来源。避免逐镜头流水账、重复前情、空洞评价和与画面脱节的旁白。"},
		{Code: TypeTechnicalDocument, Name: "技术文档", Icon: "术", Accent: "green", Description: "产品与技术资料", Roles: "目标读者：明确其技术水平、权限、前置知识和使用目标。\n操作者：按步骤执行配置、调用、验证与故障排查。\n系统或服务：明确职责、输入输出、状态、依赖和失败模式。\n审核者：检查安全性、准确性、可复现性、兼容性和遗漏项。", Prompt: "你正在编写可执行、可验证、可维护的技术文档。先明确受众、目标、适用版本、环境、前置条件和非目标，再组织概念、架构、步骤、接口、示例、验证、错误处理、限制和回滚方案。所有命令、参数、字段、返回值、版本和测试结论必须来自可信资料；不确定内容使用明确标记，不得臆造。步骤应给出操作、预期结果和失败处理，代码示例保持最小完整且内部一致。术语、命名和格式全文统一，区分事实、建议与推断，并提示安全风险和破坏性操作。输出结构化 Markdown，优先使用短段落、列表、表格和可复制代码块，避免宣传语言、循环解释和无法验证的保证。"},
	}
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
	if projectType == TypeTVCommentary {
		rootType, childType = "SEASON", "EPISODE"
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
