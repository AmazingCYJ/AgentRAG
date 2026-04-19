package ragtrace

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
	// ErrTraceNotFound 表示 Trace 记录不存在。
	ErrTraceNotFound = errors.New("链路记录不存在")
)

// Run 表示链路运行记录。
type Run struct {
	TraceID        string    `json:"traceId"`
	TraceName      string    `json:"traceName,omitempty"`
	EntryMethod    string    `json:"entryMethod,omitempty"`
	ConversationID string    `json:"conversationId,omitempty"`
	TaskID         string    `json:"taskId,omitempty"`
	UserName       string    `json:"userName,omitempty"`
	Username       string    `json:"username,omitempty"`
	UserID         string    `json:"userId,omitempty"`
	Status         string    `json:"status,omitempty"`
	ErrorMessage   string    `json:"errorMessage,omitempty"`
	DurationMs     int64     `json:"durationMs,omitempty"`
	StartTime      time.Time `json:"startTime,omitempty"`
	EndTime        time.Time `json:"endTime,omitempty"`
}

// Node 表示链路节点明细。
type Node struct {
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

// Detail 表示链路详情。
type Detail struct {
	Run   Run    `json:"run"`
	Nodes []Node `json:"nodes"`
}

// PageResult 定义分页结构。
type PageResult struct {
	Records []Run `json:"records"`
	Total   int   `json:"total"`
	Size    int   `json:"size"`
	Current int   `json:"current"`
	Pages   int   `json:"pages"`
}

// RunQuery 定义分页筛选条件。
type RunQuery struct {
	Current        int
	Size           int
	TraceID        string
	ConversationID string
	TaskID         string
	Status         string
}

// ChatTraceRecord 定义聊天流程写入 Trace 的输入。
type ChatTraceRecord struct {
	TraceName      string
	ConversationID string
	TaskID         string
	UserName       string
	Username       string
	UserID         string
	Status         string
	ErrorMessage   string
	DurationMs     int64
	StartTime      time.Time
	EndTime        time.Time
	DeepThinking   bool
}

// Service 提供 RAG Trace 查询与记录能力。
type Service struct {
	mu    sync.RWMutex
	runs  map[string]Run
	nodes map[string][]Node

	newID func() string
	store *platformstate.FileStore
}

// NewService 创建 Trace 服务，并写入一条初始示例数据。
func NewService(store *platformstate.FileStore) *Service {
	service := &Service{
		runs:  make(map[string]Run),
		nodes: make(map[string][]Node),
		newID: func() string {
			return strings.ReplaceAll(guid.S(), "-", "")
		},
		store: store,
	}
	if snapshot, err := service.loadSnapshot(); err == nil && len(snapshot.RagTraceRuns) > 0 {
		for _, run := range snapshot.RagTraceRuns {
			service.runs[run.TraceID] = Run{
				TraceID:        run.TraceID,
				TraceName:      run.TraceName,
				EntryMethod:    run.EntryMethod,
				ConversationID: run.ConversationID,
				TaskID:         run.TaskID,
				UserName:       run.UserName,
				Username:       run.Username,
				UserID:         run.UserID,
				Status:         run.Status,
				ErrorMessage:   run.ErrorMessage,
				DurationMs:     run.DurationMs,
				StartTime:      run.StartTime,
				EndTime:        run.EndTime,
			}
		}
		for _, node := range snapshot.RagTraceNodes {
			service.nodes[node.TraceID] = append(service.nodes[node.TraceID], Node{
				TraceID:      node.TraceID,
				NodeID:       node.NodeID,
				ParentNodeID: node.ParentNodeID,
				Depth:        node.Depth,
				NodeType:     node.NodeType,
				NodeName:     node.NodeName,
				ClassName:    node.ClassName,
				MethodName:   node.MethodName,
				Status:       node.Status,
				ErrorMessage: node.ErrorMessage,
				DurationMs:   node.DurationMs,
				StartTime:    node.StartTime,
				EndTime:      node.EndTime,
			})
		}
	} else {
		service.seed()
	}
	return service
}

func (s *Service) loadSnapshot() (platformstate.Snapshot, error) {
	if s.store == nil {
		return platformstate.Snapshot{}, nil
	}
	return s.store.Load()
}

// PageRuns 返回链路运行分页数据。
func (s *Service) PageRuns(query RunQuery) PageResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	current := query.Current
	if current <= 0 {
		current = 1
	}
	size := query.Size
	if size <= 0 {
		size = 10
	}

	filtered := make([]Run, 0, len(s.runs))
	for _, run := range s.runs {
		if matchRun(run, query) {
			filtered = append(filtered, run)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].StartTime.After(filtered[j].StartTime)
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

	records := make([]Run, end-start)
	copy(records, filtered[start:end])
	return PageResult{
		Records: records,
		Total:   total,
		Size:    size,
		Current: current,
		Pages:   pages,
	}
}

// Detail 返回指定 Trace 详情。
func (s *Service) Detail(traceID string) (Detail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	run, ok := s.runs[traceID]
	if !ok {
		return Detail{}, ErrTraceNotFound
	}
	nodes := s.copyNodesLocked(traceID)
	return Detail{
		Run:   run,
		Nodes: nodes,
	}, nil
}

// ListNodes 返回指定 Trace 的节点列表。
func (s *Service) ListNodes(traceID string) ([]Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.runs[traceID]; !ok {
		return nil, ErrTraceNotFound
	}
	return s.copyNodesLocked(traceID), nil
}

// SnapshotRuns 返回全部链路运行记录快照。
func (s *Service) SnapshotRuns() []Run {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Run, 0, len(s.runs))
	for _, run := range s.runs {
		result = append(result, run)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartTime.After(result[j].StartTime)
	})
	return result
}

