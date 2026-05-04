package intenttree

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	platformstate "github.com/AmazingCYJ/AgentRAG/internal/platform/state"
	"github.com/gogf/gf/v2/util/guid"
)

var (
	ErrNodeNotFound       = errors.New("意图节点不存在")
	ErrIntentCodeRequired = errors.New("意图标识不能为空")
	ErrIntentNameRequired = errors.New("节点名称不能为空")
	ErrIntentCodeExists   = errors.New("意图标识已存在")
	ErrParentNotFound     = errors.New("父节点不存在")
)

// Node 表示内部意图节点。
type Node struct {
	ID                  string
	KBID                string
	IntentCode          string
	Name                string
	Level               int
	ParentCode          string
	Description         string
	Examples            []string
	CollectionName      string
	MCPToolID           string
	TopK                *int
	Kind                int
	SortOrder           int
	Enabled             int
	PromptSnippet       string
	PromptTemplate      string
	ParamPromptTemplate string
	CreateTime          time.Time
	UpdateTime          time.Time
}

// TreeNode 定义前端树接口返回结构。
type TreeNode struct {
	ID                  string     `json:"id"`
	IntentCode          string     `json:"intentCode"`
	Name                string     `json:"name"`
	Level               int        `json:"level"`
	ParentCode          string     `json:"parentCode,omitempty"`
	Description         string     `json:"description,omitempty"`
	Examples            string     `json:"examples,omitempty"`
	CollectionName      string     `json:"collectionName,omitempty"`
	MCPToolID           string     `json:"mcpToolId,omitempty"`
	TopK                *int       `json:"topK,omitempty"`
	Kind                int        `json:"kind"`
	SortOrder           int        `json:"sortOrder"`
	Enabled             int        `json:"enabled"`
	PromptSnippet       string     `json:"promptSnippet,omitempty"`
	PromptTemplate      string     `json:"promptTemplate,omitempty"`
	ParamPromptTemplate string     `json:"paramPromptTemplate,omitempty"`
	Children            []TreeNode `json:"children,omitempty"`
}

// CreateRequest 定义新增节点输入。
type CreateRequest struct {
	KBID                string
	IntentCode          string
	Name                string
	Level               int
	ParentCode          string
	Description         string
	Examples            []string
	MCPToolID           string
	TopK                *int
	Kind                int
	SortOrder           int
	Enabled             int
	PromptSnippet       string
	PromptTemplate      string
	ParamPromptTemplate string
}

// UpdateRequest 定义更新节点输入。
type UpdateRequest struct {
	Name                string
	Level               int
	ParentCode          string
	Description         string
	Examples            []string
	CollectionName      string
	MCPToolID           string
	TopK                *int
	Kind                int
	SortOrder           int
	Enabled             int
	PromptSnippet       string
	PromptTemplate      string
	ParamPromptTemplate string
}

// RouteHint 定义意图树返回的最小路由结果。
type RouteHint struct {
	Kind           int
	ToolID         string
	CollectionName string
	Score          int
}

// Repository 定义意图节点持久化仓储。
type Repository interface {
	Bootstrap() error
	LoadNodes() ([]Node, error)
	SaveNodes(nodes []Node) error
}

type fileStoreRepository struct {
	store *platformstate.FileStore
}

func (r *fileStoreRepository) Bootstrap() error { return nil }

func (r *fileStoreRepository) LoadNodes() ([]Node, error) {
	if r == nil || r.store == nil {
		return nil, nil
	}
	snapshot, err := r.store.Load()
	if err != nil {
		return nil, err
	}
	nodes := make([]Node, 0, len(snapshot.IntentNodes))
	for _, record := range snapshot.IntentNodes {
		nodes = append(nodes, Node{
			ID:                  record.ID,
			KBID:                record.KBID,
			IntentCode:          record.IntentCode,
			Name:                record.Name,
			Level:               record.Level,
			ParentCode:          record.ParentCode,
			Description:         record.Description,
			Examples:            normalizeExamples(record.Examples),
			CollectionName:      record.CollectionName,
			MCPToolID:           record.MCPToolID,
			TopK:                cloneIntPointer(record.TopK),
			Kind:                record.Kind,
			SortOrder:           record.SortOrder,
			Enabled:             record.Enabled,
			PromptSnippet:       record.PromptSnippet,
			PromptTemplate:      record.PromptTemplate,
			ParamPromptTemplate: record.ParamPromptTemplate,
			CreateTime:          record.CreateTime,
			UpdateTime:          record.UpdateTime,
		})
	}
	return nodes, nil
}

