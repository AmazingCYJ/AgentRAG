package knowledge

import (
	"context"
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
	if err := service.StartDocumentChunk(context.Background(), doc.ID); err != nil {
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
	_, err = service.CreateChunk(context.Background(), doc.ID, KnowledgeChunkCreateRequest{
		Content: "请假申请需要先进入审批中心并选择请假流程。",
	})
	if err != nil {
		t.Fatalf("create chunk failed: %v", err)
	}

	contextText, err := service.BuildPromptContext(context.Background(), "怎么配置请假流程", 3)
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

func TestBuildPromptContextIncludesSourceMetadata(t *testing.T) {
	service := NewService(nil)
	kbID, err := service.CreateKnowledgeBase(KnowledgeBaseCreateRequest{
		Name:           "流程知识库",
		EmbeddingModel: "embedding-openai-large",
		CollectionName: "flowdocs",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("create knowledge base failed: %v", err)
	}
	doc, err := service.UploadDocument(kbID, KnowledgeDocumentUploadRequest{
		SourceType:     "url",
		SourceLocation: "https://example.com/flow.md",
		FileName:       "流程手册.md",
	}, "admin")
	if err != nil {
		t.Fatalf("upload document failed: %v", err)
	}
	_, err = service.CreateChunk(context.Background(), doc.ID, KnowledgeChunkCreateRequest{
		Content: "审批流程需要先配置节点和负责人。",
	})
	if err != nil {
		t.Fatalf("create chunk failed: %v", err)
	}

	contextText, err := service.BuildPromptContext(context.Background(), "审批流程负责人怎么配置", 1)
	if err != nil {
		t.Fatalf("build prompt context failed: %v", err)
	}
	for _, expected := range []string{
		"知识库：流程知识库",
		"文档：流程手册.md",
		"来源：https://example.com/flow.md",
		"Chunk：0",
		"审批流程需要先配置节点和负责人。",
	} {
		if !strings.Contains(contextText, expected) {
			t.Fatalf("expected context to contain %q, got %s", expected, contextText)
		}
	}
}

func TestBuildPromptContextMatchesBusinessSynonyms(t *testing.T) {
	service := NewService(nil)
	kbID, err := service.CreateKnowledgeBase(KnowledgeBaseCreateRequest{
		Name:           "财务知识库",
		EmbeddingModel: "embedding-openai-large",
		CollectionName: "finance_docs",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("create knowledge base failed: %v", err)
	}
	doc, err := service.UploadDocument(kbID, KnowledgeDocumentUploadRequest{
		SourceType: "file",
		FileName:   "报销制度.md",
	}, "admin")
	if err != nil {
		t.Fatalf("upload document failed: %v", err)
	}
	_, err = service.CreateChunk(context.Background(), doc.ID, KnowledgeChunkCreateRequest{
		Content: "报销流程需要准备发票、审批单和付款账号。",
	})
	if err != nil {
		t.Fatalf("create chunk failed: %v", err)
	}

	contextText, err := service.BuildPromptContext(context.Background(), "怎么报账", 1)
	if err != nil {
		t.Fatalf("build prompt context failed: %v", err)
	}
	if !strings.Contains(contextText, "报销流程需要准备发票") {
		t.Fatalf("expected synonym query to retrieve reimbursement chunk, got %s", contextText)
	}
}

func TestBuildPromptContextUsesVectorSimilarityForTiedKeywordMatches(t *testing.T) {
	service := NewService(nil)
	kbID, err := service.CreateKnowledgeBase(KnowledgeBaseCreateRequest{
		Name:           "财务知识库",
		EmbeddingModel: "embedding-local-bge",
		CollectionName: "finance_docs",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("create knowledge base failed: %v", err)
	}
	noisyDoc, err := service.UploadDocument(kbID, KnowledgeDocumentUploadRequest{
		SourceType: "file",
		FileName:   "A噪声说明.md",
	}, "admin")
	if err != nil {
		t.Fatalf("upload noisy document failed: %v", err)
	}
	_, err = service.CreateChunk(context.Background(), noisyDoc.ID, KnowledgeChunkCreateRequest{
		Content: "报销流程 发票 账号 密码 登录 审批人 通知 配置 表单 字段 权限 菜单 角色 同步 导出",
	})
	if err != nil {
		t.Fatalf("create noisy chunk failed: %v", err)
	}
	focusedDoc, err := service.UploadDocument(kbID, KnowledgeDocumentUploadRequest{
		SourceType: "file",
		FileName:   "Z报销指南.md",
	}, "admin")
	if err != nil {
		t.Fatalf("upload focused document failed: %v", err)
	}
	_, err = service.CreateChunk(context.Background(), focusedDoc.ID, KnowledgeChunkCreateRequest{
		Content: "报销流程 发票",
	})
	if err != nil {
		t.Fatalf("create focused chunk failed: %v", err)
	}

	contextText, err := service.BuildPromptContext(context.Background(), "报销流程 发票", 1)
	if err != nil {
		t.Fatalf("build prompt context failed: %v", err)
	}
	if !strings.Contains(contextText, "Z报销指南.md") || !strings.Contains(contextText, "报销流程 发票") {
		t.Fatalf("expected vector similarity to prefer focused chunk, got %s", contextText)
	}
	if strings.Contains(contextText, "A噪声说明.md") {
		t.Fatalf("expected noisy chunk to be ranked below focused chunk, got %s", contextText)
	}
}

func TestSearchDocumentsMatchesBusinessSynonyms(t *testing.T) {
	service := NewService(nil)
	kbID, err := service.CreateKnowledgeBase(KnowledgeBaseCreateRequest{
		Name:           "财务知识库",
		EmbeddingModel: "embedding-openai-large",
		CollectionName: "finance_docs",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("create knowledge base failed: %v", err)
	}
	doc, err := service.UploadDocument(kbID, KnowledgeDocumentUploadRequest{
		SourceType: "file",
		FileName:   "报销制度.md",
	}, "admin")
	if err != nil {
		t.Fatalf("upload document failed: %v", err)
	}

	result := service.SearchDocuments(context.Background(), "报账", 5)
	if len(result) != 1 {
		t.Fatalf("expected one synonym-matched document, got %#v", result)
	}
	if result[0].ID != doc.ID || result[0].KBName != "财务知识库" {
		t.Fatalf("unexpected search result %#v", result[0])
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

	if err := service.StartDocumentChunk(context.Background(), doc.ID); err != nil {
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

func TestKnowledgeChunkCachesLocalEmbedding(t *testing.T) {
	service := NewService(nil)
	kbID, err := service.CreateKnowledgeBase(KnowledgeBaseCreateRequest{
		Name:           "向量知识库",
		EmbeddingModel: "embedding-local-bge",
		CollectionName: "vector_docs",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("create knowledge base failed: %v", err)
	}
	doc, err := service.UploadDocument(kbID, KnowledgeDocumentUploadRequest{
		SourceType: "file",
		FileName:   "向量说明.md",
	}, "admin")
	if err != nil {
		t.Fatalf("upload document failed: %v", err)
	}
	chunk, err := service.CreateChunk(context.Background(), doc.ID, KnowledgeChunkCreateRequest{
		Content: "报销流程需要发票和审批单",
	})
	if err != nil {
		t.Fatalf("create chunk failed: %v", err)
	}
	if chunk.EmbeddingModel != "embedding-local-bge" {
		t.Fatalf("expected chunk embedding model to follow knowledge base, got %s", chunk.EmbeddingModel)
	}
	if len(chunk.Embedding) != localEmbeddingDimensions {
		t.Fatalf("expected chunk embedding dimensions %d, got %d", localEmbeddingDimensions, len(chunk.Embedding))
	}
	oldEmbedding := cloneFloat64Slice(chunk.Embedding)

	if err := service.UpdateChunk(context.Background(), doc.ID, chunk.ID, KnowledgeChunkUpdateRequest{Content: "账号登录需要密码"}); err != nil {
		t.Fatalf("update chunk failed: %v", err)
	}
	page, err := service.PageChunks(doc.ID, KnowledgeChunkPageRequest{Current: 1, Size: 10})
	if err != nil {
		t.Fatalf("page chunks failed: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("expected one chunk, got %#v", page)
	}
	updated := page.Records[0]
	if updated.EmbeddingModel != "embedding-local-bge" || len(updated.Embedding) != localEmbeddingDimensions {
		t.Fatalf("expected updated chunk embedding cache, got %#v", updated)
	}
	if cosineSimilarity(oldEmbedding, updated.Embedding) >= 0.999999 {
		t.Fatalf("expected embedding to change after content update")
	}
}

func TestKnowledgeChunkUsesInjectedEmbeddingService(t *testing.T) {
	embeddingService := &fakeEmbeddingService{
		vectors: [][]float64{
			{0.1, 0.2, 0.3},
			{0.4, 0.5, 0.6},
		},
	}
	service := NewServiceWithDependencies(nil, embeddingService)
	kbID, err := service.CreateKnowledgeBase(KnowledgeBaseCreateRequest{
		Name:           "外部向量知识库",
		EmbeddingModel: "embedding-remote",
		CollectionName: "remote_docs",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("create knowledge base failed: %v", err)
	}
	doc, err := service.UploadDocument(kbID, KnowledgeDocumentUploadRequest{
		SourceType: "file",
		FileName:   "外部向量说明.md",
	}, "admin")
	if err != nil {
		t.Fatalf("upload document failed: %v", err)
	}
	chunk, err := service.CreateChunk(context.Background(), doc.ID, KnowledgeChunkCreateRequest{Content: "第一段内容"})
	if err != nil {
		t.Fatalf("create chunk failed: %v", err)
	}
	if chunk.EmbeddingModel != "embedding-remote" || len(chunk.Embedding) != 3 || chunk.Embedding[2] != 0.3 {
		t.Fatalf("expected injected embedding on create, got %#v", chunk)
	}

	if err := service.UpdateChunk(context.Background(), doc.ID, chunk.ID, KnowledgeChunkUpdateRequest{Content: "第二段内容"}); err != nil {
		t.Fatalf("update chunk failed: %v", err)
	}
	page, err := service.PageChunks(doc.ID, KnowledgeChunkPageRequest{Current: 1, Size: 10})
	if err != nil {
		t.Fatalf("page chunks failed: %v", err)
	}
	if len(page.Records[0].Embedding) != 3 || page.Records[0].Embedding[0] != 0.4 {
		t.Fatalf("expected injected embedding on update, got %#v", page.Records[0])
	}
	if len(embeddingService.models) != 2 || embeddingService.models[0] != "embedding-remote" || embeddingService.models[1] != "embedding-remote" {
		t.Fatalf("expected embedding service to receive knowledge base model, got %#v", embeddingService.models)
	}
}

func TestKnowledgeChunkSyncsVectorStoreOnCreateAndUpdate(t *testing.T) {
	embeddingService := &fakeEmbeddingService{
		vectors: [][]float64{
			{0.1, 0.2, 0.3},
			{0.4, 0.5, 0.6},
		},
	}
	vectorStore := &fakeVectorStore{}
	service := NewServiceWithDependencies(nil, embeddingService, vectorStore)
	kbID, err := service.CreateKnowledgeBase(KnowledgeBaseCreateRequest{
		Name:           "向量同步知识库",
		EmbeddingModel: "embedding-remote",
		CollectionName: "remote_docs",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("create knowledge base failed: %v", err)
	}
	doc, err := service.UploadDocument(kbID, KnowledgeDocumentUploadRequest{
		SourceType: "file",
		FileName:   "向量同步.md",
	}, "admin")
	if err != nil {
		t.Fatalf("upload document failed: %v", err)
	}

	chunk, err := service.CreateChunk(context.Background(), doc.ID, KnowledgeChunkCreateRequest{Content: "第一段内容"})
	if err != nil {
		t.Fatalf("create chunk failed: %v", err)
	}
	if len(vectorStore.upserts) != 1 || len(vectorStore.upserts[0]) != 1 {
		t.Fatalf("expected one vector upsert on create, got %#v", vectorStore.upserts)
	}
	createdVector := vectorStore.upserts[0][0]
	if createdVector.ID != chunk.ID || createdVector.ChunkID != chunk.ID || createdVector.DocID != doc.ID || createdVector.KBID != kbID {
		t.Fatalf("unexpected created vector identity %#v", createdVector)
	}
	if createdVector.CollectionName != "remote_docs" || createdVector.EmbeddingModel != "embedding-remote" || createdVector.Content != "第一段内容" {
		t.Fatalf("unexpected created vector payload %#v", createdVector)
	}
	if len(createdVector.Embedding) != 3 || createdVector.Embedding[0] != 0.1 {
		t.Fatalf("unexpected created vector embedding %#v", createdVector.Embedding)
	}

	if err := service.UpdateChunk(context.Background(), doc.ID, chunk.ID, KnowledgeChunkUpdateRequest{Content: "第二段内容"}); err != nil {
		t.Fatalf("update chunk failed: %v", err)
	}
	if len(vectorStore.upserts) != 2 || len(vectorStore.upserts[1]) != 1 {
		t.Fatalf("expected second vector upsert on update, got %#v", vectorStore.upserts)
	}
	updatedVector := vectorStore.upserts[1][0]
	if updatedVector.ID != chunk.ID || updatedVector.Content != "第二段内容" {
		t.Fatalf("unexpected updated vector payload %#v", updatedVector)
	}
	if len(updatedVector.Embedding) != 3 || updatedVector.Embedding[0] != 0.4 {
		t.Fatalf("unexpected updated vector embedding %#v", updatedVector.Embedding)
	}
}

func TestKnowledgeServiceDeletesVectorsForRemovedChunks(t *testing.T) {
	embeddingService := &fakeEmbeddingService{
		vectors: [][]float64{
			{1, 0, 0},
			{0, 1, 0},
			{0, 0, 1},
		},
	}
	vectorStore := &fakeVectorStore{}
	service := NewServiceWithDependencies(nil, embeddingService, vectorStore)
	kbID, err := service.CreateKnowledgeBase(KnowledgeBaseCreateRequest{
		Name:           "删除同步知识库",
		EmbeddingModel: "embedding-remote",
		CollectionName: "delete_docs",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("create knowledge base failed: %v", err)
	}
	doc, err := service.UploadDocument(kbID, KnowledgeDocumentUploadRequest{SourceType: "file", FileName: "删除同步.md"}, "admin")
	if err != nil {
		t.Fatalf("upload document failed: %v", err)
	}
	chunkA, err := service.CreateChunk(context.Background(), doc.ID, KnowledgeChunkCreateRequest{Content: "第一段"})
	if err != nil {
		t.Fatalf("create first chunk failed: %v", err)
	}
	chunkB, err := service.CreateChunk(context.Background(), doc.ID, KnowledgeChunkCreateRequest{Content: "第二段"})
	if err != nil {
		t.Fatalf("create second chunk failed: %v", err)
	}
	docForKBDelete, err := service.UploadDocument(kbID, KnowledgeDocumentUploadRequest{SourceType: "file", FileName: "知识库删除.md"}, "admin")
	if err != nil {
		t.Fatalf("upload second document failed: %v", err)
	}
	chunkC, err := service.CreateChunk(context.Background(), docForKBDelete.ID, KnowledgeChunkCreateRequest{Content: "第三段"})
	if err != nil {
		t.Fatalf("create third chunk failed: %v", err)
	}

	if err := service.DeleteChunk(context.Background(), doc.ID, chunkA.ID); err != nil {
		t.Fatalf("delete chunk failed: %v", err)
	}
	if len(vectorStore.deleted) != 1 || !sameStrings(vectorStore.deleted[0], []string{chunkA.ID}) {
		t.Fatalf("expected deleted chunk vector id %s, got %#v", chunkA.ID, vectorStore.deleted)
	}

	if err := service.DeleteDocument(context.Background(), doc.ID); err != nil {
		t.Fatalf("delete document failed: %v", err)
	}
	if len(vectorStore.deleted) != 2 || !sameStrings(vectorStore.deleted[1], []string{chunkB.ID}) {
		t.Fatalf("expected deleted document vector id %s, got %#v", chunkB.ID, vectorStore.deleted)
	}

	if err := service.DeleteKnowledgeBase(context.Background(), kbID); err != nil {
		t.Fatalf("delete knowledge base failed: %v", err)
	}
	if len(vectorStore.deleted) != 3 || !sameStrings(vectorStore.deleted[2], []string{chunkC.ID}) {
		t.Fatalf("expected deleted knowledge base vector id %s, got %#v", chunkC.ID, vectorStore.deleted)
	}
}

func TestBuildPromptContextUsesVectorStoreResultsWhenAvailable(t *testing.T) {
	embeddingService := &fakeEmbeddingService{
		vectors: [][]float64{
			{1, 0, 0},
			{0, 1, 0},
			{0.7, 0.2, 0.1},
		},
	}
	vectorStore := &fakeVectorStore{}
	service := NewServiceWithDependencies(nil, embeddingService, vectorStore)
	kbID, err := service.CreateKnowledgeBase(KnowledgeBaseCreateRequest{
		Name:           "向量检索知识库",
		EmbeddingModel: "embedding-remote",
		CollectionName: "vector_search_docs",
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("create knowledge base failed: %v", err)
	}
	doc, err := service.UploadDocument(kbID, KnowledgeDocumentUploadRequest{
		SourceType:     "url",
		SourceLocation: "https://example.com/vector.md",
		FileName:       "向量检索.md",
	}, "admin")
	if err != nil {
		t.Fatalf("upload document failed: %v", err)
	}
	_, err = service.CreateChunk(context.Background(), doc.ID, KnowledgeChunkCreateRequest{Content: "无关内容"})
	if err != nil {
		t.Fatalf("create first chunk failed: %v", err)
	}
	targetChunk, err := service.CreateChunk(context.Background(), doc.ID, KnowledgeChunkCreateRequest{Content: "付款账号需要先完成审批"})
	if err != nil {
		t.Fatalf("create target chunk failed: %v", err)
	}
	vectorStore.searchResults = []VectorSearchResult{
		{Vector: KnowledgeVector{ID: targetChunk.ID, ChunkID: targetChunk.ID}, Score: 0.92},
	}

	contextText, err := service.BuildPromptContext(context.Background(), "付款账户怎么配置", 1)
	if err != nil {
		t.Fatalf("build prompt context failed: %v", err)
	}
	if len(vectorStore.searches) != 1 {
		t.Fatalf("expected one vector search request, got %#v", vectorStore.searches)
	}
	searchRequest := vectorStore.searches[0]
	if searchRequest.Query != "付款账户怎么配置" || searchRequest.CollectionName != "vector_search_docs" || searchRequest.EmbeddingModel != "embedding-remote" {
		t.Fatalf("unexpected vector search request %#v", searchRequest)
	}
	if len(searchRequest.Embedding) != 3 || searchRequest.Embedding[0] != 0.7 {
		t.Fatalf("expected query embedding from injected service, got %#v", searchRequest.Embedding)
	}
	if !strings.Contains(contextText, "向量检索知识库") || !strings.Contains(contextText, "向量检索.md") || !strings.Contains(contextText, "付款账号需要先完成审批") {
		t.Fatalf("expected vector result context, got %s", contextText)
	}

	documents := service.SearchDocuments(context.Background(), "付款账户怎么配置", 1)
	if len(documents) != 1 || documents[0].ID != doc.ID || documents[0].KBName != "向量检索知识库" {
		t.Fatalf("expected vector document search result, got %#v", documents)
	}
	if len(vectorStore.searches) != 2 {
		t.Fatalf("expected two vector search requests, got %#v", vectorStore.searches)
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

type fakeEmbeddingService struct {
	vectors [][]float64
	models  []string
	texts   []string
}

type fakeVectorStore struct {
	upserts       [][]KnowledgeVector
	deleted       [][]string
	searches      []VectorSearchRequest
	searchResults []VectorSearchResult
}

func (s *fakeEmbeddingService) Embed(_ context.Context, model string, texts []string) ([][]float64, error) {
	s.models = append(s.models, model)
	s.texts = append(s.texts, texts...)
	if len(s.vectors) == 0 {
		return nil, nil
	}
	vector := s.vectors[0]
	s.vectors = s.vectors[1:]
	return [][]float64{cloneFloat64Slice(vector)}, nil
}

func (s *fakeVectorStore) Upsert(_ context.Context, vectors []KnowledgeVector) error {
	batch := make([]KnowledgeVector, 0, len(vectors))
	for _, vector := range vectors {
		vector.Embedding = cloneFloat64Slice(vector.Embedding)
		batch = append(batch, vector)
	}
	s.upserts = append(s.upserts, batch)
	return nil
}

func (s *fakeVectorStore) Delete(_ context.Context, ids []string) error {
	s.deleted = append(s.deleted, append([]string(nil), ids...))
	return nil
}

func (s *fakeVectorStore) Search(_ context.Context, request VectorSearchRequest) ([]VectorSearchResult, error) {
	request.Embedding = cloneFloat64Slice(request.Embedding)
	s.searches = append(s.searches, request)
	results := make([]VectorSearchResult, 0, len(s.searchResults))
	for _, result := range s.searchResults {
		result.Vector.Embedding = cloneFloat64Slice(result.Vector.Embedding)
		results = append(results, result)
	}
	return results, nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftSet := make(map[string]int, len(left))
	for _, item := range left {
		leftSet[item]++
	}
	for _, item := range right {
		if leftSet[item] == 0 {
			return false
		}
		leftSet[item]--
	}
	return true
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
