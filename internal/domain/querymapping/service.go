package querymapping

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	platformstate "github.com/AmazingCYJ/AgentRAG/internal/platform/state"
	"github.com/gogf/gf/v2/util/guid"
)

var (
	// ErrMappingNotFound 表示映射规则不存在。
	ErrMappingNotFound = errors.New("映射规则不存在")
	// ErrSourceTermRequired 表示原始词不能为空。
	ErrSourceTermRequired = errors.New("原始词不能为空")
	// ErrTargetTermRequired 表示目标词不能为空。
	ErrTargetTermRequired = errors.New("目标词不能为空")
)

// Item 表示关键词映射规则。
type Item struct {
	ID         string    `json:"id"`
	SourceTerm string    `json:"sourceTerm"`
	TargetTerm string    `json:"targetTerm"`
	MatchType  int       `json:"matchType"`
	Priority   int       `json:"priority"`
	Enabled    bool      `json:"enabled"`
	Remark     string    `json:"remark,omitempty"`
	CreateTime time.Time `json:"createTime,omitempty"`
	UpdateTime time.Time `json:"updateTime,omitempty"`
}

// PageResult 定义分页结果。
type PageResult struct {
	Records []Item `json:"records"`
	Total   int    `json:"total"`
	Size    int    `json:"size"`
	Current int    `json:"current"`
	Pages   int    `json:"pages"`
}

// SaveRequest 定义创建和更新输入。
type SaveRequest struct {
	SourceTerm string
	TargetTerm string
	MatchType  int
	Priority   int
	Enabled    bool
	Remark     string
}

// Repository 定义关键词映射持久化仓储。
type Repository interface {
	LoadQueryMappings() ([]Item, error)
	SaveQueryMappings(items []Item) error
}

type fileStoreRepository struct {
	store *platformstate.FileStore
}

func (r *fileStoreRepository) LoadQueryMappings() ([]Item, error) {
	if r == nil || r.store == nil {
		return nil, nil
	}
	snapshot, err := r.store.Load()
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(snapshot.QueryMappings))
	for _, record := range snapshot.QueryMappings {
		items = append(items, Item{
			ID:         record.ID,
			SourceTerm: record.SourceTerm,
			TargetTerm: record.TargetTerm,
			MatchType:  record.MatchType,
			Priority:   record.Priority,
			Enabled:    record.Enabled,
			Remark:     record.Remark,
			CreateTime: record.CreateTime,
			UpdateTime: record.UpdateTime,
		})
	}
	return items, nil
}

func (r *fileStoreRepository) SaveQueryMappings(items []Item) error {
	if r == nil || r.store == nil {
		return nil
	}
	records := make([]platformstate.QueryMappingRecord, 0, len(items))
	for _, item := range items {
		records = append(records, platformstate.QueryMappingRecord{
			ID:         item.ID,
			SourceTerm: item.SourceTerm,
			TargetTerm: item.TargetTerm,
			MatchType:  item.MatchType,
			Priority:   item.Priority,
			Enabled:    item.Enabled,
			Remark:     item.Remark,
			CreateTime: item.CreateTime,
			UpdateTime: item.UpdateTime,
		})
	}
	return r.store.Update(func(snapshot *platformstate.Snapshot) {
		snapshot.QueryMappings = records
	})
}

// Service 提供映射规则内存管理能力。
type Service struct {
	mu    sync.RWMutex
	items map[string]Item

	now   func() time.Time
	newID func() string
	repo  Repository
}

// NewService 创建映射规则服务。
func NewService(store *platformstate.FileStore) *Service {
	var repo Repository
	if store != nil {
		repo = &fileStoreRepository{store: store}
	}
	return NewServiceWithRepository(repo)
}

// NewServiceWithRepository 创建基于指定仓储的映射规则服务。
func NewServiceWithRepository(repo Repository) *Service {
	service := &Service{
		items: make(map[string]Item),
		now:   time.Now,
		newID: func() string { return strings.ReplaceAll(guid.S(), "-", "") },
		repo:  repo,
	}
	if items, err := service.loadItems(); err == nil {
		for _, item := range items {
			service.items[item.ID] = item
		}
	}
	return service
}