func (r *fileStoreRepository) SaveNodes(nodes []Node) error {
	if r == nil || r.store == nil {
		return nil
	}
	records := make([]platformstate.IntentNodeRecord, 0, len(nodes))
	for _, node := range nodes {
		records = append(records, platformstate.IntentNodeRecord{
			ID:                  node.ID,
			KBID:                node.KBID,
			IntentCode:          node.IntentCode,
			Name:                node.Name,
			Level:               node.Level,
			ParentCode:          node.ParentCode,
			Description:         node.Description,
			Examples:            normalizeExamples(node.Examples),
			CollectionName:      node.CollectionName,
			MCPToolID:           node.MCPToolID,
			TopK:                cloneIntPointer(node.TopK),
			Kind:                node.Kind,
			SortOrder:           node.SortOrder,
			Enabled:             node.Enabled,
			PromptSnippet:       node.PromptSnippet,
			PromptTemplate:      node.PromptTemplate,
			ParamPromptTemplate: node.ParamPromptTemplate,
			CreateTime:          node.CreateTime,
			UpdateTime:          node.UpdateTime,
		})
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreateTime.Before(records[j].CreateTime)
	})
	return r.store.Update(func(snapshot *platformstate.Snapshot) {
		snapshot.IntentNodes = records
	})
}

// Service 提供意图树内存管理能力。
type Service struct {
	mu    sync.RWMutex
	nodes map[string]Node
	newID func() string
	repo  Repository
}

// NewService 创建意图树服务（使用 JSON 文件持久化）。
func NewService(store *platformstate.FileStore) *Service {
	var repo Repository
	if store != nil {
		repo = &fileStoreRepository{store: store}
	}
	return NewServiceWithRepository(repo)
}

// NewServiceWithRepository 创建基于指定仓储的意图树服务。
func NewServiceWithRepository(repo Repository) *Service {
	svc := &Service{
		nodes: make(map[string]Node),
		newID: func() string { return strings.ReplaceAll(guid.S(), "-", "") },
		repo:  repo,
	}
	if repo != nil {
		if nodes, err := repo.LoadNodes(); err == nil {
			for _, node := range nodes {
				svc.nodes[node.ID] = node
			}
		}
	}
	return svc
}

// GetFullTree 返回完整意图树。
func (s *Service) GetFullTree() []TreeNode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return buildTree(s.nodes)
}

// MatchQuestion 根据问题匹配最可能的意图节点。
func (s *Service) MatchQuestion(question string) RouteHint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := strings.TrimSpace(question)
	if query == "" {
		return RouteHint{}
	}
	queryTokens := buildIntentTokens(query)
	best := RouteHint{}
	for _, node := range s.nodes {
		if node.Enabled == 0 {
			continue
		}
		target := strings.Join([]string{
			node.Name,
			node.Description,
			strings.Join(node.Examples, " "),
			node.PromptSnippet,
			node.PromptTemplate,
			node.ParamPromptTemplate,
		}, " ")
		score := intentOverlapScore(queryTokens, buildIntentTokens(target))
		if strings.Contains(target, query) {
			score += 4
		}
		if score > best.Score {
			best = RouteHint{
				Kind:           node.Kind,
				ToolID:         node.MCPToolID,
				CollectionName: node.CollectionName,
				Score:          score,
			}
		}
	}
	return best
}

