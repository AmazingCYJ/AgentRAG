package ingestion

import (
	"errors"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	platformstate "github.com/AmazingCYJ/AgentRAG/internal/platform/state"
	"github.com/gogf/gf/v2/util/guid"
)

var (
	// ErrPipelineNotFound 表示流水线不存在。
	ErrPipelineNotFound = errors.New("流水线不存在")
	// ErrPipelineNameRequired 表示流水线名称不能为空。
	ErrPipelineNameRequired = errors.New("流水线名称不能为空")
	// ErrTaskNotFound 表示任务不存在。
	ErrTaskNotFound = errors.New("任务不存在")
	// ErrPipelineIDRequired 表示流水线ID不能为空。
	ErrPipelineIDRequired = errors.New("流水线ID不能为空")
)

// PageResult 定义统一分页结构。
type PageResult[T any] struct {
	Records []T `json:"records"`
	Total   int `json:"total"`
	Size    int `json:"size"`
	Current int `json:"current"`
	Pages   int `json:"pages"`
}

// PipelineNode 表示流水线节点。
type PipelineNode struct {
	ID         int64          `json:"id"`
	NodeID     string         `json:"nodeId"`
	NodeType   string         `json:"nodeType"`
	Settings   map[string]any `json:"settings,omitempty"`
	Condition  map[string]any `json:"condition,omitempty"`
	NextNodeID string         `json:"nextNodeId,omitempty"`
}

// Pipeline 表示流水线对象。
type Pipeline struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	CreatedBy   string         `json:"createdBy,omitempty"`
	Nodes       []PipelineNode `json:"nodes,omitempty"`
	CreateTime  time.Time      `json:"createTime,omitempty"`
	UpdateTime  time.Time      `json:"updateTime,omitempty"`
}

// TaskLog 表示任务节点日志摘要。
type TaskLog struct {
	NodeID     string `json:"nodeId"`
	NodeType   string `json:"nodeType"`
	Message    string `json:"message,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// Task 表示摄入任务对象。
type Task struct {
	ID             string         `json:"id"`
	PipelineID     string         `json:"pipelineId"`
	SourceType     string         `json:"sourceType,omitempty"`
	SourceLocation string         `json:"sourceLocation,omitempty"`
	SourceFileName string         `json:"sourceFileName,omitempty"`
	Status         string         `json:"status,omitempty"`
	ChunkCount     int            `json:"chunkCount,omitempty"`
	ErrorMessage   string         `json:"errorMessage,omitempty"`
	Logs           []TaskLog      `json:"logs,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	StartedAt      time.Time      `json:"startedAt,omitempty"`
	CompletedAt    time.Time      `json:"completedAt,omitempty"`
	CreatedBy      string         `json:"createdBy,omitempty"`
	CreateTime     time.Time      `json:"createTime,omitempty"`
	UpdateTime     time.Time      `json:"updateTime,omitempty"`
}

