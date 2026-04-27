package samplequestion

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
	// ErrQuestionNotFound 表示示例问题不存在。
	ErrQuestionNotFound = errors.New("示例问题不存在")
	// ErrQuestionRequired 表示示例问题内容不能为空。
	ErrQuestionRequired = errors.New("示例问题内容不能为空")
)

const welcomeQuestionLimit = 3

// Item 表示示例问题实体。
type Item struct {
	ID          string    `json:"id"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	Question    string    `json:"question"`
	CreateTime  time.Time `json:"createTime,omitempty"`
	UpdateTime  time.Time `json:"updateTime,omitempty"`
}

// PageResult 定义分页响应结构。
type PageResult struct {
	Records []Item `json:"records"`
	Total   int    `json:"total"`
	Size    int    `json:"size"`
	Current int    `json:"current"`
	Pages   int    `json:"pages"`
}

// SaveRequest 定义新增和更新输入。
type SaveRequest struct {
	Title       string
	Description string
	Question    string
}

// Repository 定义示例问题持久化仓储。
type Repository interface {
	LoadSampleQuestions() ([]Item, error)
	SaveSampleQuestions(items []Item) error
}

type fileStoreRepository struct {
	store *platformstate.FileStore
}

func (r *fileStoreRepository) LoadSampleQuestions() ([]Item, error) {
	if r == nil || r.store == nil {
		return nil, nil
	}
	snapshot, err := r.store.Load()
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(snapshot.SampleQuestions))
	for _, record := range snapshot.SampleQuestions {
		items = append(items, Item{
			ID:          record.ID,
			Title:       record.Title,
			Description: record.Description,
			Question:    record.Question,
			CreateTime:  record.CreateTime,
			UpdateTime:  record.UpdateTime,
		})
	}
	return items, nil
}

func (r *fileStoreRepository) SaveSampleQuestions(items []Item) error {
	if r == nil || r.store == nil {
		return nil
	}
	records := make([]platformstate.SampleQuestionRecord, 0, len(items))
	for _, item := range items {
		records = append(records, platformstate.SampleQuestionRecord{
			ID:          item.ID,
			Title:       item.Title,
			Description: item.Description,
			Question:    item.Question,
			CreateTime:  item.CreateTime,
			UpdateTime:  item.UpdateTime,
		})
	}
	return r.store.Update(func(snapshot *platformstate.Snapshot) {
		snapshot.SampleQuestions = records
	})
}

// Service 提供示例问题内存存储能力。
type Service struct {
	mu    sync.RWMutex
	items []Item

	now   func() time.Time
	newID func() string
	repo  Repository
}

// NewService 创建示例问题服务，并注入默认欢迎页数据。
func NewService(store *platformstate.FileStore) *Service {
	var repo Repository
	if store != nil {
		repo = &fileStoreRepository{store: store}
	}
	return NewServiceWithRepository(repo)
}

// NewServiceWithRepository 创建基于指定仓储的示例问题服务。
func NewServiceWithRepository(repo Repository) *Service {
	now := time.Now()
	service := &Service{
		items: []Item{
			{
				ID:          compactID(guid.S()),
				Title:       "内容总结",
				Description: "提炼 3-5 条关键信息与行动点",
				Question:    "请帮我总结以下内容，并列出3-5条要点：",
				CreateTime:  now,
				UpdateTime:  now,
			},
			{
				ID:          compactID(guid.S()),
				Title:       "任务拆解",
				Description: "把目标拆成可执行步骤与优先级",
				Question:    "请把下面需求拆解为步骤，并给出优先级和里程碑：",
				CreateTime:  now,
				UpdateTime:  now,
			},
			{
				ID:          compactID(guid.S()),
				Title:       "灵感扩展",
				Description: "给出多个方案并比较优缺点",
				Question:    "围绕以下主题给出5-8个方案，并注明优缺点：",
				CreateTime:  now,
				UpdateTime:  now,
			},
		},
		now: time.Now,
		newID: func() string {
			return compactID(guid.S())
		},
		repo: repo,
	}
	if items, err := service.loadItems(); err == nil && len(items) > 0 {
		service.items = items
	}
	return service
}

func (s *Service) loadItems() ([]Item, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.LoadSampleQuestions()
}

// ListWelcome 返回欢迎页所需的示例问题列表。
func (s *Service) ListWelcome() []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]Item, len(s.items))
	copy(records, s.items)
	sort.Slice(records, func(i, j int) bool {
		return records[i].UpdateTime.After(records[j].UpdateTime)
	})
	if len(records) > welcomeQuestionLimit {
		records = records[:welcomeQuestionLimit]
	}
	return records
}

// Page 返回分页数据。
func (s *Service) Page(current, size int, keyword string) PageResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if current <= 0 {
		current = 1
	}
	if size <= 0 {
		size = 10
	}

	filtered := make([]Item, 0, len(s.items))
	for _, item := range s.items {
		if matchKeyword(item, keyword) {
			filtered = append(filtered, item)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].UpdateTime.After(filtered[j].UpdateTime)
	})

	total := len(filtered)
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

	records := make([]Item, end-start)
	copy(records, filtered[start:end])
	return PageResult{
		Records: records,
		Total:   total,
		Size:    size,
		Current: current,
		Pages:   pages,
	}
}

// GetByID 查询指定示例问题。
func (s *Service) GetByID(id string) (Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.items {
		if item.ID == id {
			return item, nil
		}
	}
	return Item{}, ErrQuestionNotFound
}

// Create 新增示例问题并返回主键。
func (s *Service) Create(req SaveRequest) (string, error) {
	question := strings.TrimSpace(req.Question)
	if question == "" {
		return "", ErrQuestionRequired
	}

	now := s.now()
	item := Item{
		ID:          s.newID(),
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Question:    question,
		CreateTime:  now,
		UpdateTime:  now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = append([]Item{item}, s.items...)
	if err := s.persistLocked(); err != nil {
		return "", err
	}
	return item.ID, nil
}

// Update 修改指定示例问题。
func (s *Service) Update(id string, req SaveRequest) error {
	question := strings.TrimSpace(req.Question)
	if question == "" {
		return ErrQuestionRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for index := range s.items {
		if s.items[index].ID != id {
			continue
		}
		s.items[index].Title = strings.TrimSpace(req.Title)
		s.items[index].Description = strings.TrimSpace(req.Description)
		s.items[index].Question = question
		s.items[index].UpdateTime = s.now()
		if err := s.persistLocked(); err != nil {
			return err
		}
		return nil
	}
	return ErrQuestionNotFound
}

// Delete 删除指定示例问题。
func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for index := range s.items {
		if s.items[index].ID != id {
			continue
		}
		s.items = append(s.items[:index], s.items[index+1:]...)
		if err := s.persistLocked(); err != nil {
			return err
		}
		return nil
	}
	return ErrQuestionNotFound
}

func (s *Service) persistLocked() error {
	if s.repo == nil {
		return nil
	}
	items := make([]Item, len(s.items))
	copy(items, s.items)
	return s.repo.SaveSampleQuestions(items)
}

func compactID(raw string) string {
	return strings.ReplaceAll(raw, "-", "")
}

func matchKeyword(item Item, keyword string) bool {
	needle := strings.ToLower(strings.TrimSpace(keyword))
	if needle == "" {
		return true
	}

	haystack := strings.ToLower(strings.Join([]string{
		item.Title,
		item.Description,
		item.Question,
	}, " "))
	return strings.Contains(haystack, needle)
}