// RecordChatTrace 写入一次聊天运行记录。
func (s *Service) RecordChatTrace(record ChatTraceRecord) string {
	traceID := "trace_" + s.newID()
	startTime := record.StartTime
	endTime := record.EndTime
	if endTime.Before(startTime) {
		endTime = startTime
	}
	durationMs := record.DurationMs
	if durationMs <= 0 {
		durationMs = endTime.Sub(startTime).Milliseconds()
		if durationMs < 1 {
			durationMs = 1
		}
	}

	run := Run{
		TraceID:        traceID,
		TraceName:      defaultTraceName(record.TraceName),
		EntryMethod:    "RAGChatController.chat",
		ConversationID: strings.TrimSpace(record.ConversationID),
		TaskID:         strings.TrimSpace(record.TaskID),
		UserName:       strings.TrimSpace(record.UserName),
		Username:       strings.TrimSpace(record.Username),
		UserID:         strings.TrimSpace(record.UserID),
		Status:         normalizeStatus(record.Status),
		ErrorMessage:   strings.TrimSpace(record.ErrorMessage),
		DurationMs:     durationMs,
		StartTime:      startTime,
		EndTime:        endTime,
	}
	nodes := buildChatNodes(traceID, run, record.DeepThinking)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.runs[traceID] = run
	s.nodes[traceID] = nodes
	_ = s.persistLocked()
	return traceID
}

func (s *Service) copyNodesLocked(traceID string) []Node {
	source := s.nodes[traceID]
	result := make([]Node, len(source))
	copy(result, source)
	return result
}

func (s *Service) seed() {
	now := time.Now().Add(-20 * time.Minute)
	run := Run{
		TraceID:        "trace_seed_demo",
		TraceName:      "示例链路",
		EntryMethod:    "RAGChatController.chat",
		ConversationID: "conv_seed_demo",
		TaskID:         "task_seed_demo",
		UserName:       "admin",
		Username:       "admin",
		UserID:         "u_admin",
		Status:         "success",
		DurationMs:     1280,
		StartTime:      now,
		EndTime:        now.Add(1280 * time.Millisecond),
	}
	nodes := []Node{
		{
			TraceID:    run.TraceID,
			NodeID:     "node-entry",
			Depth:      0,
			NodeType:   "ENTRY",
			NodeName:   "Chat Entry",
			ClassName:  "RAGChatController",
			MethodName: "chat",
			Status:     "success",
			DurationMs: 1280,
			StartTime:  run.StartTime,
			EndTime:    run.EndTime,
		},
		{
			TraceID:      run.TraceID,
			NodeID:       "node-generate",
			ParentNodeID: "node-entry",
			Depth:        1,
			NodeType:     "LLM",
			NodeName:     "Generate Response",
			ClassName:    "RAGChatService",
			MethodName:   "streamChat",
			Status:       "success",
			DurationMs:   1180,
			StartTime:    run.StartTime.Add(100 * time.Millisecond),
			EndTime:      run.EndTime,
		},
	}

	s.runs[run.TraceID] = run
	s.nodes[run.TraceID] = nodes
	_ = s.persistLocked()
}

func matchRun(run Run, query RunQuery) bool {
	if !containsFold(run.TraceID, query.TraceID) {
		return false
	}
	if !containsFold(run.ConversationID, query.ConversationID) {
		return false
	}
	if !containsFold(run.TaskID, query.TaskID) {
		return false
	}
	status := normalizeStatus(query.Status)
	if status != "" && normalizeStatus(run.Status) != status {
		return false
	}
	return true
}

func containsFold(value, keyword string) bool {
	needle := strings.ToLower(strings.TrimSpace(keyword))
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(value), needle)
}