// TaskNode 表示任务节点执行详情。
type TaskNode struct {
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

// IngestionResult 定义任务触发结果。
type IngestionResult struct {
	TaskID     string `json:"taskId"`
	PipelineID string `json:"pipelineId"`
	Status     string `json:"status,omitempty"`
	ChunkCount int    `json:"chunkCount,omitempty"`
	Message    string `json:"message,omitempty"`
}

// PipelineNodeRequest 定义流水线节点输入。
type PipelineNodeRequest struct {
	NodeID     string         `json:"nodeId"`
	NodeType   string         `json:"nodeType"`
	Settings   map[string]any `json:"settings"`
	Condition  map[string]any `json:"condition"`
	NextNodeID string         `json:"nextNodeId"`
}

// PipelineSaveRequest 定义流水线创建/更新输入。
type PipelineSaveRequest struct {
	Name        string
	Description string
	Nodes       []PipelineNodeRequest
	CreatedBy   string
}

// TaskSource 定义任务数据源。
type TaskSource struct {
	Type        string         `json:"type"`
	Location    string         `json:"location"`
	FileName    string         `json:"fileName,omitempty"`
	Credentials map[string]any `json:"credentials,omitempty"`
}

// TaskCreateRequest 定义任务创建输入。
type TaskCreateRequest struct {
	PipelineID    string
	Source        TaskSource
	Metadata      map[string]any
	VectorSpaceID map[string]any
	CreatedBy     string
}

// UploadTaskRequest 定义上传文件触发任务的输入。
type UploadTaskRequest struct {
	PipelineID string
	FileName   string
	FileSize   int64
	Content    []byte
	CreatedBy  string
}

// Repository 定义导入流水线、任务和任务节点的持久化仓储。
type Repository interface {
	LoadIngestionRecords() ([]Pipeline, []Task, []TaskNode, error)
	SaveIngestionRecords(pipelines []Pipeline, tasks []Task, taskNodes []TaskNode) error
}

type fileStoreRepository struct {
	store *platformstate.FileStore
}

func (r *fileStoreRepository) LoadIngestionRecords() ([]Pipeline, []Task, []TaskNode, error) {
	if r == nil || r.store == nil {
		return nil, nil, nil, nil
	}
	snapshot, err := r.store.Load()
	if err != nil {
		return nil, nil, nil, err
	}
	pipelines := make([]Pipeline, 0, len(snapshot.IngestionPipelines))
	for _, item := range snapshot.IngestionPipelines {
		nodes := make([]PipelineNode, 0, len(item.Nodes))
		for _, node := range item.Nodes {
			nodes = append(nodes, PipelineNode{
				ID:         node.ID,
				NodeID:     node.NodeID,
				NodeType:   node.NodeType,
				Settings:   cloneMap(node.Settings),
				Condition:  cloneMap(node.Condition),
				NextNodeID: node.NextNodeID,
			})
		}
		pipelines = append(pipelines, Pipeline{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			CreatedBy:   item.CreatedBy,
			Nodes:       nodes,
			CreateTime:  item.CreateTime,
			UpdateTime:  item.UpdateTime,
		})
	}
	tasks := make([]Task, 0, len(snapshot.IngestionTasks))
	for _, item := range snapshot.IngestionTasks {
		logs := make([]TaskLog, 0, len(item.Logs))
		for _, log := range item.Logs {
			logs = append(logs, TaskLog{
				NodeID:     log.NodeID,
				NodeType:   log.NodeType,
				Message:    log.Message,
				DurationMs: log.DurationMs,
				Success:    log.Success,
				Error:      log.Error,
			})
		}
		tasks = append(tasks, Task{
			ID:             item.ID,
			PipelineID:     item.PipelineID,
			SourceType:     item.SourceType,
			SourceLocation: item.SourceLocation,
			SourceFileName: item.SourceFileName,
			Status:         item.Status,
			ChunkCount:     item.ChunkCount,
			ErrorMessage:   item.ErrorMessage,
			Logs:           logs,
			Metadata:       cloneMap(item.Metadata),
			StartedAt:      item.StartedAt,
			CompletedAt:    item.CompletedAt,
			CreatedBy:      item.CreatedBy,
			CreateTime:     item.CreateTime,
			UpdateTime:     item.UpdateTime,
		})
	}
	taskNodes := make([]TaskNode, 0, len(snapshot.IngestionTaskNodes))
	for _, item := range snapshot.IngestionTaskNodes {
		taskNodes = append(taskNodes, TaskNode{
			ID:           item.ID,
			TaskID:       item.TaskID,
			PipelineID:   item.PipelineID,
			NodeID:       item.NodeID,
			NodeType:     item.NodeType,
			NodeOrder:    item.NodeOrder,
			Status:       item.Status,
			DurationMs:   item.DurationMs,
			Message:      item.Message,
			ErrorMessage: item.ErrorMessage,
			Output:       cloneMap(item.Output),
			CreateTime:   item.CreateTime,
			UpdateTime:   item.UpdateTime,
		})
	}
	return pipelines, tasks, taskNodes, nil
}

func (r *fileStoreRepository) SaveIngestionRecords(pipelines []Pipeline, tasks []Task, taskNodes []TaskNode) error {
	if r == nil || r.store == nil {
		return nil
	}
	pipelineRecords := make([]platformstate.IngestionPipelineRecord, 0, len(pipelines))
	for _, pipeline := range pipelines {
		nodeRecords := make([]platformstate.IngestionPipelineNodeRecord, 0, len(pipeline.Nodes))
		for _, node := range pipeline.Nodes {
			nodeRecords = append(nodeRecords, platformstate.IngestionPipelineNodeRecord{
				ID:         node.ID,
				NodeID:     node.NodeID,
				NodeType:   node.NodeType,
				Settings:   cloneMap(node.Settings),
				Condition:  cloneMap(node.Condition),
				NextNodeID: node.NextNodeID,
			})
		}
		pipelineRecords = append(pipelineRecords, platformstate.IngestionPipelineRecord{
			ID:          pipeline.ID,
			Name:        pipeline.Name,
			Description: pipeline.Description,
			CreatedBy:   pipeline.CreatedBy,
			Nodes:       nodeRecords,
			CreateTime:  pipeline.CreateTime,
			UpdateTime:  pipeline.UpdateTime,
		})
	}
	taskRecords := make([]platformstate.IngestionTaskRecord, 0, len(tasks))
	for _, task := range tasks {
		logRecords := make([]platformstate.IngestionTaskLogRecord, 0, len(task.Logs))
		for _, log := range task.Logs {
			logRecords = append(logRecords, platformstate.IngestionTaskLogRecord{
				NodeID:     log.NodeID,
				NodeType:   log.NodeType,
				Message:    log.Message,
				DurationMs: log.DurationMs,
				Success:    log.Success,
				Error:      log.Error,
			})
		}
		taskRecords = append(taskRecords, platformstate.IngestionTaskRecord{
			ID:             task.ID,
			PipelineID:     task.PipelineID,
			SourceType:     task.SourceType,
			SourceLocation: task.SourceLocation,
			SourceFileName: task.SourceFileName,
			Status:         task.Status,
			ChunkCount:     task.ChunkCount,
			ErrorMessage:   task.ErrorMessage,
			Logs:           logRecords,
			Metadata:       cloneMap(task.Metadata),
			StartedAt:      task.StartedAt,
			CompletedAt:    task.CompletedAt,
			CreatedBy:      task.CreatedBy,
			CreateTime:     task.CreateTime,
			UpdateTime:     task.UpdateTime,
		})
	}
	taskNodeRecords := make([]platformstate.IngestionTaskNodeRecord, 0, len(taskNodes))
	for _, node := range taskNodes {
		taskNodeRecords = append(taskNodeRecords, platformstate.IngestionTaskNodeRecord{
			ID:           node.ID,
			TaskID:       node.TaskID,
			PipelineID:   node.PipelineID,
			NodeID:       node.NodeID,
			NodeType:     node.NodeType,
			NodeOrder:    node.NodeOrder,
			Status:       node.Status,
			DurationMs:   node.DurationMs,
			Message:      node.Message,
			ErrorMessage: node.ErrorMessage,
			Output:       cloneMap(node.Output),
			CreateTime:   node.CreateTime,
			UpdateTime:   node.UpdateTime,
		})
	}
	return r.store.Update(func(snapshot *platformstate.Snapshot) {
		snapshot.IngestionPipelines = pipelineRecords
		snapshot.IngestionTasks = taskRecords
		snapshot.IngestionTaskNodes = taskNodeRecords
	})
}

// Service 提供数据通道流水线和任务的内存能力。
type Service struct {
	mu sync.RWMutex

	pipelines map[string]Pipeline
	tasks     map[string]Task
	taskNodes map[string][]TaskNode

	nextPipelineNodeID int64
	now                func() time.Time
	newID              func() string
	repo               Repository
}

// NewService 创建 Ingestion 服务。
func NewService(store *platformstate.FileStore) *Service {
	var repo Repository
	if store != nil {
		repo = &fileStoreRepository{store: store}
	}
	return NewServiceWithRepository(repo)
}

// NewServiceWithRepository 创建基于指定仓储的 Ingestion 服务。
func NewServiceWithRepository(repo Repository) *Service {
	service := &Service{
		pipelines:          make(map[string]Pipeline),
		tasks:              make(map[string]Task),
		taskNodes:          make(map[string][]TaskNode),
		nextPipelineNodeID: 1,
		now:                time.Now,
		newID:              func() string { return strings.ReplaceAll(guid.S(), "-", "") },
		repo:               repo,
	}
	if pipelines, tasks, taskNodes, err := service.loadRecords(); err == nil {
		for _, item := range pipelines {
			for _, node := range item.Nodes {
				service.bumpNextPipelineNodeID(node.ID)
			}
			service.pipelines[item.ID] = item
		}
		for _, item := range tasks {
			service.tasks[item.ID] = item
		}
		for _, item := range taskNodes {
			service.taskNodes[item.TaskID] = append(service.taskNodes[item.TaskID], item)
		}
	}
	return service
}

func (s *Service) loadRecords() ([]Pipeline, []Task, []TaskNode, error) {
	if s.repo == nil {
		return nil, nil, nil, nil
	}
	return s.repo.LoadIngestionRecords()
}

func (s *Service) bumpNextPipelineNodeID(id int64) {
	if id >= s.nextPipelineNodeID {
		s.nextPipelineNodeID = id + 1
	}
}

// CreatePipeline 创建流水线。
func (s *Service) CreatePipeline(req PipelineSaveRequest) (Pipeline, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Pipeline{}, ErrPipelineNameRequired
	}

	now := s.now()
	pipeline := Pipeline{
		ID:          "pipe_" + s.newID(),
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		CreatedBy:   strings.TrimSpace(req.CreatedBy),
		CreateTime:  now,
		UpdateTime:  now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pipeline.Nodes = s.buildPipelineNodesLocked(req.Nodes)
	s.pipelines[pipeline.ID] = pipeline
	if err := s.persistLocked(); err != nil {
		return Pipeline{}, err
	}
	return pipeline, nil
}

// UpdatePipeline 更新流水线。
func (s *Service) UpdatePipeline(id string, req PipelineSaveRequest) (Pipeline, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Pipeline{}, ErrPipelineNameRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pipeline, ok := s.pipelines[id]
	if !ok {
		return Pipeline{}, ErrPipelineNotFound
	}
	pipeline.Name = name
	pipeline.Description = strings.TrimSpace(req.Description)
	pipeline.Nodes = s.buildPipelineNodesLocked(req.Nodes)
	pipeline.UpdateTime = s.now()
	s.pipelines[id] = pipeline
	if err := s.persistLocked(); err != nil {
		return Pipeline{}, err
	}
	return pipeline, nil
}

// GetPipeline 获取流水线详情。
func (s *Service) GetPipeline(id string) (Pipeline, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pipeline, ok := s.pipelines[id]
	if !ok {
		return Pipeline{}, ErrPipelineNotFound
	}
	return clonePipeline(pipeline), nil
}

// PagePipelines 分页查询流水线。
func (s *Service) PagePipelines(pageNo, pageSize int, keyword string) PageResult[Pipeline] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	current, size := normalizePage(pageNo, pageSize)
	records := make([]Pipeline, 0, len(s.pipelines))
	for _, pipeline := range s.pipelines {
		if !containsFold(pipeline.Name, keyword) && !containsFold(pipeline.Description, keyword) {
			continue
		}
		records = append(records, clonePipeline(pipeline))
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreateTime.After(records[j].CreateTime)
	})
	return paginate(records, current, size)
}

