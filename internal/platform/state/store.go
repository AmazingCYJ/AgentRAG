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
	ID                  string    `json:"id"`
	KBID                string    `json:"kbId,omitempty"`
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
	TextContent     string    `json:"textContent,omitempty"`
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

// ConversationSessionRecord 定义持久化到磁盘的会话记录。
type ConversationSessionRecord struct {
	ConversationID string    `json:"conversationId"`
	UserID         string    `json:"userId"`
	Title          string    `json:"title"`
	LastTime       time.Time `json:"lastTime,omitempty"`
}

// ConversationMessageRecord 定义持久化到磁盘的消息记录。
type ConversationMessageRecord struct {
	ID               string    `json:"id"`
	ConversationID   string    `json:"conversationId"`
	UserID           string    `json:"userId"`
	Role             string    `json:"role"`
	Content          string    `json:"content"`
	ThinkingContent  string    `json:"thinkingContent,omitempty"`
	ThinkingDuration *int      `json:"thinkingDuration,omitempty"`
	Vote             *int      `json:"vote,omitempty"`
	CreateTime       time.Time `json:"createTime,omitempty"`
}

// RagTraceRunRecord 定义持久化到磁盘的 Trace 运行记录。
type RagTraceRunRecord struct {
	TraceID        string    `json:"traceId"`
	TraceName      string    `json:"traceName,omitempty"`
	EntryMethod    string    `json:"entryMethod,omitempty"`
	ConversationID string    `json:"conversationId,omitempty"`
	TaskID         string    `json:"taskId,omitempty"`
	UserName       string    `json:"userName,omitempty"`
	UserID         string    `json:"userId,omitempty"`
	Status         string    `json:"status,omitempty"`
	ErrorMessage   string    `json:"errorMessage,omitempty"`
	DurationMs     int64     `json:"durationMs,omitempty"`
	StartTime      time.Time `json:"startTime,omitempty"`
	EndTime        time.Time `json:"endTime,omitempty"`
}

// RagTraceNodeRecord 定义持久化到磁盘的 Trace 节点记录。
type RagTraceNodeRecord struct {
	TraceID      string    `json:"traceId"`
	NodeID       string    `json:"nodeId"`
	ParentNodeID string    `json:"parentNodeId,omitempty"`
	Depth        int       `json:"depth,omitempty"`
	NodeType     string    `json:"nodeType,omitempty"`
	NodeName     string    `json:"nodeName,omitempty"`
	ClassName    string    `json:"className,omitempty"`
	MethodName   string    `json:"methodName,omitempty"`
	Status       string    `json:"status,omitempty"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	DurationMs   int64     `json:"durationMs,omitempty"`
	StartTime    time.Time `json:"startTime,omitempty"`
	EndTime      time.Time `json:"endTime,omitempty"`
}

// IngestionPipelineRecord 定义持久化到磁盘的流水线记录。
type IngestionPipelineRecord struct {
	ID          string                        `json:"id"`
	Name        string                        `json:"name"`
	Description string                        `json:"description,omitempty"`
	CreatedBy   string                        `json:"createdBy,omitempty"`
	Nodes       []IngestionPipelineNodeRecord `json:"nodes,omitempty"`
	CreateTime  time.Time                     `json:"createTime,omitempty"`
	UpdateTime  time.Time                     `json:"updateTime,omitempty"`
}

// IngestionPipelineNodeRecord 定义持久化到磁盘的流水线节点记录。
type IngestionPipelineNodeRecord struct {
	ID         int64          `json:"id"`
	NodeID     string         `json:"nodeId"`
	NodeType   string         `json:"nodeType"`
	Settings   map[string]any `json:"settings,omitempty"`
	Condition  map[string]any `json:"condition,omitempty"`
	NextNodeID string         `json:"nextNodeId,omitempty"`
}

// IngestionTaskRecord 定义持久化到磁盘的采集任务记录。
type IngestionTaskRecord struct {
	ID             string                   `json:"id"`
	PipelineID     string                   `json:"pipelineId"`
	SourceType     string                   `json:"sourceType,omitempty"`
	SourceLocation string                   `json:"sourceLocation,omitempty"`
	SourceFileName string                   `json:"sourceFileName,omitempty"`
	Status         string                   `json:"status,omitempty"`
	ChunkCount     int                      `json:"chunkCount,omitempty"`
	ErrorMessage   string                   `json:"errorMessage,omitempty"`
	Logs           []IngestionTaskLogRecord `json:"logs,omitempty"`
	Metadata       map[string]any           `json:"metadata,omitempty"`
	StartedAt      time.Time                `json:"startedAt,omitempty"`
	CompletedAt    time.Time                `json:"completedAt,omitempty"`
	CreatedBy      string                   `json:"createdBy,omitempty"`
	CreateTime     time.Time                `json:"createTime,omitempty"`
	UpdateTime     time.Time                `json:"updateTime,omitempty"`
}

// IngestionTaskLogRecord 定义持久化到磁盘的任务日志记录。
type IngestionTaskLogRecord struct {
	NodeID     string `json:"nodeId"`
	NodeType   string `json:"nodeType"`
	Message    string `json:"message,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// IngestionTaskNodeRecord 定义持久化到磁盘的任务节点记录。
type IngestionTaskNodeRecord struct {
	ID           string         `json:"id"`
	TaskID       string         `json:"taskId"`
	PipelineID   string         `json:"pipelineId"`
	NodeID       string         `json:"nodeId"`
	NodeType     string         `json:"nodeType"`
	NodeOrder    int            `json:"nodeOrder,omitempty"`
	Status       string         `json:"status,omitempty"`
	DurationMs   int64          `json:"durationMs,omitempty"`
	Message      string         `json:"message,omitempty"`
	ErrorMessage string         `json:"errorMessage,omitempty"`
	Output       map[string]any `json:"output,omitempty"`
	CreateTime   time.Time      `json:"createTime,omitempty"`
	UpdateTime   time.Time      `json:"updateTime,omitempty"`
}

// Snapshot 定义当前阶段状态快照。
type Snapshot struct {
	Users                []UserRecord                `json:"users,omitempty"`
	SampleQuestions      []SampleQuestionRecord      `json:"sampleQuestions,omitempty"`
	QueryMappings        []QueryMappingRecord        `json:"queryMappings,omitempty"`
	IntentNodes          []IntentNodeRecord          `json:"intentNodes,omitempty"`
	KnowledgeBases       []KnowledgeBaseRecord       `json:"knowledgeBases,omitempty"`
	KnowledgeDocs        []KnowledgeDocumentRecord   `json:"knowledgeDocs,omitempty"`
	KnowledgeChunks      []KnowledgeChunkRecord      `json:"knowledgeChunks,omitempty"`
	KnowledgeLogs        []KnowledgeChunkLogRecord   `json:"knowledgeLogs,omitempty"`
	ConversationSessions []ConversationSessionRecord `json:"conversationSessions,omitempty"`
	ConversationMessages []ConversationMessageRecord `json:"conversationMessages,omitempty"`
	RagTraceRuns         []RagTraceRunRecord         `json:"ragTraceRuns,omitempty"`
	RagTraceNodes        []RagTraceNodeRecord        `json:"ragTraceNodes,omitempty"`
	IngestionPipelines   []IngestionPipelineRecord   `json:"ingestionPipelines,omitempty"`
	IngestionTasks       []IngestionTaskRecord       `json:"ingestionTasks,omitempty"`
	IngestionTaskNodes   []IngestionTaskNodeRecord   `json:"ingestionTaskNodes,omitempty"`
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
