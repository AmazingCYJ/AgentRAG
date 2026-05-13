package knowledge

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	platformstate "github.com/AmazingCYJ/AgentRAG/internal/platform/state"
	"github.com/gogf/gf/v2/util/guid"
)

var (
	// ErrKnowledgeBaseNotFound 表示知识库不存在。
	ErrKnowledgeBaseNotFound = errors.New("知识库不存在")
	// ErrKnowledgeBaseNameRequired 表示知识库名称不能为空。
	ErrKnowledgeBaseNameRequired = errors.New("知识库名称不能为空")
	// ErrEmbeddingModelRequired 表示嵌入模型不能为空。
	ErrEmbeddingModelRequired = errors.New("Embedding 模型不能为空")
	// ErrCollectionNameRequired 表示集合名称不能为空。
	ErrCollectionNameRequired = errors.New("Collection 名称不能为空")
	// ErrDocumentNotFound 表示文档不存在。
	ErrDocumentNotFound = errors.New("文档不存在")
	// ErrDocumentNameRequired 表示文档名称不能为空。
	ErrDocumentNameRequired = errors.New("文档名称不能为空")
	// ErrChunkNotFound 表示分块不存在。
	ErrChunkNotFound = errors.New("Chunk 不存在")
	// ErrChunkContentRequired 表示分块内容不能为空。
	ErrChunkContentRequired = errors.New("Chunk 内容不能为空")
)

var businessSynonyms = map[string][]string{
	"报账":   {"报销"},
	"报销":   {"报账"},
	"付款账户": {"付款账号"},
	"付款账号": {"付款账户"},
	"账户":   {"账号"},
	"账号":   {"账户"},
	"休假":   {"请假"},
	"请假":   {"休假"},
	"审批人":  {"负责人"},
	"负责人":  {"审批人"},
}

// PageResult 定义统一分页结构。
type PageResult[T any] struct {
	Records []T `json:"records"`
	Total   int `json:"total"`
	Size    int `json:"size"`
	Current int `json:"current"`
	Pages   int `json:"pages"`
}

// KnowledgeBase 表示知识库对象。
type KnowledgeBase struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	EmbeddingModel string    `json:"embeddingModel"`
	CollectionName string    `json:"collectionName"`
	CreatedBy      string    `json:"createdBy,omitempty"`
	DocumentCount  int       `json:"documentCount,omitempty"`
	CreateTime     time.Time `json:"createTime,omitempty"`
	UpdateTime     time.Time `json:"updateTime,omitempty"`
}

// ChunkStrategy 表示分块策略选项。
type ChunkStrategy struct {
	Value         string         `json:"value"`
	Label         string         `json:"label"`
	DefaultConfig map[string]int `json:"defaultConfig"`
}