// DeletePipeline 删除流水线。
func (s *Service) DeletePipeline(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pipelines[id]; !ok {
		return ErrPipelineNotFound
	}
	delete(s.pipelines, id)
	if err := s.persistLocked(); err != nil {
		return err
	}
	return nil
}

// CreateTask 创建并执行任务。
func (s *Service) CreateTask(req TaskCreateRequest) (IngestionResult, error) {
	pipelineID := strings.TrimSpace(req.PipelineID)
	if pipelineID == "" {
		return IngestionResult{}, ErrPipelineIDRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pipeline, ok := s.pipelines[pipelineID]
	if !ok {
		return IngestionResult{}, ErrPipelineNotFound
	}

	result := s.createTaskLocked(pipeline, req)
	if err := s.persistLocked(); err != nil {
		return IngestionResult{}, err
	}
	return result, nil
}

// UploadTask 上传文件并执行任务。
func (s *Service) UploadTask(upload UploadTaskRequest) (IngestionResult, error) {
	pipelineID := strings.TrimSpace(upload.PipelineID)
	s.mu.Lock()
	defer s.mu.Unlock()

	pipeline, ok := s.pipelines[pipelineID]
	if !ok {
		return IngestionResult{}, ErrPipelineNotFound
	}

	taskReq := TaskCreateRequest{
		PipelineID: pipelineID,
		Source: TaskSource{
			Type:     "file",
			Location: strings.TrimSpace(upload.FileName),
			FileName: strings.TrimSpace(upload.FileName),
		},
		Metadata: map[string]any{
			"fileSize": upload.FileSize,
		},
		CreatedBy: strings.TrimSpace(upload.CreatedBy),
	}
	if len(upload.Content) > 0 {
		taskReq.Metadata["mimeType"] = detectUploadMIME(upload.Content, upload.FileName)
		taskReq.Metadata["contentPreview"] = buildContentPreview(upload.Content, 2048)
	}
	result := s.createTaskLocked(pipeline, taskReq)
	if err := s.persistLocked(); err != nil {
		return IngestionResult{}, err
	}
	return result, nil
}

// GetTask 获取任务详情。
func (s *Service) GetTask(id string) (Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrTaskNotFound
	}
	return cloneTask(task), nil
}

