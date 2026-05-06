package knowledge

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	platformstate "github.com/AmazingCYJ/AgentRAG/internal/platform/state"
)

func TestKnowledgeDataPersistsAcrossServiceRecreation(t *testing.T) {
	store, err := platformstate.NewFileStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatalf("create state store failed: %v", err)
	}

	service := NewService(store)
	kbID, err := service.CreateKnowledgeBase(KnowledgeBaseCreateRequest{
		Name:           "产品文档库",
		EmbeddingModel: "embedding-openai-large",
		CollectionName: "productdocs",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("create knowledge base failed: %v", err)
	}

	doc, err := service.UploadDocument(kbID, KnowledgeDocumentUploadRequest{
		SourceType:    "file",
		ProcessMode:   "chunk",
		ChunkStrategy: "structure_aware",
		FileName:      "产品说明.md",
		FileSize:      1024,
	}, "admin")
	if err != nil {
		t.Fatalf("upload document failed: %v", err)
	}
	if err := service.StartDocumentChunk(doc.ID); err != nil {
		t.Fatalf("start document chunk failed: %v", err)
	}

	recreated := NewService(store)
	recreatedKB, err := recreated.GetKnowledgeBase(kbID)
	if err != nil {
		t.Fatalf("get recreated knowledge base failed: %v", err)
	}
	if recreatedKB.Name != "产品文档库" {
		t.Fatalf("expected recreated knowledge base name 产品文档库, got %s", recreatedKB.Name)
	}
	recreatedDoc, err := recreated.GetDocument(doc.ID)
	if err != nil {
		t.Fatalf("get recreated document failed: %v", err)
	}
	if recreatedDoc.DocName != "产品说明.md" {
		t.Fatalf("expected recreated doc name 产品说明.md, got %s", recreatedDoc.DocName)
	}
	page, err := recreated.PageChunks(doc.ID, KnowledgeChunkPageRequest{Current: 1, Size: 20})
	if err != nil {
		t.Fatalf("page recreated chunks failed: %v", err)
	}
	if page.Total == 0 {
		t.Fatal("expected recreated chunk records")
	}
	logs, err := recreated.PageChunkLogs(doc.ID, 1, 10)
	if err != nil {
		t.Fatalf("page recreated chunk logs failed: %v", err)
	}
	if logs.Total == 0 {
		t.Fatal("expected recreated chunk logs")
	}
}

func TestBuildPromptContextReturnsRelevantChunks(t *testing.T) {
	store, err := platformstate.NewFileStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatalf("create state store failed: %v", err)
	}

	service := NewService(store)
	kbID, err := service.CreateKnowledgeBase(KnowledgeBaseCreateRequest{
		Name:           "OA文档库",
		EmbeddingModel: "embedding-openai-large",
		CollectionName: "oadocs",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("create knowledge base failed: %v", err)
	}
	doc, err := service.UploadDocument(kbID, KnowledgeDocumentUploadRequest{
		SourceType: "file",
		FileName:   "请假手册.md",
	}, "admin")
	if err != nil {
		t.Fatalf("upload document failed: %v", err)
	}
	_, err = service.CreateChunk(doc.ID, KnowledgeChunkCreateRequest{
		Content: "请假申请需要先进入审批中心并选择请假流程。",
	})
	if err != nil {
		t.Fatalf("create chunk failed: %v", err)
	}

	contextText, err := service.BuildPromptContext(nil, "怎么配置请假流程", 3)
	if err != nil {
		t.Fatalf("build prompt context failed: %v", err)
	}
	if !strings.Contains(contextText, "请假手册") {
		t.Fatalf("expected context to contain doc name, got %s", contextText)
	}
	if !strings.Contains(contextText, "审批中心") {
		t.Fatalf("expected context to contain chunk content, got %s", contextText)
	}
}