// KnowledgeDocument 表示文档对象。
type KnowledgeDocument struct {
	ID              string    `json:"id"`
	KBID            string    `json:"kbId"`
	DocName         string    `json:"docName"`
	SourceType      string    `json:"sourceType,omitempty"`
	SourceLocation  string    `json:"sourceLocation,omitempty"`
	TextContent     string    `json:"-"`
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

// KnowledgeDocumentSearchItem 表示文档搜索项。
type KnowledgeDocumentSearchItem struct {
	ID      string `json:"id"`
	KBID    string `json:"kbId"`
	DocName string `json:"docName"`
	KBName  string `json:"kbName,omitempty"`
}

// KnowledgeChunk 表示分块对象。
type KnowledgeChunk struct {
	ID             string    `json:"id"`
	KBID           string    `json:"kbId,omitempty"`
	DocID          string    `json:"docId"`
	ChunkIndex     int       `json:"chunkIndex,omitempty"`
	Content        string    `json:"content,omitempty"`
	ContentHash    string    `json:"contentHash,omitempty"`
	CharCount      int       `json:"charCount,omitempty"`
	TokenCount     int       `json:"tokenCount,omitempty"`
	EmbeddingModel string    `json:"embeddingModel,omitempty"`
	Embedding      []float64 `json:"-"`
	Enabled        int       `json:"enabled,omitempty"`
	CreateTime     time.Time `json:"createTime,omitempty"`
	UpdateTime     time.Time `json:"updateTime,omitempty"`
}

// KnowledgeDocumentChunkLog 表示文档分块日志。
type KnowledgeDocumentChunkLog struct {
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

// KnowledgeBaseCreateRequest 定义知识库创建输入。
type KnowledgeBaseCreateRequest struct {
	Name           string
	EmbeddingModel string
	CollectionName string
	CreatedBy      string
}

// KnowledgeBaseUpdateRequest 定义知识库更新输入。
type KnowledgeBaseUpdateRequest struct {
	Name           string
	EmbeddingModel string
}

// KnowledgeBasePageRequest 定义知识库分页请求。
type KnowledgeBasePageRequest struct {
	Current int
	Size    int
	Name    string
}

// KnowledgeDocumentUploadRequest 定义文档上传输入。
type KnowledgeDocumentUploadRequest struct {
	SourceType      string
	SourceLocation  string
	TextContent     string
	ScheduleEnabled bool
	ScheduleCron    string
	ProcessMode     string
	ChunkStrategy   string
	ChunkConfig     string
	PipelineID      string
	FileName        string
	FileSize        int64
}

// KnowledgeDocumentUpdateRequest 定义文档更新输入。
type KnowledgeDocumentUpdateRequest struct {
	DocName         string
	ProcessMode     string
	ChunkStrategy   string
	ChunkConfig     string
	PipelineID      string
	SourceLocation  string
	ScheduleEnabled *int
	ScheduleCron    string
	UpdatedBy       string
}

// KnowledgeDocumentPageRequest 定义文档分页请求。
type KnowledgeDocumentPageRequest struct {
	Current int
	Size    int
	Status  string
	Keyword string
}

// KnowledgeChunkPageRequest 定义 Chunk 分页请求。
type KnowledgeChunkPageRequest struct {
	Current int
	Size    int
	Enabled *int
}

// KnowledgeChunkCreateRequest 定义 Chunk 创建输入。
type KnowledgeChunkCreateRequest struct {
	Content string
	Index   *int
	ChunkID string
}

// KnowledgeChunkUpdateRequest 定义 Chunk 更新输入。
type KnowledgeChunkUpdateRequest struct {
	Content string
}

// Repository 定义知识库、文档、Chunk 和处理日志的持久化仓储。
type Repository interface {
	LoadKnowledgeRecords() ([]KnowledgeBase, []KnowledgeDocument, []KnowledgeChunk, []KnowledgeDocumentChunkLog, error)
	SaveKnowledgeRecords(bases []KnowledgeBase, docs []KnowledgeDocument, chunks []KnowledgeChunk, logs []KnowledgeDocumentChunkLog) error
}

type fileStoreRepository struct {
	store *platformstate.FileStore
}

func (r *fileStoreRepository) LoadKnowledgeRecords() ([]KnowledgeBase, []KnowledgeDocument, []KnowledgeChunk, []KnowledgeDocumentChunkLog, error) {
	if r == nil || r.store == nil {
		return nil, nil, nil, nil, nil
	}
	snapshot, err := r.store.Load()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	bases := make([]KnowledgeBase, 0, len(snapshot.KnowledgeBases))
	for _, item := range snapshot.KnowledgeBases {
		bases = append(bases, KnowledgeBase{
			ID:             item.ID,
			Name:           item.Name,
			EmbeddingModel: item.EmbeddingModel,
			CollectionName: item.CollectionName,
			CreatedBy:      item.CreatedBy,
			DocumentCount:  item.DocumentCount,
			CreateTime:     item.CreateTime,
			UpdateTime:     item.UpdateTime,
		})
	}
	docs := make([]KnowledgeDocument, 0, len(snapshot.KnowledgeDocs))
	for _, item := range snapshot.KnowledgeDocs {
		docs = append(docs, KnowledgeDocument{
			ID:              item.ID,
			KBID:            item.KBID,
			DocName:         item.DocName,
			SourceType:      item.SourceType,
			SourceLocation:  item.SourceLocation,
			TextContent:     item.TextContent,
			ScheduleEnabled: item.ScheduleEnabled,
			ScheduleCron:    item.ScheduleCron,
			Enabled:         item.Enabled,
			ChunkCount:      item.ChunkCount,
			FileURL:         item.FileURL,
			FileType:        item.FileType,
			FileSize:        item.FileSize,
			ProcessMode:     item.ProcessMode,
			ChunkStrategy:   item.ChunkStrategy,
			ChunkConfig:     item.ChunkConfig,
			PipelineID:      item.PipelineID,
			Status:          item.Status,
			CreatedBy:       item.CreatedBy,
			UpdatedBy:       item.UpdatedBy,
			CreateTime:      item.CreateTime,
			UpdateTime:      item.UpdateTime,
		})
	}
	chunks := make([]KnowledgeChunk, 0, len(snapshot.KnowledgeChunks))
	for _, item := range snapshot.KnowledgeChunks {
		chunks = append(chunks, KnowledgeChunk{
			ID:             item.ID,
			KBID:           item.KBID,
			DocID:          item.DocID,
			ChunkIndex:     item.ChunkIndex,
			Content:        item.Content,
			ContentHash:    item.ContentHash,
			CharCount:      item.CharCount,
			TokenCount:     item.TokenCount,
			EmbeddingModel: item.EmbeddingModel,
			Embedding:      cloneFloat64Slice(item.Embedding),
			Enabled:        item.Enabled,
			CreateTime:     item.CreateTime,
			UpdateTime:     item.UpdateTime,
		})
	}
	logs := make([]KnowledgeDocumentChunkLog, 0, len(snapshot.KnowledgeLogs))
	for _, item := range snapshot.KnowledgeLogs {
		logs = append(logs, KnowledgeDocumentChunkLog{
			ID:              item.ID,
			DocID:           item.DocID,
			Status:          item.Status,
			ProcessMode:     item.ProcessMode,
			ChunkStrategy:   item.ChunkStrategy,
			PipelineID:      item.PipelineID,
			PipelineName:    item.PipelineName,
			ExtractDuration: item.ExtractDuration,
			ChunkDuration:   item.ChunkDuration,
			EmbedDuration:   item.EmbedDuration,
			PersistDuration: item.PersistDuration,
			OtherDuration:   item.OtherDuration,
			TotalDuration:   item.TotalDuration,
			ChunkCount:      item.ChunkCount,
			ErrorMessage:    item.ErrorMessage,
			StartTime:       item.StartTime,
			EndTime:         item.EndTime,
			CreateTime:      item.CreateTime,
		})
	}
	return bases, docs, chunks, logs, nil
}

func (r *fileStoreRepository) SaveKnowledgeRecords(bases []KnowledgeBase, docs []KnowledgeDocument, chunks []KnowledgeChunk, logs []KnowledgeDocumentChunkLog) error {
	if r == nil || r.store == nil {
		return nil
	}
	baseRecords := make([]platformstate.KnowledgeBaseRecord, 0, len(bases))
	for _, item := range bases {
		baseRecords = append(baseRecords, platformstate.KnowledgeBaseRecord{
			ID:             item.ID,
			Name:           item.Name,
			EmbeddingModel: item.EmbeddingModel,
			CollectionName: item.CollectionName,
			CreatedBy:      item.CreatedBy,
			DocumentCount:  item.DocumentCount,
			CreateTime:     item.CreateTime,
			UpdateTime:     item.UpdateTime,
		})
	}
	docRecords := make([]platformstate.KnowledgeDocumentRecord, 0, len(docs))
	for _, item := range docs {
		docRecords = append(docRecords, platformstate.KnowledgeDocumentRecord{
			ID:              item.ID,
			KBID:            item.KBID,
			DocName:         item.DocName,
			SourceType:      item.SourceType,
			SourceLocation:  item.SourceLocation,
			TextContent:     item.TextContent,
			ScheduleEnabled: item.ScheduleEnabled,
			ScheduleCron:    item.ScheduleCron,
			Enabled:         item.Enabled,
			ChunkCount:      item.ChunkCount,
			FileURL:         item.FileURL,
			FileType:        item.FileType,
			FileSize:        item.FileSize,
			ProcessMode:     item.ProcessMode,
			ChunkStrategy:   item.ChunkStrategy,
			ChunkConfig:     item.ChunkConfig,
			PipelineID:      item.PipelineID,
			Status:          item.Status,
			CreatedBy:       item.CreatedBy,
			UpdatedBy:       item.UpdatedBy,
			CreateTime:      item.CreateTime,
			UpdateTime:      item.UpdateTime,
		})
	}
	chunkRecords := make([]platformstate.KnowledgeChunkRecord, 0, len(chunks))
	for _, item := range chunks {
		chunkRecords = append(chunkRecords, platformstate.KnowledgeChunkRecord{
			ID:             item.ID,
			KBID:           item.KBID,
			DocID:          item.DocID,
			ChunkIndex:     item.ChunkIndex,
			Content:        item.Content,
			ContentHash:    item.ContentHash,
			CharCount:      item.CharCount,
			TokenCount:     item.TokenCount,
			EmbeddingModel: item.EmbeddingModel,
			Embedding:      cloneFloat64Slice(item.Embedding),
			Enabled:        item.Enabled,
			CreateTime:     item.CreateTime,
			UpdateTime:     item.UpdateTime,
		})
	}
	logRecords := make([]platformstate.KnowledgeChunkLogRecord, 0, len(logs))
	for _, item := range logs {
		logRecords = append(logRecords, platformstate.KnowledgeChunkLogRecord{
			ID:              item.ID,
			DocID:           item.DocID,
			Status:          item.Status,
			ProcessMode:     item.ProcessMode,
			ChunkStrategy:   item.ChunkStrategy,
			PipelineID:      item.PipelineID,
			PipelineName:    item.PipelineName,
			ExtractDuration: item.ExtractDuration,
			ChunkDuration:   item.ChunkDuration,
			EmbedDuration:   item.EmbedDuration,
			PersistDuration: item.PersistDuration,
			OtherDuration:   item.OtherDuration,
			TotalDuration:   item.TotalDuration,
			ChunkCount:      item.ChunkCount,
			ErrorMessage:    item.ErrorMessage,
			StartTime:       item.StartTime,
			EndTime:         item.EndTime,
			CreateTime:      item.CreateTime,
		})
	}
	return r.store.Update(func(snapshot *platformstate.Snapshot) {
		snapshot.KnowledgeBases = baseRecords
		snapshot.KnowledgeDocs = docRecords
		snapshot.KnowledgeChunks = chunkRecords
		snapshot.KnowledgeLogs = logRecords
	})
}

// Service 提供知识库相关的内存存储与业务能力。
type Service struct {
	mu sync.RWMutex

	knowledgeBases map[string]KnowledgeBase
	documents      map[string]KnowledgeDocument
	chunks         map[string]KnowledgeChunk
	chunkLogs      map[string][]KnowledgeDocumentChunkLog

	now         func() time.Time
	newID       func() string
	repo        Repository
	embed       EmbeddingService
	vectorStore VectorStore
}

type knowledgeSnapshot struct {
	knowledgeBases map[string]KnowledgeBase
	documents      map[string]KnowledgeDocument
	chunks         map[string]KnowledgeChunk
}

type promptContextMatch struct {
	kbName     string
	docName    string
	source     string
	chunkIndex int
	content    string
	score      retrievalScore
}

// NewService 创建知识库服务。
func NewService(store *platformstate.FileStore) *Service {
	var repo Repository
	if store != nil {
		repo = &fileStoreRepository{store: store}
	}
	return NewServiceWithRepository(repo)
}

// NewServiceWithRepository 创建基于指定仓储的知识库服务。
func NewServiceWithRepository(repo Repository) *Service {
	return NewServiceWithDependencies(repo, nil)
}

// NewServiceWithDependencies 创建基于指定依赖的知识库服务。
func NewServiceWithDependencies(repo Repository, embed EmbeddingService, vectorStores ...VectorStore) *Service {
	if embed == nil {
		embed = newLocalEmbeddingService()
	}
	var vectorStore VectorStore
	if len(vectorStores) > 0 {
		vectorStore = vectorStores[0]
	}
	service := &Service{
		knowledgeBases: make(map[string]KnowledgeBase),
		documents:      make(map[string]KnowledgeDocument),
		chunks:         make(map[string]KnowledgeChunk),
		chunkLogs:      make(map[string][]KnowledgeDocumentChunkLog),
		now:            time.Now,
		newID:          func() string { return strings.ReplaceAll(guid.S(), "-", "") },
		repo:           repo,
		embed:          embed,
		vectorStore:    vectorStore,
	}
	if bases, docs, chunks, logs, err := service.loadRecords(); err == nil {
		for _, item := range bases {
			service.knowledgeBases[item.ID] = item
		}
		for _, item := range docs {
			service.documents[item.ID] = item
		}
		for _, item := range chunks {
			service.chunks[item.ID] = item
		}
		for _, item := range logs {
			service.chunkLogs[item.DocID] = append(service.chunkLogs[item.DocID], item)
		}
	}
	return service
}

func (s *Service) loadRecords() ([]KnowledgeBase, []KnowledgeDocument, []KnowledgeChunk, []KnowledgeDocumentChunkLog, error) {
	if s.repo == nil {
		return nil, nil, nil, nil, nil
	}
	return s.repo.LoadKnowledgeRecords()
}

// ListChunkStrategies 返回支持的分块策略列表。
func (s *Service) ListChunkStrategies() []ChunkStrategy {
	return []ChunkStrategy{
		{
			Value: "fixed_size",
			Label: "固定大小",
			DefaultConfig: map[string]int{
				"chunkSize":   512,
				"overlapSize": 128,
			},
		},
		{
			Value: "structure_aware",
			Label: "语义感知（Markdown友好）",
			DefaultConfig: map[string]int{
				"targetChars":  1400,
				"maxChars":     1800,
				"minChars":     600,
				"overlapChars": 0,
			},
		},
	}
}

// CreateKnowledgeBase 创建知识库。
func (s *Service) CreateKnowledgeBase(req KnowledgeBaseCreateRequest) (string, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return "", ErrKnowledgeBaseNameRequired
	}
	embeddingModel := strings.TrimSpace(req.EmbeddingModel)
	if embeddingModel == "" {
		return "", ErrEmbeddingModelRequired
	}
	collectionName := strings.TrimSpace(req.CollectionName)
	if collectionName == "" {
		return "", ErrCollectionNameRequired
	}

	now := s.now()
	id := "kb_" + s.newID()
	item := KnowledgeBase{
		ID:             id,
		Name:           name,
		EmbeddingModel: embeddingModel,
		CollectionName: collectionName,
		CreatedBy:      strings.TrimSpace(req.CreatedBy),
		CreateTime:     now,
		UpdateTime:     now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.knowledgeBases[id] = item
	if err := s.persistLocked(); err != nil {
		return "", err
	}
	return id, nil
}

// UpdateKnowledgeBase 更新知识库。
func (s *Service) UpdateKnowledgeBase(id string, req KnowledgeBaseUpdateRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.knowledgeBases[id]
	if !ok {
		return ErrKnowledgeBaseNotFound
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		item.Name = name
	}
	if embeddingModel := strings.TrimSpace(req.EmbeddingModel); embeddingModel != "" {
		item.EmbeddingModel = embeddingModel
	}
	item.UpdateTime = s.now()
	s.knowledgeBases[id] = item
	if err := s.persistLocked(); err != nil {
		return err
	}
	return nil
}

// DeleteKnowledgeBase 删除知识库及其关联数据。
func (s *Service) DeleteKnowledgeBase(ctx context.Context, id string) error {
	ctx = normalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.knowledgeBases[id]; !ok {
		return ErrKnowledgeBaseNotFound
	}
	vectorIDs := make([]string, 0)
	delete(s.knowledgeBases, id)
	for docID, doc := range s.documents {
		if doc.KBID != id {
			continue
		}
		delete(s.documents, docID)
		delete(s.chunkLogs, docID)
		for chunkID, chunk := range s.chunks {
			if chunk.DocID == docID {
				vectorIDs = append(vectorIDs, chunk.ID)
				delete(s.chunks, chunkID)
			}
		}
	}
	if err := s.persistLocked(); err != nil {
		return err
	}
	if err := s.deleteVectors(ctx, vectorIDs); err != nil {
		return err
	}
	return nil
}

// GetKnowledgeBase 查询知识库详情。
func (s *Service) GetKnowledgeBase(id string) (KnowledgeBase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.knowledgeBases[id]
	if !ok {
		return KnowledgeBase{}, ErrKnowledgeBaseNotFound
	}
	item.DocumentCount = s.countDocumentsByKBLocked(id)
	return item, nil
}

// PageKnowledgeBases 分页查询知识库。
func (s *Service) PageKnowledgeBases(req KnowledgeBasePageRequest) PageResult[KnowledgeBase] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	current, size := normalizePage(req.Current, req.Size)
	filtered := make([]KnowledgeBase, 0, len(s.knowledgeBases))
	for _, item := range s.knowledgeBases {
		if !containsFold(item.Name, req.Name) {
			continue
		}
		item.DocumentCount = s.countDocumentsByKBLocked(item.ID)
		filtered = append(filtered, item)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreateTime.After(filtered[j].CreateTime)
	})
	return paginate(filtered, current, size)
}