// PageTasks 分页查询任务。
func (s *Service) PageTasks(pageNo, pageSize int, status string) PageResult[Task] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	current, size := normalizePage(pageNo, pageSize)
	records := make([]Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		if normalized := strings.ToLower(strings.TrimSpace(status)); normalized != "" && strings.ToLower(task.Status) != normalized {
			continue
		}
		records = append(records, cloneTask(task))
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreateTime.After(records[j].CreateTime)
	})
	return paginate(records, current, size)
}

// ListTaskNodes 获取任务节点记录。
func (s *Service) ListTaskNodes(taskID string) ([]TaskNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.tasks[taskID]; !ok {
		return nil, ErrTaskNotFound
	}
	nodes := s.taskNodes[taskID]
	result := make([]TaskNode, len(nodes))
	copy(result, nodes)
	return result, nil
}

func (s *Service) buildPipelineNodesLocked(items []PipelineNodeRequest) []PipelineNode {
	if len(items) == 0 {
		return []PipelineNode{}
	}
	result := make([]PipelineNode, 0, len(items))
	for _, item := range items {
		node := PipelineNode{
			ID:         s.nextPipelineNodeID,
			NodeID:     strings.TrimSpace(item.NodeID),
			NodeType:   strings.TrimSpace(item.NodeType),
			Settings:   cloneMap(item.Settings),
			Condition:  cloneMap(item.Condition),
			NextNodeID: strings.TrimSpace(item.NextNodeID),
		}
		s.nextPipelineNodeID++
		result = append(result, node)
	}
	return result
}

