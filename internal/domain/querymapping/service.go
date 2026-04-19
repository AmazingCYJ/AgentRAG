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

// Service 提供映射规则内存管理能力。
type Service struct {
	mu    sync.RWMutex
	items map[string]Item

	now   func() time.Time
	newID func() string
	store *platformstate.FileStore
}

// NewService 创建映射规则服务。
func NewService(store *platformstate.FileStore) *Service {
	service := &Service{
		items: make(map[string]Item),
		now:   time.Now,
		newID: func() string { return strings.ReplaceAll(guid.S(), "-", "") },
		store: store,
	}
	if snapshot, err := service.loadSnapshot(); err == nil {
		for _, record := range snapshot.QueryMappings {
			service.items[record.ID] = Item{
				ID:         record.ID,
				SourceTerm: record.SourceTerm,
				TargetTerm: record.TargetTerm,
				MatchType:  record.MatchType,
				Priority:   record.Priority,
				Enabled:    record.Enabled,
				Remark:     record.Remark,
				CreateTime: record.CreateTime,
				UpdateTime: record.UpdateTime,
			}
		}
	}
	return service
}

func (s *Service) loadSnapshot() (platformstate.Snapshot, error) {
	if s.store == nil {
		return platformstate.Snapshot{}, nil
	}
	return s.store.Load()
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
	if s.store == nil {
		return nil
	}
	records := make([]platformstate.QueryMappingRecord, 0, len(s.items))
	for _, item := range s.items {
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
	sort.Slice(records, func(i, j int) bool {
		if records[i].Priority == records[j].Priority {
			return records[i].CreateTime.Before(records[j].CreateTime)
		}
		return records[i].Priority < records[j].Priority
	})
	return s.store.Update(func(snapshot *platformstate.Snapshot) {
		snapshot.QueryMappings = records
	})
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