// UploadDocument 上传文档并返回记录。
func (s *Service) UploadDocument(kbID string, req KnowledgeDocumentUploadRequest, operator string) (KnowledgeDocument, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	kb, ok := s.knowledgeBases[kbID]
	if !ok {
		return KnowledgeDocument{}, ErrKnowledgeBaseNotFound
	}

	now := s.now()
	docID := "doc_" + s.newID()
	docName := inferDocName(req)
	if docName == "" {
		return KnowledgeDocument{}, ErrDocumentNameRequired
	}
	processMode := normalizeProcessMode(req.ProcessMode)
	chunkStrategy := normalizeChunkStrategy(req.ChunkStrategy)
	item := KnowledgeDocument{
		ID:              docID,
		KBID:            kbID,
		DocName:         docName,
		SourceType:      normalizeSourceType(req.SourceType),
		SourceLocation:  strings.TrimSpace(req.SourceLocation),
		TextContent:     normalizeTextContent(req.TextContent),
		ScheduleEnabled: boolToInt(req.ScheduleEnabled),
		ScheduleCron:    strings.TrimSpace(req.ScheduleCron),
		Enabled:         true,
		FileURL:         inferFileURL(docID, req.FileName),
		FileType:        strings.TrimPrefix(strings.ToLower(filepath.Ext(req.FileName)), "."),
		FileSize:        req.FileSize,
		ProcessMode:     processMode,
		ChunkStrategy:   chunkStrategy,
		ChunkConfig:     strings.TrimSpace(req.ChunkConfig),
		PipelineID:      strings.TrimSpace(req.PipelineID),
		Status:          "pending",
		CreatedBy:       strings.TrimSpace(operator),
		UpdatedBy:       strings.TrimSpace(operator),
		CreateTime:      now,
		UpdateTime:      now,
	}
	s.documents[docID] = item

	kb.UpdateTime = now
	s.knowledgeBases[kbID] = kb
	if err := s.persistLocked(); err != nil {
		return KnowledgeDocument{}, err
	}
	return item, nil
}