// CreateNode 创建新的意图节点。
func (s *Service) CreateNode(req CreateRequest) (string, error) {
	intentCode := strings.TrimSpace(req.IntentCode)
	if intentCode == "" {
		return "", ErrIntentCodeRequired
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return "", ErrIntentNameRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasIntentCodeLocked(intentCode, "") {
		return "", ErrIntentCodeExists
	}
	parentCode := strings.TrimSpace(req.ParentCode)
	if parentCode != "" && !s.hasIntentCodeValueLocked(parentCode) {
		return "", ErrParentNotFound
	}

	id := s.newID()
	now := time.Now()
	node := Node{
		ID:                  id,
		KBID:                strings.TrimSpace(req.KBID),
		IntentCode:          intentCode,
		Name:                name,
		Level:               req.Level,
		ParentCode:          parentCode,
		Description:         strings.TrimSpace(req.Description),
		Examples:            normalizeExamples(req.Examples),
		CollectionName:      inferCollectionName(req.KBID),
		MCPToolID:           strings.TrimSpace(req.MCPToolID),
		TopK:                cloneIntPointer(req.TopK),
		Kind:                req.Kind,
		SortOrder:           req.SortOrder,
		Enabled:             normalizeEnabled(req.Enabled),
		PromptSnippet:       strings.TrimSpace(req.PromptSnippet),
		PromptTemplate:      strings.TrimSpace(req.PromptTemplate),
		ParamPromptTemplate: strings.TrimSpace(req.ParamPromptTemplate),
		CreateTime:          now,
		UpdateTime:          now,
	}
	s.nodes[id] = node
	if err := s.persistLocked(); err != nil {
		return "", err
	}
	return id, nil
}

// UpdateNode 更新指定意图节点。
func (s *Service) UpdateNode(id string, req UpdateRequest) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ErrIntentNameRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	node, ok := s.nodes[id]
	if !ok {
		return ErrNodeNotFound
	}
	parentCode := strings.TrimSpace(req.ParentCode)
	if parentCode != "" && parentCode != node.IntentCode && !s.hasIntentCodeValueLocked(parentCode) {
		return ErrParentNotFound
	}

	node.Name = name
	node.Level = req.Level
	node.ParentCode = parentCode
	node.Description = strings.TrimSpace(req.Description)
	node.Examples = normalizeExamples(req.Examples)
	node.CollectionName = strings.TrimSpace(req.CollectionName)
	node.MCPToolID = strings.TrimSpace(req.MCPToolID)
	node.TopK = cloneIntPointer(req.TopK)
	node.Kind = req.Kind
	node.SortOrder = req.SortOrder
	node.Enabled = normalizeEnabled(req.Enabled)
	node.PromptSnippet = strings.TrimSpace(req.PromptSnippet)
	node.PromptTemplate = strings.TrimSpace(req.PromptTemplate)
	node.ParamPromptTemplate = strings.TrimSpace(req.ParamPromptTemplate)
	node.UpdateTime = time.Now()
	s.nodes[id] = node
	return s.persistLocked()
}

// DeleteNode 删除指定节点及其子节点。
func (s *Service) DeleteNode(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.nodes[id]; !ok {
		return ErrNodeNotFound
	}
	s.deleteNodeAndChildrenLocked(id)
	return s.persistLocked()
}

// BatchEnableNodes 批量启用节点。
func (s *Service) BatchEnableNodes(ids []string) {
	s.batchUpdateEnabled(ids, 1)
}

// BatchDisableNodes 批量停用节点。
func (s *Service) BatchDisableNodes(ids []string) {
	s.batchUpdateEnabled(ids, 0)
}

// BatchDeleteNodes 批量删除节点。
func (s *Service) BatchDeleteNodes(ids []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		if _, ok := s.nodes[id]; ok {
			s.deleteNodeAndChildrenLocked(id)
		}
	}
	_ = s.persistLocked()
}

func (s *Service) batchUpdateEnabled(ids []string, enabled int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		node, ok := s.nodes[id]
		if !ok {
			continue
		}
		node.Enabled = enabled
		node.UpdateTime = time.Now()
		s.nodes[id] = node
	}
	_ = s.persistLocked()
}

func (s *Service) deleteNodeAndChildrenLocked(id string) {
	intentCode := s.nodes[id].IntentCode
	delete(s.nodes, id)
	for childID, node := range s.nodes {
		if node.ParentCode == intentCode {
			s.deleteNodeAndChildrenLocked(childID)
		}
	}
}

