package usermgmt

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	platformstate "github.com/AmazingCYJ/AgentRAG/internal/platform/state"
	"github.com/gogf/gf/v2/util/guid"
)

var (
	// ErrUserNotFound 表示用户不存在。
	ErrUserNotFound = errors.New("用户不存在")
	// ErrUsernameRequired 表示用户名不能为空。
	ErrUsernameRequired = errors.New("用户名不能为空")
	// ErrPasswordRequired 表示密码不能为空。
	ErrPasswordRequired = errors.New("密码不能为空")
	// ErrUsernameExists 表示用户名已存在。
	ErrUsernameExists = errors.New("用户名已存在")
	// ErrProtectedUser 表示默认管理员不可删除。
	ErrProtectedUser = errors.New("默认管理员不可删除")
	// ErrInvalidCurrentPassword 表示当前密码不正确。
	ErrInvalidCurrentPassword = errors.New("当前密码错误")
	// ErrNewPasswordRequired 表示新密码不能为空。
	ErrNewPasswordRequired = errors.New("新密码不能为空")
)

// User 表示后台用户实体。
type User struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	Password   string
	Role       string    `json:"role"`
	Avatar     string    `json:"avatar,omitempty"`
	CreateTime time.Time `json:"createTime,omitempty"`
	UpdateTime time.Time `json:"updateTime,omitempty"`
}

// PageResult 定义用户分页结构。
type PageResult struct {
	Records []User `json:"records"`
	Total   int    `json:"total"`
	Size    int    `json:"size"`
	Current int    `json:"current"`
	Pages   int    `json:"pages"`
}

// CreateRequest 定义用户创建输入。
type CreateRequest struct {
	Username string
	Password string
	Role     string
	Avatar   string
}

// UpdateRequest 定义用户更新输入。
type UpdateRequest struct {
	Username string
	Password string
	Role     string
	Avatar   string
}

// UserRepository 定义用户持久化仓储。
type UserRepository interface {
	LoadUsers() ([]User, error)
	SaveUsers(users []User) error
}

type fileStoreUserRepository struct {
	store *platformstate.FileStore
}

func (r *fileStoreUserRepository) LoadUsers() ([]User, error) {
	if r == nil || r.store == nil {
		return nil, nil
	}
	snapshot, err := r.store.Load()
	if err != nil {
		return nil, err
	}
	users := make([]User, 0, len(snapshot.Users))
	for _, record := range snapshot.Users {
		users = append(users, User{
			ID:         record.ID,
			Username:   record.Username,
			Password:   record.Password,
			Role:       record.Role,
			Avatar:     record.Avatar,
			CreateTime: record.CreateTime,
			UpdateTime: record.UpdateTime,
		})
	}
	return users, nil
}

func (r *fileStoreUserRepository) SaveUsers(users []User) error {
	if r == nil || r.store == nil {
		return nil
	}
	records := make([]platformstate.UserRecord, 0, len(users))
	for _, user := range users {
		records = append(records, platformstate.UserRecord{
			ID:         user.ID,
			Username:   user.Username,
			Password:   user.Password,
			Role:       user.Role,
			Avatar:     user.Avatar,
			CreateTime: user.CreateTime,
			UpdateTime: user.UpdateTime,
		})
	}
	return r.store.Save(platformstate.Snapshot{Users: records})
}

// Service 提供后台用户管理能力。
type Service struct {
	mu          sync.RWMutex
	users       map[string]User
	protectedID string
	now         func() time.Time
	newID       func() string
	repository  UserRepository
}

// NewService 创建用户管理服务，并注入默认管理员。
func NewService(cfg appconfig.AuthConfig, store *platformstate.FileStore) *Service {
	var repository UserRepository
	if store != nil {
		repository = &fileStoreUserRepository{store: store}
	}
	return NewServiceWithRepository(cfg, repository)
}

// NewServiceWithRepository 创建基于指定仓储的用户管理服务。
func NewServiceWithRepository(cfg appconfig.AuthConfig, repository UserRepository) *Service {
	now := time.Now()
	adminID := strings.TrimSpace(cfg.Bootstrap.UserID)
	if adminID == "" {
		adminID = "u_admin"
	}
	admin := User{
		ID:         adminID,
		Username:   fallback(cfg.Bootstrap.Username, "admin"),
		Password:   fallback(cfg.Bootstrap.Password, "admin123"),
		Role:       fallback(cfg.Bootstrap.Role, "admin"),
		Avatar:     strings.TrimSpace(cfg.Bootstrap.Avatar),
		CreateTime: now,
		UpdateTime: now,
	}
	service := &Service{
		users:       map[string]User{},
		protectedID: admin.ID,
		now:         time.Now,
		newID: func() string {
			return "user_" + strings.ReplaceAll(guid.S(), "-", "")
		},
		repository: repository,
	}
	if users, err := service.loadUsers(); err == nil && len(users) > 0 {
		for _, user := range users {
			service.users[user.ID] = user
		}
	}
	if _, ok := service.users[admin.ID]; !ok {
		service.users[admin.ID] = admin
	}
	return service
}