// GetDocument 查询文档详情。
func (s *Service) GetDocument(docID string) (KnowledgeDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.documents[docID]
	if !ok {
		return KnowledgeDocument{}, ErrDocumentNotFound
	}
	item.ChunkCount = s.countChunksByDocLocked(docID)
	return item, nil
}

// UpdateDocument 更新文档信息。
func (s *Service) UpdateDocument(docID string, req KnowledgeDocumentUpdateRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.documents[docID]
	if !ok {
		return ErrDocumentNotFound
	}
	if name := strings.TrimSpace(req.DocName); name != "" {
		item.DocName = name
	}
	if mode := strings.TrimSpace(req.ProcessMode); mode != "" {
		item.ProcessMode = normalizeProcessMode(mode)
	}
	if strategy := strings.TrimSpace(req.ChunkStrategy); strategy != "" {
		item.ChunkStrategy = normalizeChunkStrategy(strategy)
	}
	if chunkConfig := strings.TrimSpace(req.ChunkConfig); chunkConfig != "" {
		item.ChunkConfig = chunkConfig
	}
	if pipelineID := strings.TrimSpace(req.PipelineID); pipelineID != "" {
		item.PipelineID = pipelineID
	}
	if sourceLocation := strings.TrimSpace(req.SourceLocation); sourceLocation != "" {
		item.SourceLocation = sourceLocation
	}
	if req.ScheduleEnabled != nil {
		item.ScheduleEnabled = *req.ScheduleEnabled
	}
	if scheduleCron := strings.TrimSpace(req.ScheduleCron); scheduleCron != "" {
		item.ScheduleCron = scheduleCron
	}
	if updatedBy := strings.TrimSpace(req.UpdatedBy); updatedBy != "" {
		item.UpdatedBy = updatedBy
	}
	item.UpdateTime = s.now()
	s.documents[docID] = item
	if err := s.persistLocked(); err != nil {
		return err
	}
	return nil
}

// StartDocumentChunk 启动文档分块，生成示例 Chunk 和日志。
func (s *Service) StartDocumentChunk(ctx context.Context, docID string) error {
	ctx = normalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.documents[docID]
	if !ok {
		return ErrDocumentNotFound
	}

	startTime := s.now()
	chunkContents := buildChunkContents(item)
	createdChunks := make([]KnowledgeChunk, 0, len(chunkContents))
	for index, content := range chunkContents {
		chunk, err := s.createChunkLocked(ctx, item, KnowledgeChunkCreateRequest{
			Content: content,
			Index:   ptrInt(index),
		})
		if err != nil {
			return err
		}
		createdChunks = append(createdChunks, chunk)
	}

	item.Status = "success"
	item.ChunkCount = s.countChunksByDocLocked(docID)
	item.UpdateTime = s.now()
	s.documents[docID] = item

	log := KnowledgeDocumentChunkLog{
		ID:              "log_" + s.newID(),
		DocID:           docID,
		Status:          "success",
		ProcessMode:     item.ProcessMode,
		ChunkStrategy:   item.ChunkStrategy,
		PipelineID:      item.PipelineID,
		PipelineName:    item.PipelineID,
		ExtractDuration: 80,
		ChunkDuration:   120,
		EmbedDuration:   160,
		PersistDuration: 60,
		OtherDuration:   20,
		TotalDuration:   440,
		ChunkCount:      item.ChunkCount,
		StartTime:       startTime,
		EndTime:         startTime.Add(440 * time.Millisecond),
		CreateTime:      s.now(),
	}
	s.chunkLogs[docID] = append([]KnowledgeDocumentChunkLog{log}, s.chunkLogs[docID]...)
	if err := s.persistLocked(); err != nil {
		return err
	}
	if err := s.upsertChunkVectors(ctx, createdChunks); err != nil {
		return err
	}
	return nil
}

// DeleteDocument 删除文档及其关联 Chunk 和日志。
func (s *Service) DeleteDocument(ctx context.Context, docID string) error {
	ctx = normalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.documents[docID]
	if !ok {
		return ErrDocumentNotFound
	}
	vectorIDs := make([]string, 0)
	delete(s.documents, docID)
	delete(s.chunkLogs, docID)
	for chunkID, chunk := range s.chunks {
		if chunk.DocID == docID {
			vectorIDs = append(vectorIDs, chunk.ID)
			delete(s.chunks, chunkID)
		}
	}
	kb := s.knowledgeBases[item.KBID]
	kb.UpdateTime = s.now()
	s.knowledgeBases[item.KBID] = kb
	if err := s.persistLocked(); err != nil {
		return err
	}
	if err := s.deleteVectors(ctx, vectorIDs); err != nil {
		return err
	}
	return nil
}

