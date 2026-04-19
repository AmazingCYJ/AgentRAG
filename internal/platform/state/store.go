package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// UserRecord 定义持久化到磁盘的用户记录。
type UserRecord struct {
	ID         string    `json:"id"`
	Username   string    `json:"username"`
	Password   string    `json:"password"`
	Role       string    `json:"role"`
	Avatar     string    `json:"avatar,omitempty"`
	CreateTime time.Time `json:"createTime,omitempty"`
	UpdateTime time.Time `json:"updateTime,omitempty"`
}

// Snapshot 定义当前阶段状态快照。
type Snapshot struct {
	Users []UserRecord `json:"users,omitempty"`
}

// FileStore 提供基于 JSON 文件的轻量持久化。
type FileStore struct {
	mu   sync.Mutex
	path string
}

// NewFileStore 创建文件状态存储。
func NewFileStore(path string) (*FileStore, error) {
	return &FileStore{path: path}, nil
}

// Load 读取状态快照，不存在时返回空结果。
func (s *FileStore) Load() (Snapshot, error) {
	if s == nil || s.path == "" {
		return Snapshot{}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	content, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return Snapshot{}, nil
		}
		return Snapshot{}, err
	}
	if len(content) == 0 {
		return Snapshot{}, nil
	}

	var snapshot Snapshot
	if err := json.Unmarshal(content, &snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

// Save 写入状态快照。
func (s *FileStore) Save(snapshot Snapshot) error {
	if s == nil || s.path == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}