func (s *Service) loadItems() ([]Item, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.LoadQueryMappings()
}

// Page 返回分页结果。
func (s *Service) Page(current, size int, keyword string) PageResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if current <= 0 {
		current = 1
	}
	if size <= 0 {
		size = 10
	}

	records := make([]Item, 0, len(s.items))
	for _, item := range s.items {
		if matchKeyword(item, keyword) {
			records = append(records, item)
		}
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Priority == records[j].Priority {
			return records[i].CreateTime.Before(records[j].CreateTime)
		}
		return records[i].Priority < records[j].Priority
	})

	total := len(records)
	pages := 1
	if total > 0 {
		pages = (total + size - 1) / size
	}
	start := (current - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}

	result := make([]Item, end-start)
	copy(result, records[start:end])
	return PageResult{
		Records: result,
		Total:   total,
		Size:    size,
		Current: current,
		Pages:   pages,
	}
}

// GetByID 返回指定映射规则。
func (s *Service) GetByID(id string) (Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[id]
	if !ok {
		return Item{}, ErrMappingNotFound
	}
	return item, nil
}

// Create 新增映射规则。
func (s *Service) Create(req SaveRequest) (string, error) {
	sourceTerm := strings.TrimSpace(req.SourceTerm)
	if sourceTerm == "" {
		return "", ErrSourceTermRequired
	}
	targetTerm := strings.TrimSpace(req.TargetTerm)
	if targetTerm == "" {
		return "", ErrTargetTermRequired
	}

	now := s.now()
	item := Item{
		ID:         s.newID(),
		SourceTerm: sourceTerm,
		TargetTerm: targetTerm,
		MatchType:  normalizeMatchType(req.MatchType),
		Priority:   req.Priority,
		Enabled:    req.Enabled,
		Remark:     strings.TrimSpace(req.Remark),
		CreateTime: now,
		UpdateTime: now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[item.ID] = item
	if err := s.persistLocked(); err != nil {
		return "", err
	}
	return item.ID, nil
}

// Update 修改指定映射规则。
func (s *Service) Update(id string, req SaveRequest) error {
	sourceTerm := strings.TrimSpace(req.SourceTerm)
	if sourceTerm == "" {
		return ErrSourceTermRequired
	}
	targetTerm := strings.TrimSpace(req.TargetTerm)
	if targetTerm == "" {
		return ErrTargetTermRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[id]
	if !ok {
		return ErrMappingNotFound
	}
	item.SourceTerm = sourceTerm
	item.TargetTerm = targetTerm
	item.MatchType = normalizeMatchType(req.MatchType)
	item.Priority = req.Priority
	item.Enabled = req.Enabled
	item.Remark = strings.TrimSpace(req.Remark)
	item.UpdateTime = s.now()
	s.items[id] = item
	if err := s.persistLocked(); err != nil {
		return err
	}
	return nil
}

// Delete 删除指定映射规则。
func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return ErrMappingNotFound
	}
	delete(s.items, id)
	if err := s.persistLocked(); err != nil {
		return err
	}
	return nil
}

func (s *Service) persistLocked() error {
	if s.repo == nil {
		return nil
	}
	items := make([]Item, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority == items[j].Priority {
			return items[i].CreateTime.Before(items[j].CreateTime)
		}
		return items[i].Priority < items[j].Priority
	})
	return s.repo.SaveQueryMappings(items)
}

func normalizeMatchType(matchType int) int {
	if matchType >= 1 && matchType <= 4 {
		return matchType
	}
	return 1
}

func matchKeyword(item Item, keyword string) bool {
	needle := strings.ToLower(strings.TrimSpace(keyword))
	if needle == "" {
		return true
	}
	haystack := strings.ToLower(strings.Join([]string{
		item.SourceTerm,
		item.TargetTerm,
		item.Remark,
	}, " "))
	return strings.Contains(haystack, needle)
}