// PageDocuments 分页查询文档。
func (s *Service) PageDocuments(kbID string, req KnowledgeDocumentPageRequest) (PageResult[KnowledgeDocument], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.knowledgeBases[kbID]; !ok {
		return PageResult[KnowledgeDocument]{}, ErrKnowledgeBaseNotFound
	}
	current, size := normalizePage(req.Current, req.Size)
	filtered := make([]KnowledgeDocument, 0)
	for _, item := range s.documents {
		if item.KBID != kbID {
			continue
		}
		if status := strings.TrimSpace(req.Status); status != "" && !strings.EqualFold(item.Status, status) {
			continue
		}
		if keyword := strings.TrimSpace(req.Keyword); keyword != "" &&
			!containsFold(item.DocName, keyword) &&
			!containsFold(item.SourceLocation, keyword) {
			continue
		}
		item.ChunkCount = s.countChunksByDocLocked(item.ID)
		filtered = append(filtered, item)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreateTime.After(filtered[j].CreateTime)
	})
	return paginate(filtered, current, size), nil
}

// SearchDocuments 搜索文档。
func (s *Service) SearchDocuments(ctx context.Context, keyword string, limit int) []KnowledgeDocumentSearchItem {
	ctx = normalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil
	}
	if limit <= 0 {
		limit = 8
	}
	keyword = strings.TrimSpace(keyword)
	snapshot := s.snapshotRecords()
	if keyword != "" && s.vectorStore != nil {
		if docs, err := s.searchDocumentsByVector(ctx, snapshot, keyword, limit); err == nil && len(docs) > 0 {
			return docs
		} else if ctxErr := ctx.Err(); ctxErr != nil {
			return nil
		}
	}

	queryTokens := tokenize(keyword)
	var queryEmbedding []float64
	if keyword != "" {
		embedding, err := embedOne(ctx, s.embed, "", keyword)
		if err != nil {
			return nil
		}
		queryEmbedding = embedding
	}

	type scoredDocument struct {
		item  KnowledgeDocumentSearchItem
		score retrievalScore
	}

	filtered := make([]scoredDocument, 0, limit)
	for _, item := range snapshot.documents {
		score := retrievalScore{lexical: 1}
		if keyword != "" {
			score = scoreKnowledgeMatch(queryTokens, queryEmbedding, item.DocName+" "+item.SourceLocation)
			if !score.matched() {
				continue
			}
		}
		filtered = append(filtered, scoredDocument{
			item: KnowledgeDocumentSearchItem{
				ID:      item.ID,
				KBID:    item.KBID,
				DocName: item.DocName,
				KBName:  snapshot.knowledgeBases[item.KBID].Name,
			},
			score: score,
		})
	}
	sort.Slice(filtered, func(i, j int) bool {
		if compareRetrievalScore(filtered[i].score, filtered[j].score) == 0 {
			return filtered[i].item.DocName < filtered[j].item.DocName
		}
		return compareRetrievalScore(filtered[i].score, filtered[j].score) > 0
	})
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	result := make([]KnowledgeDocumentSearchItem, 0, len(filtered))
	for _, scored := range filtered {
		result = append(result, scored.item)
	}
	return result
}

// BuildPromptContext 基于现有 Chunk 构建给大模型使用的检索上下文。
func (s *Service) BuildPromptContext(ctx context.Context, query string, limit int) (string, error) {
	ctx = normalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return "", err
	}
	needle := tokenize(query)
	if len(needle) == 0 {
		return "", nil
	}
	snapshot := s.snapshotRecords()
	if s.vectorStore != nil {
		matches, err := s.searchPromptContextByVector(ctx, snapshot, query, limit)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return "", ctxErr
			}
		} else if len(matches) > 0 {
			return formatPromptContext(matches, limit), nil
		}
	}

	queryEmbedding, err := embedOne(ctx, s.embed, "", query)
	if err != nil {
		return "", err
	}

	matches := make([]promptContextMatch, 0)
	for _, chunk := range snapshot.chunks {
		if chunk.Enabled == 0 {
			continue
		}
		doc, ok := snapshot.documents[chunk.DocID]
		if !ok || !doc.Enabled {
			continue
		}
		score := scoreKnowledgeChunk(needle, queryEmbedding, chunk, doc)
		if !score.matched() {
			continue
		}
		kbName := ""
		if kb, ok := snapshot.knowledgeBases[doc.KBID]; ok {
			kbName = kb.Name
		}
		matches = append(matches, promptContextMatch{
			kbName:     kbName,
			docName:    doc.DocName,
			source:     defaultText(doc.SourceLocation, defaultText(doc.FileURL, doc.DocName)),
			chunkIndex: chunk.ChunkIndex,
			content:    chunk.Content,
			score:      score,
		})
	}

	return formatPromptContext(matches, limit), nil
}

// EnableDocument 启用或禁用文档。
func (s *Service) EnableDocument(docID string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.documents[docID]
	if !ok {
		return ErrDocumentNotFound
	}
	item.Enabled = enabled
	item.UpdateTime = s.now()
	s.documents[docID] = item
	if err := s.persistLocked(); err != nil {
		return err
	}
	return nil
}

// PageChunkLogs 分页查询文档分块日志。
func (s *Service) PageChunkLogs(docID string, current, size int) (PageResult[KnowledgeDocumentChunkLog], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.documents[docID]; !ok {
		return PageResult[KnowledgeDocumentChunkLog]{}, ErrDocumentNotFound
	}
	logs := make([]KnowledgeDocumentChunkLog, len(s.chunkLogs[docID]))
	copy(logs, s.chunkLogs[docID])
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].CreateTime.After(logs[j].CreateTime)
	})
	current, size = normalizePage(current, size)
	return paginate(logs, current, size), nil
}

// PageChunks 分页查询 Chunk。
func (s *Service) PageChunks(docID string, req KnowledgeChunkPageRequest) (PageResult[KnowledgeChunk], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, ok := s.documents[docID]
	if !ok {
		return PageResult[KnowledgeChunk]{}, ErrDocumentNotFound
	}
	current, size := normalizePage(req.Current, req.Size)
	filtered := make([]KnowledgeChunk, 0)
	for _, chunk := range s.chunks {
		if chunk.DocID != docID {
			continue
		}
		if req.Enabled != nil && chunk.Enabled != *req.Enabled {
			continue
		}
		chunk.KBID = doc.KBID
		filtered = append(filtered, chunk)
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].ChunkIndex == filtered[j].ChunkIndex {
			return filtered[i].CreateTime.Before(filtered[j].CreateTime)
		}
		return filtered[i].ChunkIndex < filtered[j].ChunkIndex
	})
	return paginate(filtered, current, size), nil
}

