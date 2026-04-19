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

// SampleQuestionRecord 定义持久化到磁盘的示例问题记录。
type SampleQuestionRecord struct {
	ID          string    `json:"id"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	Question    string    `json:"question"`
	CreateTime  time.Time `json:"createTime,omitempty"`
	UpdateTime  time.Time `json:"updateTime,omitempty"`
}

// QueryMappingRecord 定义持久化到磁盘的关键词映射记录。
type QueryMappingRecord struct {
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

// IntentNodeRecord 定义持久化到磁盘的意图节点记录。
type IntentNodeRecord struct {
	ID                  int64     `json:"id"`
	IntentCode          string    `json:"intentCode"`
	Name                string    `json:"name"`
	Level               int       `json:"level"`
	ParentCode          string    `json:"parentCode,omitempty"`
	Description         string    `json:"description,omitempty"`
	Examples            []string  `json:"examples,omitempty"`
	CollectionName      string    `json:"collectionName,omitempty"`
	MCPToolID           string    `json:"mcpToolId,omitempty"`
	TopK                *int      `json:"topK,omitempty"`
	Kind                int       `json:"kind"`
	SortOrder           int       `json:"sortOrder"`
	Enabled             int       `json:"enabled"`
	PromptSnippet       string    `json:"promptSnippet,omitempty"`
	PromptTemplate      string    `json:"promptTemplate,omitempty"`
	ParamPromptTemplate string    `json:"paramPromptTemplate,omitempty"`
	CreateTime          time.Time `json:"createTime,omitempty"`
	UpdateTime          time.Time `json:"updateTime,omitempty"`
}

// KnowledgeBaseRecord 定义持久化到磁盘的知识库记录。
type KnowledgeBaseRecord struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	EmbeddingModel string    `json:"embeddingModel"`
	CollectionName string    `json:"collectionName"`
	CreatedBy      string    `json:"createdBy,omitempty"`
	DocumentCount  int       `json:"documentCount,omitempty"`
	CreateTime     time.Time `json:"createTime,omitempty"`
	UpdateTime     time.Time `json:"updateTime,omitempty"`
}

// KnowledgeDocumentRecord 定义持久化到磁盘的文档记录。
type KnowledgeDocumentRecord struct {
	ID              string    `json:"id"`
	KBID            string    `json:"kbId"`
	DocName         string    `json:"docName"`
	SourceType      string    `json:"sourceType,omitempty"`
	SourceLocation  string    `json:"sourceLocation,omitempty"`
	ScheduleEnabled int       `json:"scheduleEnabled,omitempty"`
	ScheduleCron    string    `json:"scheduleCron,omitempty"`
	Enabled         bool      `json:"enabled"`
	ChunkCount      int       `json:"chunkCount,omitempty"`
	FileURL         string    `json:"fileUrl,omitempty"`
	FileType        string    `json:"fileType,omitempty"`
	FileSize        int64     `json:"fileSize,omitempty"`
	ProcessMode     string    `json:"processMode,omitempty"`
	ChunkStrategy   string    `json:"chunkStrategy,omitempty"`
	ChunkConfig     string    `json:"chunkConfig,omitempty"`
	PipelineID      string    `json:"pipelineId,omitempty"`
	Status          string    `json:"status,omitempty"`
	CreatedBy       string    `json:"createdBy,omitempty"`
	UpdatedBy       string    `json:"updatedBy,omitempty"`
	CreateTime      time.Time `json:"createTime,omitempty"`
	UpdateTime      time.Time `json:"updateTime,omitempty"`
}

// KnowledgeChunkRecord 定义持久化到磁盘的 Chunk 记录。
type KnowledgeChunkRecord struct {
	ID          string    `json:"id"`
	KBID        string    `json:"kbId,omitempty"`
	DocID       string    `json:"docId"`
	ChunkIndex  int       `json:"chunkIndex,omitempty"`
	Content     string    `json:"content,omitempty"`
	ContentHash string    `json:"contentHash,omitempty"`
	CharCount   int       `json:"charCount,omitempty"`
	TokenCount  int       `json:"tokenCount,omitempty"`
	Enabled     int       `json:"enabled,omitempty"`
	CreateTime  time.Time `json:"createTime,omitempty"`
	UpdateTime  time.Time `json:"updateTime,omitempty"`
}

// KnowledgeChunkLogRecord 定义持久化到磁盘的文档分块日志记录。
type KnowledgeChunkLogRecord struct {
	ID              string    `json:"id"`
	DocID           string    `json:"docId"`
	Status          string    `json:"status"`
	ProcessMode     string    `json:"processMode,omitempty"`
	ChunkStrategy   string    `json:"chunkStrategy,omitempty"`
	PipelineID      string    `json:"pipelineId,omitempty"`
	PipelineName    string    `json:"pipelineName,omitempty"`
	ExtractDuration int64     `json:"extractDuration,omitempty"`
	ChunkDuration   int64     `json:"chunkDuration,omitempty"`
	EmbedDuration   int64     `json:"embedDuration,omitempty"`
	PersistDuration int64     `json:"persistDuration,omitempty"`
	OtherDuration   int64     `json:"otherDuration,omitempty"`
	TotalDuration   int64     `json:"totalDuration,omitempty"`
	ChunkCount      int       `json:"chunkCount,omitempty"`
	ErrorMessage    string    `json:"errorMessage,omitempty"`
	StartTime       time.Time `json:"startTime,omitempty"`
	EndTime         time.Time `json:"endTime,omitempty"`
	CreateTime      time.Time `json:"createTime,omitempty"`
}

// Snapshot 定义当前阶段状态快照。
type Snapshot struct {
	Users           []UserRecord              `json:"users,omitempty"`
	SampleQuestions []SampleQuestionRecord    `json:"sampleQuestions,omitempty"`
	QueryMappings   []QueryMappingRecord      `json:"queryMappings,omitempty"`
	IntentNodes     []IntentNodeRecord        `json:"intentNodes,omitempty"`
	KnowledgeBases  []KnowledgeBaseRecord     `json:"knowledgeBases,omitempty"`
	KnowledgeDocs   []KnowledgeDocumentRecord `json:"knowledgeDocs,omitempty"`
	KnowledgeChunks []KnowledgeChunkRecord    `json:"knowledgeChunks,omitempty"`
	KnowledgeLogs   []KnowledgeChunkLogRecord `json:"knowledgeLogs,omitempty"`
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

// Update 在持有文件锁的前提下读取、修改并写回快照，避免不同服务互相覆盖。
func (s *FileStore) Update(mutator func(snapshot *Snapshot)) error {
	if s == nil || s.path == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var snapshot Snapshot
	content, err := os.ReadFile(s.path)
	if err == nil && len(content) > 0 {
		if err := json.Unmarshal(content, &snapshot); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	mutator(&snapshot)

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	nextContent, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, nextContent, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}