func TestStartDocumentChunkUsesUploadedTextContent(t *testing.T) {
	service := NewService(nil)
	kbID, err := service.CreateKnowledgeBase(KnowledgeBaseCreateRequest{
		Name:           "制度文档库",
		EmbeddingModel: "embedding-openai-large",
		CollectionName: "policydocs",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("create knowledge base failed: %v", err)
	}
	doc, err := service.UploadDocument(kbID, KnowledgeDocumentUploadRequest{
		SourceType:    "file",
		FileName:      "报销制度.md",
		TextContent:   "# 报销制度\n\n差旅报销需要提交发票和审批单。\n\n审批通过后由财务打款。",
		ProcessMode:   "chunk",
		ChunkStrategy: "structure_aware",
	}, "admin")
	if err != nil {
		t.Fatalf("upload document failed: %v", err)
	}

	if err := service.StartDocumentChunk(doc.ID); err != nil {
		t.Fatalf("start document chunk failed: %v", err)
	}
	page, err := service.PageChunks(doc.ID, KnowledgeChunkPageRequest{Current: 1, Size: 10})
	if err != nil {
		t.Fatalf("page chunks failed: %v", err)
	}
	if page.Total == 0 {
		t.Fatal("expected chunks generated from uploaded text")
	}
	combined := ""
	for _, chunk := range page.Records {
		combined += chunk.Content
	}
	if !strings.Contains(combined, "差旅报销需要提交发票和审批单") {
		t.Fatalf("expected chunks to contain uploaded content, got %s", combined)
	}
	if strings.Contains(combined, "当前处理模式") {
		t.Fatalf("expected uploaded content chunks instead of synthetic chunks, got %s", combined)
	}
}

func TestEnableDocumentPersistsState(t *testing.T) {
	repository := &memoryKnowledgeRepository{}
	service := NewServiceWithRepository(repository)
	kbID, err := service.CreateKnowledgeBase(KnowledgeBaseCreateRequest{
		Name:           "启用状态知识库",
		EmbeddingModel: "embedding-openai-large",
		CollectionName: "enabledocs",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("create knowledge base failed: %v", err)
	}
	doc, err := service.UploadDocument(kbID, KnowledgeDocumentUploadRequest{
		SourceType:    "file",
		FileName:      "开关状态.md",
		ProcessMode:   "chunk",
		ChunkStrategy: "structure_aware",
	}, "admin")
	if err != nil {
		t.Fatalf("upload document failed: %v", err)
	}

	if err := service.EnableDocument(doc.ID, false); err != nil {
		t.Fatalf("disable document failed: %v", err)
	}

	recreated := NewServiceWithRepository(repository)
	recreatedDoc, err := recreated.GetDocument(doc.ID)
	if err != nil {
		t.Fatalf("get recreated document failed: %v", err)
	}
	if recreatedDoc.Enabled {
		t.Fatal("expected disabled document state to persist")
	}
}

type memoryKnowledgeRepository struct {
	bases  []KnowledgeBase
	docs   []KnowledgeDocument
	chunks []KnowledgeChunk
	logs   []KnowledgeDocumentChunkLog
}

func (r *memoryKnowledgeRepository) LoadKnowledgeRecords() ([]KnowledgeBase, []KnowledgeDocument, []KnowledgeChunk, []KnowledgeDocumentChunkLog, error) {
	return r.bases, r.docs, r.chunks, r.logs, nil
}

func (r *memoryKnowledgeRepository) SaveKnowledgeRecords(bases []KnowledgeBase, docs []KnowledgeDocument, chunks []KnowledgeChunk, logs []KnowledgeDocumentChunkLog) error {
	r.bases = append([]KnowledgeBase(nil), bases...)
	r.docs = append([]KnowledgeDocument(nil), docs...)
	r.chunks = append([]KnowledgeChunk(nil), chunks...)
	r.logs = append([]KnowledgeDocumentChunkLog(nil), logs...)
	return nil
}

func TestKnowledgeServiceLoadsAndPersistsThroughRepository(t *testing.T) {
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)
	repository := &memoryKnowledgeRepository{
		bases: []KnowledgeBase{
			{
				ID:             "kb_existing",
				Name:           "已有知识库",
				EmbeddingModel: "embedding-openai-large",
				CollectionName: "existingdocs",
				CreatedBy:      "admin",
				CreateTime:     now,
				UpdateTime:     now,
			},
		},
		docs: []KnowledgeDocument{
			{
				ID:            "doc_existing",
				KBID:          "kb_existing",
				DocName:       "已有文档.md",
				SourceType:    "file",
				Enabled:       true,
				ChunkCount:    1,
				FileType:      "md",
				ProcessMode:   "chunk",
				ChunkStrategy: "structure_aware",
				Status:        "success",
				CreatedBy:     "admin",
				UpdatedBy:     "admin",
				CreateTime:    now,
				UpdateTime:    now,
			},
		},
		chunks: []KnowledgeChunk{
			{
				ID:          "chunk_existing",
				KBID:        "kb_existing",
				DocID:       "doc_existing",
				ChunkIndex:  0,
				Content:     "已有 Chunk 内容",
				ContentHash: "hash_existing",
				CharCount:   10,
				TokenCount:  3,
				Enabled:     1,
				CreateTime:  now,
				UpdateTime:  now,
			},
		},
		logs: []KnowledgeDocumentChunkLog{
			{
				ID:         "log_existing",
				DocID:      "doc_existing",
				Status:     "success",
				ChunkCount: 1,
				CreateTime: now,
			},
		},
	}

	service := NewServiceWithRepository(repository)
	loadedKB, err := service.GetKnowledgeBase("kb_existing")
	if err != nil {
		t.Fatalf("load knowledge base failed: %v", err)
	}
	if loadedKB.Name != "已有知识库" || loadedKB.DocumentCount != 1 {
		t.Fatalf("unexpected loaded knowledge base %#v", loadedKB)
	}
	loadedChunks, err := service.PageChunks("doc_existing", KnowledgeChunkPageRequest{Current: 1, Size: 10})
	if err != nil {
		t.Fatalf("page loaded chunks failed: %v", err)
	}
	if loadedChunks.Total != 1 {
		t.Fatalf("expected 1 loaded chunk, got %d", loadedChunks.Total)
	}

	kbID, err := service.CreateKnowledgeBase(KnowledgeBaseCreateRequest{
		Name:           "新增知识库",
		EmbeddingModel: "embedding-local-bge",
		CollectionName: "newdocs",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("create knowledge base failed: %v", err)
	}
	if kbID == "" {
		t.Fatal("expected new knowledge base id")
	}
	if len(repository.bases) != 2 {
		t.Fatalf("expected repository to save 2 knowledge bases, got %d", len(repository.bases))
	}
}