// CreateChunk 创建新的 Chunk。
func (s *Service) CreateChunk(ctx context.Context, docID string, req KnowledgeChunkCreateRequest) (KnowledgeChunk, error) {
	ctx = normalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return KnowledgeChunk{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	doc, ok := s.documents[docID]
	if !ok {
		return KnowledgeChunk{}, ErrDocumentNotFound
	}
	chunk, err := s.createChunkLocked(ctx, doc, req)
	if err != nil {
		return KnowledgeChunk{}, err
	}
	if err := s.persistLocked(); err != nil {
		return KnowledgeChunk{}, err
	}
	if err := s.upsertChunkVectors(ctx, []KnowledgeChunk{chunk}); err != nil {
		return KnowledgeChunk{}, err
	}
	return chunk, nil
}

// UpdateChunk 更新 Chunk 内容。
func (s *Service) UpdateChunk(ctx context.Context, docID, chunkID string, req KnowledgeChunkUpdateRequest) error {
	ctx = normalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		return ErrChunkContentRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	doc, ok := s.documents[docID]
	if !ok {
		return ErrDocumentNotFound
	}
	chunk, ok := s.chunks[chunkID]
	if !ok || chunk.DocID != docID {
		return ErrChunkNotFound
	}
	chunk.Content = content
	chunk.ContentHash = buildContentHash(content)
	chunk.CharCount = countChars(content)
	chunk.TokenCount = estimateTokens(content)
	chunk.KBID = doc.KBID
	chunk.EmbeddingModel = s.embeddingModelForDocLocked(doc)
	embedding, err := embedOne(ctx, s.embed, chunk.EmbeddingModel, content)
	if err != nil {
		return err
	}
	chunk.Embedding = embedding
	chunk.UpdateTime = s.now()
	s.chunks[chunkID] = chunk
	if err := s.persistLocked(); err != nil {
		return err
	}
	if err := s.upsertChunkVectors(ctx, []KnowledgeChunk{chunk}); err != nil {
		return err
	}
	return nil
}

// DeleteChunk 删除指定 Chunk。
func (s *Service) DeleteChunk(ctx context.Context, docID, chunkID string) error {
	ctx = normalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.documents[docID]; !ok {
		return ErrDocumentNotFound
	}
	chunk, ok := s.chunks[chunkID]
	if !ok || chunk.DocID != docID {
		return ErrChunkNotFound
	}
	delete(s.chunks, chunkID)
	s.updateDocumentChunkCountLocked(docID)
	if err := s.persistLocked(); err != nil {
		return err
	}
	if err := s.deleteVectors(ctx, []string{chunk.ID}); err != nil {
		return err
	}
	return nil
}

// ToggleChunk 启用或禁用单个 Chunk。
func (s *Service) ToggleChunk(docID, chunkID string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.documents[docID]; !ok {
		return ErrDocumentNotFound
	}
	chunk, ok := s.chunks[chunkID]
	if !ok || chunk.DocID != docID {
		return ErrChunkNotFound
	}
	chunk.Enabled = boolToInt(enabled)
	chunk.UpdateTime = s.now()
	s.chunks[chunkID] = chunk
	if err := s.persistLocked(); err != nil {
		return err
	}
	return nil
}

// BatchToggleChunks 批量启用或禁用 Chunk。
func (s *Service) BatchToggleChunks(docID string, chunkIDs []string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.documents[docID]; !ok {
		return ErrDocumentNotFound
	}
	targetAll := len(chunkIDs) == 0
	targetSet := make(map[string]struct{}, len(chunkIDs))
	for _, id := range chunkIDs {
		targetSet[id] = struct{}{}
	}
	for id, chunk := range s.chunks {
		if chunk.DocID != docID {
			continue
		}
		if !targetAll {
			if _, ok := targetSet[id]; !ok {
				continue
			}
		}
		chunk.Enabled = boolToInt(enabled)
		chunk.UpdateTime = s.now()
		s.chunks[id] = chunk
	}
	if err := s.persistLocked(); err != nil {
		return err
	}
	return nil
}

func (s *Service) createChunkLocked(ctx context.Context, doc KnowledgeDocument, req KnowledgeChunkCreateRequest) (KnowledgeChunk, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return KnowledgeChunk{}, ErrChunkContentRequired
	}
	now := s.now()
	chunkID := strings.TrimSpace(req.ChunkID)
	if chunkID == "" {
		chunkID = "chunk_" + s.newID()
	}
	index := s.nextChunkIndexLocked(doc.ID)
	if req.Index != nil {
		index = *req.Index
	}
	embeddingModel := s.embeddingModelForDocLocked(doc)
	embedding, err := embedOne(ctx, s.embed, embeddingModel, content)
	if err != nil {
		return KnowledgeChunk{}, err
	}
	chunk := KnowledgeChunk{
		ID:             chunkID,
		KBID:           doc.KBID,
		DocID:          doc.ID,
		ChunkIndex:     index,
		Content:        content,
		ContentHash:    buildContentHash(content),
		CharCount:      countChars(content),
		TokenCount:     estimateTokens(content),
		EmbeddingModel: embeddingModel,
		Embedding:      embedding,
		Enabled:        1,
		CreateTime:     now,
		UpdateTime:     now,
	}
	s.chunks[chunk.ID] = chunk
	s.updateDocumentChunkCountLocked(doc.ID)
	return chunk, nil
}

func (s *Service) embeddingModelForDocLocked(doc KnowledgeDocument) string {
	if kb, ok := s.knowledgeBases[doc.KBID]; ok {
		return strings.TrimSpace(kb.EmbeddingModel)
	}
	return ""
}

func (s *Service) upsertChunkVectors(ctx context.Context, chunks []KnowledgeChunk) error {
	if s.vectorStore == nil || len(chunks) == 0 {
		return nil
	}
	vectors := make([]KnowledgeVector, 0, len(chunks))
	for _, chunk := range chunks {
		doc, ok := s.documents[chunk.DocID]
		if !ok {
			continue
		}
		kb, ok := s.knowledgeBases[doc.KBID]
		if !ok {
			continue
		}
		vectors = append(vectors, KnowledgeVector{
			ID:             chunk.ID,
			KBID:           doc.KBID,
			DocID:          doc.ID,
			CollectionName: strings.TrimSpace(kb.CollectionName),
			ChunkID:        chunk.ID,
			ChunkIndex:     chunk.ChunkIndex,
			Content:        chunk.Content,
			EmbeddingModel: strings.TrimSpace(chunk.EmbeddingModel),
			Embedding:      cloneFloat64Slice(chunk.Embedding),
		})
	}
	if len(vectors) == 0 {
		return nil
	}
	return s.vectorStore.Upsert(ctx, vectors)
}

func (s *Service) deleteVectors(ctx context.Context, ids []string) error {
	if s.vectorStore == nil || len(ids) == 0 {
		return nil
	}
	uniqueIDs := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}
	if len(uniqueIDs) == 0 {
		return nil
	}
	return s.vectorStore.Delete(ctx, uniqueIDs)
}

func (s *Service) snapshotRecords() knowledgeSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := knowledgeSnapshot{
		knowledgeBases: make(map[string]KnowledgeBase, len(s.knowledgeBases)),
		documents:      make(map[string]KnowledgeDocument, len(s.documents)),
		chunks:         make(map[string]KnowledgeChunk, len(s.chunks)),
	}
	for id, item := range s.knowledgeBases {
		snapshot.knowledgeBases[id] = item
	}
	for id, item := range s.documents {
		snapshot.documents[id] = item
	}
	for id, item := range s.chunks {
		item.Embedding = cloneFloat64Slice(item.Embedding)
		snapshot.chunks[id] = item
	}
	return snapshot
}

