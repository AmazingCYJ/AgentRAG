package knowledge

import (
	"path/filepath"
	"strings"
	"testing"

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