func normalizeStatus(status string) string {
	value := strings.ToLower(strings.TrimSpace(status))
	switch value {
	case "success", "failed", "running":
		return value
	default:
		if value == "" {
			return "success"
		}
		return value
	}
}

func defaultTraceName(traceName string) string {
	value := strings.TrimSpace(traceName)
	if value == "" {
		return "RAG Chat Run"
	}
	return value
}

func buildChatNodes(traceID string, run Run, deepThinking bool) []Node {
	start := run.StartTime
	end := run.EndTime
	total := run.DurationMs
	if total <= 0 {
		total = 1
	}

	makeNode := func(id, parentID string, depth int, nodeType, nodeName, className, methodName string, offsetMs, durationMs int64, status, errorMessage string) Node {
		nodeStart := start.Add(time.Duration(offsetMs) * time.Millisecond)
		nodeEnd := nodeStart.Add(time.Duration(durationMs) * time.Millisecond)
		if nodeEnd.After(end) {
			nodeEnd = end
		}
		return Node{
			TraceID:      traceID,
			NodeID:       id,
			ParentNodeID: parentID,
			Depth:        depth,
			NodeType:     nodeType,
			NodeName:     nodeName,
			ClassName:    className,
			MethodName:   methodName,
			Status:       status,
			ErrorMessage: errorMessage,
			DurationMs:   durationMs,
			StartTime:    nodeStart,
			EndTime:      nodeEnd,
		}
	}

	nodes := []Node{
		makeNode("node-entry", "", 0, "ENTRY", "Chat Entry", "RAGChatController", "chat", 0, total, run.Status, run.ErrorMessage),
		makeNode("node-memory", "node-entry", 1, "MEMORY", "Load Conversation", "ConversationService", "ListMessages", 0, maxInt64(40, total/8), "success", ""),
	}

	offset := maxInt64(40, total/8)
	if deepThinking {
		thinkingDuration := maxInt64(80, total/3)
		nodes = append(nodes, makeNode(
			"node-thinking",
			"node-entry",
			1,
			"REASONING",
			"Deep Thinking",
			"EinoWorkflow",
			"reason",
			offset,
			thinkingDuration,
			run.Status,
			run.ErrorMessage,
		))
		offset += thinkingDuration
	}

	generateDuration := total - offset
	if generateDuration < 80 {
		generateDuration = maxInt64(80, total/3)
	}
	nodes = append(nodes, makeNode(
		"node-generate",
		"node-entry",
		1,
		"LLM",
		"Generate Response",
		"EinoWorkflow",
		"stream",
		offset,
		generateDuration,
		run.Status,
		run.ErrorMessage,
	))

	return nodes
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func (s *Service) persistLocked() error {
	if s.store == nil {
		return nil
	}
	runRecords := make([]platformstate.RagTraceRunRecord, 0, len(s.runs))
	for _, run := range s.runs {
		runRecords = append(runRecords, platformstate.RagTraceRunRecord{
			TraceID:        run.TraceID,
			TraceName:      run.TraceName,
			EntryMethod:    run.EntryMethod,
			ConversationID: run.ConversationID,
			TaskID:         run.TaskID,
			UserName:       run.UserName,
			Username:       run.Username,
			UserID:         run.UserID,
			Status:         run.Status,
			ErrorMessage:   run.ErrorMessage,
			DurationMs:     run.DurationMs,
			StartTime:      run.StartTime,
			EndTime:        run.EndTime,
		})
	}
	nodeRecords := make([]platformstate.RagTraceNodeRecord, 0)
	for _, nodes := range s.nodes {
		for _, node := range nodes {
			nodeRecords = append(nodeRecords, platformstate.RagTraceNodeRecord{
				TraceID:      node.TraceID,
				NodeID:       node.NodeID,
				ParentNodeID: node.ParentNodeID,
				Depth:        node.Depth,
				NodeType:     node.NodeType,
				NodeName:     node.NodeName,
				ClassName:    node.ClassName,
				MethodName:   node.MethodName,
				Status:       node.Status,
				ErrorMessage: node.ErrorMessage,
				DurationMs:   node.DurationMs,
				StartTime:    node.StartTime,
				EndTime:      node.EndTime,
			})
		}
	}
	sort.Slice(runRecords, func(i, j int) bool { return runRecords[i].StartTime.After(runRecords[j].StartTime) })
	sort.Slice(nodeRecords, func(i, j int) bool {
		if nodeRecords[i].TraceID == nodeRecords[j].TraceID {
			return nodeRecords[i].StartTime.Before(nodeRecords[j].StartTime)
		}
		return nodeRecords[i].TraceID < nodeRecords[j].TraceID
	})
	return s.store.Update(func(snapshot *platformstate.Snapshot) {
		snapshot.RagTraceRuns = runRecords
		snapshot.RagTraceNodes = nodeRecords
	})
}