func (s *Service) searchPromptContextByVector(ctx context.Context, snapshot knowledgeSnapshot, query string, limit int) ([]promptContextMatch, error) {
	results, err := s.searchVectors(ctx, snapshot, query, limit)
	if err != nil {
		return nil, err
	}
	matches := make([]promptContextMatch, 0, len(results))
	seenChunks := make(map[string]struct{}, len(results))
	for _, result := range results {
		chunkID := defaultText(result.Vector.ChunkID, result.Vector.ID)
		if _, ok := seenChunks[chunkID]; ok {
			continue
		}
		chunk, doc, kb, ok := snapshot.vectorResultRecords(result)
		if !ok {
			continue
		}
		seenChunks[chunkID] = struct{}{}
		matches = append(matches, promptContextMatch{
			kbName:     kb.Name,
			docName:    doc.DocName,
			source:     defaultText(doc.SourceLocation, defaultText(doc.FileURL, doc.DocName)),
			chunkIndex: chunk.ChunkIndex,
			content:    chunk.Content,
			score: retrievalScore{
				lexical: 1,
				vector:  result.Score,
			},
		})
	}
	sortPromptContextMatches(matches)
	return limitPromptContextMatches(matches, limit), nil
}

func (s *Service) searchDocumentsByVector(ctx context.Context, snapshot knowledgeSnapshot, keyword string, limit int) ([]KnowledgeDocumentSearchItem, error) {
	results, err := s.searchVectors(ctx, snapshot, keyword, limit)
	if err != nil {
		return nil, err
	}
	type scoredDocument struct {
		item  KnowledgeDocumentSearchItem
		score retrievalScore
	}
	byDocID := make(map[string]scoredDocument, len(results))
	for _, result := range results {
		_, doc, kb, ok := snapshot.vectorResultRecords(result)
		if !ok {
			continue
		}
		scored := scoredDocument{
			item: KnowledgeDocumentSearchItem{
				ID:      doc.ID,
				KBID:    doc.KBID,
				DocName: doc.DocName,
				KBName:  kb.Name,
			},
			score: retrievalScore{
				lexical: 1,
				vector:  result.Score,
			},
		}
		current, exists := byDocID[doc.ID]
		if !exists || compareRetrievalScore(scored.score, current.score) > 0 {
			byDocID[doc.ID] = scored
		}
	}
	documents := make([]scoredDocument, 0, len(byDocID))
	for _, item := range byDocID {
		documents = append(documents, item)
	}
	sort.Slice(documents, func(i, j int) bool {
		if compareRetrievalScore(documents[i].score, documents[j].score) == 0 {
			return documents[i].item.DocName < documents[j].item.DocName
		}
		return compareRetrievalScore(documents[i].score, documents[j].score) > 0
	})
	if len(documents) > limit {
		documents = documents[:limit]
	}
	result := make([]KnowledgeDocumentSearchItem, 0, len(documents))
	for _, item := range documents {
		result = append(result, item.item)
	}
	return result, nil
}

func (s *Service) searchVectors(ctx context.Context, snapshot knowledgeSnapshot, query string, limit int) ([]VectorSearchResult, error) {
	if s.vectorStore == nil {
		return nil, nil
	}
	groups := make(map[string]KnowledgeBase, len(snapshot.knowledgeBases))
	for _, kb := range snapshot.knowledgeBases {
		collectionName := strings.TrimSpace(kb.CollectionName)
		if collectionName == "" {
			continue
		}
		embeddingModel := strings.TrimSpace(kb.EmbeddingModel)
		key := collectionName + "\x00" + embeddingModel
		if _, ok := groups[key]; ok {
			continue
		}
		kb.CollectionName = collectionName
		kb.EmbeddingModel = embeddingModel
		groups[key] = kb
	}
	if len(groups) == 0 {
		return nil, nil
	}

	searchLimit := vectorSearchLimit(limit)
	results := make([]VectorSearchResult, 0)
	for _, kb := range groups {
		embedding, err := embedOne(ctx, s.embed, kb.EmbeddingModel, query)
		if err != nil {
			return nil, err
		}
		found, err := s.vectorStore.Search(ctx, VectorSearchRequest{
			Query:          query,
			CollectionName: kb.CollectionName,
			EmbeddingModel: kb.EmbeddingModel,
			Embedding:      embedding,
			Limit:          searchLimit,
		})
		if err != nil {
			return nil, err
		}
		results = append(results, found...)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			leftID := defaultText(results[i].Vector.ChunkID, results[i].Vector.ID)
			rightID := defaultText(results[j].Vector.ChunkID, results[j].Vector.ID)
			return leftID < rightID
		}
		return results[i].Score > results[j].Score
	})
	return results, nil
}

func (snapshot knowledgeSnapshot) vectorResultRecords(result VectorSearchResult) (KnowledgeChunk, KnowledgeDocument, KnowledgeBase, bool) {
	chunkID := defaultText(result.Vector.ChunkID, result.Vector.ID)
	chunk, ok := snapshot.chunks[chunkID]
	if !ok || chunk.Enabled == 0 {
		return KnowledgeChunk{}, KnowledgeDocument{}, KnowledgeBase{}, false
	}
	doc, ok := snapshot.documents[chunk.DocID]
	if !ok || !doc.Enabled {
		return KnowledgeChunk{}, KnowledgeDocument{}, KnowledgeBase{}, false
	}
	kb, ok := snapshot.knowledgeBases[doc.KBID]
	if !ok {
		return KnowledgeChunk{}, KnowledgeDocument{}, KnowledgeBase{}, false
	}
	return chunk, doc, kb, true
}

func formatPromptContext(matches []promptContextMatch, limit int) string {
	sortPromptContextMatches(matches)
	matches = limitPromptContextMatches(matches, limit)
	if len(matches) == 0 {
		return ""
	}

	parts := make([]string, 0, len(matches))
	for index, item := range matches {
		// 给大模型保留来源信息，便于生成可追溯的回答。
		lines := []string{
			"[" + strconv.Itoa(index+1) + "] 知识库：" + defaultText(item.kbName, "未命名知识库"),
			"文档：" + item.docName,
			"来源：" + defaultText(item.source, item.docName),
			"Chunk：" + strconv.Itoa(item.chunkIndex),
			item.content,
		}
		parts = append(parts, strings.Join(lines, "\n"))
	}
	return strings.Join(parts, "\n\n")
}

func sortPromptContextMatches(matches []promptContextMatch) {
	sort.Slice(matches, func(i, j int) bool {
		if compareRetrievalScore(matches[i].score, matches[j].score) == 0 {
			if matches[i].docName == matches[j].docName {
				return matches[i].chunkIndex < matches[j].chunkIndex
			}
			return matches[i].docName < matches[j].docName
		}
		return compareRetrievalScore(matches[i].score, matches[j].score) > 0
	})
}

func limitPromptContextMatches(matches []promptContextMatch, limit int) []promptContextMatch {
	if limit <= 0 {
		limit = 4
	}
	if len(matches) > limit {
		return matches[:limit]
	}
	return matches
}

func vectorSearchLimit(limit int) int {
	if limit <= 0 {
		limit = 4
	}
	if limit < 4 {
		limit = 4
	}
	return limit * 4
}

func (s *Service) countDocumentsByKBLocked(kbID string) int {
	count := 0
	for _, doc := range s.documents {
		if doc.KBID == kbID {
			count++
		}
	}
	return count
}

func (s *Service) countChunksByDocLocked(docID string) int {
	count := 0
	for _, chunk := range s.chunks {
		if chunk.DocID == docID {
			count++
		}
	}
	return count
}