func (s *Service) hasIntentCodeLocked(intentCode string, excludeID string) bool {
	for id, node := range s.nodes {
		if id == excludeID {
			continue
		}
		if node.IntentCode == intentCode {
			return true
		}
	}
	return false
}

func (s *Service) hasIntentCodeValueLocked(intentCode string) bool {
	for _, node := range s.nodes {
		if node.IntentCode == intentCode {
			return true
		}
	}
	return false
}

func buildTree(nodes map[string]Node) []TreeNode {
	if len(nodes) == 0 {
		return []TreeNode{}
	}

	childrenByParent := make(map[string][]Node)
	for _, node := range nodes {
		childrenByParent[node.ParentCode] = append(childrenByParent[node.ParentCode], node)
	}
	for key := range childrenByParent {
		sort.Slice(childrenByParent[key], func(i, j int) bool {
			if childrenByParent[key][i].SortOrder == childrenByParent[key][j].SortOrder {
				return childrenByParent[key][i].CreateTime.Before(childrenByParent[key][j].CreateTime)
			}
			return childrenByParent[key][i].SortOrder < childrenByParent[key][j].SortOrder
		})
	}

	var walk func(parentCode string) []TreeNode
	walk = func(parentCode string) []TreeNode {
		children := childrenByParent[parentCode]
		result := make([]TreeNode, 0, len(children))
		for _, child := range children {
			result = append(result, TreeNode{
				ID:                  child.ID,
				IntentCode:          child.IntentCode,
				Name:                child.Name,
				Level:               child.Level,
				ParentCode:          child.ParentCode,
				Description:         child.Description,
				Examples:            encodeExamples(child.Examples),
				CollectionName:      child.CollectionName,
				MCPToolID:           child.MCPToolID,
				TopK:                cloneIntPointer(child.TopK),
				Kind:                child.Kind,
				SortOrder:           child.SortOrder,
				Enabled:             child.Enabled,
				PromptSnippet:       child.PromptSnippet,
				PromptTemplate:      child.PromptTemplate,
				ParamPromptTemplate: child.ParamPromptTemplate,
				Children:            walk(child.IntentCode),
			})
		}
		return result
	}

	return walk("")
}

func encodeExamples(examples []string) string {
	items := normalizeExamples(examples)
	if len(items) == 0 {
		return ""
	}
	content, err := json.Marshal(items)
	if err != nil {
		return strings.Join(items, "\n")
	}
	return string(content)
}

func normalizeExamples(examples []string) []string {
	if len(examples) == 0 {
		return nil
	}
	result := make([]string, 0, len(examples))
	for _, example := range examples {
		value := strings.TrimSpace(example)
		if value != "" {
			result = append(result, value)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func inferCollectionName(kbID string) string {
	value := strings.TrimSpace(kbID)
	if value == "" {
		return ""
	}
	return value
}

func normalizeEnabled(enabled int) int {
	if enabled == 0 {
		return 0
	}
	return 1
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}

func buildIntentTokens(text string) []string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return nil
	}
	fields := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t' || r == '，' || r == '。' || r == '、' || r == ',' || r == '.' || r == ':' || r == '：'
	})
	result := make([]string, 0, len(fields)+16)
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		result = append(result, field)
		appendIntentNGrams(&result, []rune(field), 2)
		appendIntentNGrams(&result, []rune(field), 3)
	}
	if len(result) == 0 {
		result = append(result, normalized)
		appendIntentNGrams(&result, []rune(normalized), 2)
		appendIntentNGrams(&result, []rune(normalized), 3)
	}
	return result
}

func intentOverlapScore(a, b []string) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	bag := make(map[string]struct{}, len(b))
	for _, item := range b {
		bag[item] = struct{}{}
	}
	score := 0
	for _, item := range a {
		if _, ok := bag[item]; ok {
			score++
		}
	}
	return score
}

func appendIntentNGrams(target *[]string, runes []rune, size int) {
	if len(runes) < size || size <= 0 {
		return
	}
	for i := 0; i <= len(runes)-size; i++ {
		*target = append(*target, string(runes[i:i+size]))
	}
}

func (s *Service) persistLocked() error {
	if s.repo == nil {
		return nil
	}
	nodes := make([]Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}
	return s.repo.SaveNodes(nodes)
}