func (s *Service) createTaskLocked(pipeline Pipeline, req TaskCreateRequest) IngestionResult {
	now := s.now()
	taskID := "task_" + s.newID()
	nodes := make([]TaskNode, 0, len(pipeline.Nodes))
	logs := make([]TaskLog, 0, len(pipeline.Nodes))
	for index, node := range pipeline.Nodes {
		duration := int64(120 + index*40)
		taskNode := TaskNode{
			ID:         "tasknode_" + s.newID(),
			TaskID:     taskID,
			PipelineID: pipeline.ID,
			NodeID:     node.NodeID,
			NodeType:   node.NodeType,
			NodeOrder:  index + 1,
			Status:     "success",
			DurationMs: duration,
			Message:    "节点执行完成",
			Output: map[string]any{
				"nextNodeId": node.NextNodeID,
			},
			CreateTime: now,
			UpdateTime: now,
		}
		nodes = append(nodes, taskNode)
		logs = append(logs, TaskLog{
			NodeID:     node.NodeID,
			NodeType:   node.NodeType,
			Message:    "节点执行完成",
			DurationMs: duration,
			Success:    true,
		})
	}

	sourceType := normalizeSourceType(req.Source.Type)
	task := Task{
		ID:             taskID,
		PipelineID:     pipeline.ID,
		SourceType:     sourceType,
		SourceLocation: strings.TrimSpace(req.Source.Location),
		SourceFileName: strings.TrimSpace(req.Source.FileName),
		Status:         "success",
		ChunkCount:     maxInt(len(nodes), 1),
		Logs:           logs,
		Metadata:       cloneMap(req.Metadata),
		StartedAt:      now,
		CompletedAt:    now.Add(time.Duration(200+len(nodes)*50) * time.Millisecond),
		CreatedBy:      strings.TrimSpace(req.CreatedBy),
		CreateTime:     now,
		UpdateTime:     now,
	}
	if task.SourceFileName == "" && sourceType == "file" {
		task.SourceFileName = filepath.Base(task.SourceLocation)
	}
	s.tasks[taskID] = task
	s.taskNodes[taskID] = nodes

	return IngestionResult{
		TaskID:     taskID,
		PipelineID: pipeline.ID,
		Status:     task.Status,
		ChunkCount: task.ChunkCount,
		Message:    "任务执行成功",
	}
}

func cloneMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func clonePipeline(p Pipeline) Pipeline {
	result := p
	if len(p.Nodes) > 0 {
		result.Nodes = make([]PipelineNode, len(p.Nodes))
		for index, node := range p.Nodes {
			result.Nodes[index] = node
			result.Nodes[index].Settings = cloneMap(node.Settings)
			result.Nodes[index].Condition = cloneMap(node.Condition)
		}
	}
	return result
}

func cloneTask(t Task) Task {
	result := t
	if len(t.Logs) > 0 {
		result.Logs = make([]TaskLog, len(t.Logs))
		copy(result.Logs, t.Logs)
	}
	result.Metadata = cloneMap(t.Metadata)
	return result
}

func normalizeSourceType(sourceType string) string {
	value := strings.ToLower(strings.TrimSpace(sourceType))
	if value == "" {
		return "file"
	}
	return value
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func detectUploadMIME(content []byte, fileName string) string {
	if len(content) > 0 {
		return http.DetectContentType(content)
	}
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".md", ".markdown":
		return "text/markdown; charset=utf-8"
	case ".txt", ".csv", ".json", ".yaml", ".yml":
		return "text/plain; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

func buildContentPreview(content []byte, limit int) string {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	if limit <= 0 {
		limit = 2048
	}
	runes := []rune(strings.TrimSpace(text))
	if len(runes) > limit {
		runes = runes[:limit]
	}
	return string(runes)
}

func (s *Service) persistLocked() error {
	if s.repo == nil {
		return nil
	}
	pipelines := make([]Pipeline, 0, len(s.pipelines))
	for _, pipeline := range s.pipelines {
		pipelines = append(pipelines, clonePipeline(pipeline))
	}
	tasks := make([]Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, cloneTask(task))
	}
	taskNodes := make([]TaskNode, 0)
	for _, nodes := range s.taskNodes {
		for _, node := range nodes {
			next := node
			next.Output = cloneMap(node.Output)
			taskNodes = append(taskNodes, next)
		}
	}
	sort.Slice(pipelines, func(i, j int) bool { return pipelines[i].CreateTime.Before(pipelines[j].CreateTime) })
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreateTime.After(tasks[j].CreateTime) })
	sort.Slice(taskNodes, func(i, j int) bool {
		if taskNodes[i].TaskID == taskNodes[j].TaskID {
			return taskNodes[i].NodeOrder < taskNodes[j].NodeOrder
		}
		return taskNodes[i].TaskID < taskNodes[j].TaskID
	})
	return s.repo.SaveIngestionRecords(pipelines, tasks, taskNodes)
}