func (s *Service) nextChunkIndexLocked(docID string) int {
	maxIndex := -1
	for _, chunk := range s.chunks {
		if chunk.DocID == docID && chunk.ChunkIndex > maxIndex {
			maxIndex = chunk.ChunkIndex
		}
	}
	return maxIndex + 1
}

func (s *Service) updateDocumentChunkCountLocked(docID string) {
	doc := s.documents[docID]
	doc.ChunkCount = s.countChunksByDocLocked(docID)
	doc.UpdateTime = s.now()
	s.documents[docID] = doc
}

func normalizePage(current, size int) (int, int) {
	if current <= 0 {
		current = 1
	}
	if size <= 0 {
		size = 10
	}
	return current, size
}

func paginate[T any](records []T, current, size int) PageResult[T] {
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
	result := make([]T, end-start)
	copy(result, records[start:end])
	return PageResult[T]{
		Records: result,
		Total:   total,
		Size:    size,
		Current: current,
		Pages:   pages,
	}
}

func containsFold(value, keyword string) bool {
	needle := strings.ToLower(strings.TrimSpace(keyword))
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(value), needle)
}

func normalizeSourceType(sourceType string) string {
	value := strings.ToLower(strings.TrimSpace(sourceType))
	if value == "url" {
		return "url"
	}
	return "file"
}

func normalizeProcessMode(processMode string) string {
	value := strings.ToLower(strings.TrimSpace(processMode))
	if value == "pipeline" {
		return "pipeline"
	}
	return "chunk"
}

func normalizeChunkStrategy(strategy string) string {
	value := strings.ToLower(strings.TrimSpace(strategy))
	if value == "" {
		return "structure_aware"
	}
	return value
}

func inferDocName(req KnowledgeDocumentUploadRequest) string {
	if name := strings.TrimSpace(req.FileName); name != "" {
		return name
	}
	if location := strings.TrimSpace(req.SourceLocation); location != "" {
		parts := strings.Split(strings.TrimRight(location, "/"), "/")
		return parts[len(parts)-1]
	}
	return ""
}

func inferFileURL(docID, fileName string) string {
	if strings.TrimSpace(fileName) == "" {
		return ""
	}
	return "memory://" + docID + "/" + fileName
}

func normalizeTextContent(content string) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

func buildChunkContents(doc KnowledgeDocument) []string {
	if text := strings.TrimSpace(doc.TextContent); text != "" {
		return splitTextIntoChunks(text, doc.ChunkStrategy)
	}
	return []string{
		doc.DocName + " - 章节摘要",
		"来源：" + defaultText(doc.SourceLocation, doc.DocName),
		"当前处理模式：" + defaultText(doc.ProcessMode, "chunk"),
	}
}

func splitTextIntoChunks(content, strategy string) []string {
	paragraphs := strings.Split(normalizeTextContent(content), "\n")
	chunks := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		for _, chunk := range splitLongParagraph(paragraph, chunkTargetSize(strategy)) {
			chunks = append(chunks, chunk)
		}
	}
	if len(chunks) == 0 && strings.TrimSpace(content) != "" {
		chunks = append(chunks, strings.TrimSpace(content))
	}
	return chunks
}

func splitLongParagraph(paragraph string, targetSize int) []string {
	runes := []rune(strings.TrimSpace(paragraph))
	if len(runes) == 0 {
		return nil
	}
	if targetSize <= 0 || len(runes) <= targetSize {
		return []string{string(runes)}
	}
	result := make([]string, 0, (len(runes)+targetSize-1)/targetSize)
	for start := 0; start < len(runes); start += targetSize {
		end := start + targetSize
		if end > len(runes) {
			end = len(runes)
		}
		result = append(result, string(runes[start:end]))
	}
	return result
}

func chunkTargetSize(strategy string) int {
	if strings.EqualFold(strings.TrimSpace(strategy), "fixed_size") {
		return 512
	}
	return 1400
}

func countChars(content string) int {
	return len([]rune(content))
}

func estimateTokens(content string) int {
	charCount := countChars(content)
	if charCount == 0 {
		return 0
	}
	return (charCount + 3) / 4
}

func buildContentHash(content string) string {
	value := strings.TrimSpace(content)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) > 24 {
		return string(runes[:24])
	}
	return value
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func defaultText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func ptrInt(value int) *int {
	return &value
}

func tokenize(text string) []string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	fields := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t' || r == '，' || r == '。' || r == '、' || r == ',' || r == '.' || r == ':' || r == '：'
	})
	result := make([]string, 0, len(fields)+16)
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			appendToken(&result, field)
			appendNGrams(&result, []rune(field), 2)
			appendNGrams(&result, []rune(field), 3)
			appendBusinessSynonyms(&result, field)
		}
	}
	if len(result) == 0 && normalized != "" {
		appendToken(&result, normalized)
		appendNGrams(&result, []rune(normalized), 2)
		appendNGrams(&result, []rune(normalized), 3)
		appendBusinessSynonyms(&result, normalized)
	}
	return result
}

func overlapScore(a, b []string) int {
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

func appendNGrams(target *[]string, runes []rune, size int) {
	if len(runes) < size || size <= 0 {
		return
	}
	for i := 0; i <= len(runes)-size; i++ {
		*target = append(*target, string(runes[i:i+size]))
	}
}

func appendBusinessSynonyms(target *[]string, field string) {
	for source, synonyms := range businessSynonyms {
		if !strings.Contains(field, source) {
			continue
		}
		for _, synonym := range synonyms {
			// 本地同义词只做召回增强，不替换原始内容。
			appendToken(target, synonym)
			appendNGrams(target, []rune(synonym), 2)
			appendNGrams(target, []rune(synonym), 3)
		}
	}
}

func appendToken(target *[]string, token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}
	*target = append(*target, token)
}

func (s *Service) persistLocked() error {
	if s.repo == nil {
		return nil
	}
	bases := make([]KnowledgeBase, 0, len(s.knowledgeBases))
	for _, item := range s.knowledgeBases {
		bases = append(bases, item)
	}
	docs := make([]KnowledgeDocument, 0, len(s.documents))
	for _, item := range s.documents {
		docs = append(docs, item)
	}
	chunks := make([]KnowledgeChunk, 0, len(s.chunks))
	for _, item := range s.chunks {
		chunks = append(chunks, item)
	}
	logs := make([]KnowledgeDocumentChunkLog, 0)
	for _, docLogs := range s.chunkLogs {
		for _, item := range docLogs {
			logs = append(logs, item)
		}
	}
	sort.Slice(bases, func(i, j int) bool { return bases[i].CreateTime.After(bases[j].CreateTime) })
	sort.Slice(docs, func(i, j int) bool { return docs[i].CreateTime.After(docs[j].CreateTime) })
	sort.Slice(chunks, func(i, j int) bool {
		if chunks[i].DocID == chunks[j].DocID {
			return chunks[i].ChunkIndex < chunks[j].ChunkIndex
		}
		return chunks[i].DocID < chunks[j].DocID
	})
	sort.Slice(logs, func(i, j int) bool {
		if logs[i].DocID == logs[j].DocID {
			return logs[i].CreateTime.After(logs[j].CreateTime)
		}
		return logs[i].DocID < logs[j].DocID
	})
	return s.repo.SaveKnowledgeRecords(bases, docs, chunks, logs)
}