func (s *Service) loadUsers() ([]User, error) {
	if s.repository == nil {
		return nil, nil
	}
	return s.repository.LoadUsers()
}

// Page 分页查询用户。
func (s *Service) Page(current, size int, keyword string) PageResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if current <= 0 {
		current = 1
	}
	if size <= 0 {
		size = 10
	}

	items := make([]User, 0, len(s.users))
	for _, user := range s.users {
		if !containsFold(user.Username, keyword) && !containsFold(user.Role, keyword) {
			continue
		}
		items = append(items, sanitize(user))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreateTime.Before(items[j].CreateTime)
	})

	total := len(items)
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
	records := make([]User, end-start)
	copy(records, items[start:end])
	return PageResult{
		Records: records,
		Total:   total,
		Size:    size,
		Current: current,
		Pages:   pages,
	}
}

// Create 创建用户。
func (s *Service) Create(req CreateRequest) (string, error) {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return "", ErrUsernameRequired
	}
	password := strings.TrimSpace(req.Password)
	if password == "" {
		return "", ErrPasswordRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.usernameExistsLocked(username, "") {
		return "", ErrUsernameExists
	}

	now := s.now()
	id := s.newID()
	s.users[id] = User{
		ID:         id,
		Username:   username,
		Password:   password,
		Role:       fallback(req.Role, "user"),
		Avatar:     strings.TrimSpace(req.Avatar),
		CreateTime: now,
		UpdateTime: now,
	}
	if err := s.persistLocked(); err != nil {
		return "", err
	}
	return id, nil
}

// Update 更新用户。
func (s *Service) Update(id string, req UpdateRequest) error {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return ErrUsernameRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}
	if s.usernameExistsLocked(username, id) {
		return ErrUsernameExists
	}
	user.Username = username
	if strings.TrimSpace(req.Password) != "" {
		user.Password = strings.TrimSpace(req.Password)
	}
	user.Role = fallback(req.Role, user.Role)
	user.Avatar = strings.TrimSpace(req.Avatar)
	user.UpdateTime = s.now()
	s.users[id] = user
	if err := s.persistLocked(); err != nil {
		return err
	}
	return nil
}

// Delete 删除用户。
func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id == s.protectedID {
		return ErrProtectedUser
	}
	if _, ok := s.users[id]; !ok {
		return ErrUserNotFound
	}
	delete(s.users, id)
	if err := s.persistLocked(); err != nil {
		return err
	}
	return nil
}

// Count 返回用户总数。
func (s *Service) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users)
}

// GetByID 根据用户ID查询用户。
func (s *Service) GetByID(id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}

// Authenticate 根据用户名和密码认证用户。
func (s *Service) Authenticate(username, password string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if strings.EqualFold(user.Username, strings.TrimSpace(username)) && user.Password == password {
			return user, true
		}
	}
	return User{}, false
}

// ChangePassword 修改指定用户密码。
func (s *Service) ChangePassword(id, currentPassword, newPassword string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}
	if user.Password != currentPassword {
		return ErrInvalidCurrentPassword
	}
	if strings.TrimSpace(newPassword) == "" {
		return ErrNewPasswordRequired
	}
	user.Password = newPassword
	user.UpdateTime = s.now()
	s.users[id] = user
	if err := s.persistLocked(); err != nil {
		return err
	}
	return nil
}

func (s *Service) persistLocked() error {
	if s.repository == nil {
		return nil
	}
	users := make([]User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].CreateTime.Before(users[j].CreateTime)
	})
	return s.repository.SaveUsers(users)
}

func (s *Service) usernameExistsLocked(username, excludeID string) bool {
	for id, user := range s.users {
		if id == excludeID {
			continue
		}
		if strings.EqualFold(user.Username, username) {
			return true
		}
	}
	return false
}

func sanitize(user User) User {
	user.Password = ""
	return user
}

func containsFold(value, keyword string) bool {
	needle := strings.ToLower(strings.TrimSpace(keyword))
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(value), needle)
}

func fallback(value, fallbackValue string) string {
	if strings.TrimSpace(value) == "" {
		return fallbackValue
	}
	return strings.TrimSpace(value)
}
