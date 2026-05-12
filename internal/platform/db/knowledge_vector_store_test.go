package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	domainknowledge "github.com/AmazingCYJ/AgentRAG/internal/domain/knowledge"
	_ "modernc.org/sqlite"
)

func TestSQLKnowledgeVectorStoreSearchesAndDeletesVectors(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	store := NewSQLKnowledgeVectorStore(database)
	store.now = func() time.Time { return time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC) }
	if err := store.Bootstrap(); err != nil {
		t.Fatalf("bootstrap vector store failed: %v", err)
	}

	ctx := context.Background()
	if err := store.Upsert(ctx, []domainknowledge.KnowledgeVector{
		{
			ID:             "vec_invoice",
			KBID:           "kb_finance",
			DocID:          "doc_finance",
			CollectionName: "finance_docs",
			ChunkID:        "chunk_invoice",
			ChunkIndex:     0,
			Content:        "报销发票要求",
			EmbeddingModel: "embedding-local",
			Embedding:      []float64{1, 0, 0},
		},
		{
			ID:             "vec_leave",
			KBID:           "kb_oa",
			DocID:          "doc_leave",
			CollectionName: "oa_docs",
			ChunkID:        "chunk_leave",
			ChunkIndex:     1,
			Content:        "请假流程",
			EmbeddingModel: "embedding-local",
			Embedding:      []float64{0, 1, 0},
		},
	}); err != nil {
		t.Fatalf("upsert vectors failed: %v", err)
	}

	results, err := store.Search(ctx, domainknowledge.VectorSearchRequest{
		CollectionName: "finance_docs",
		EmbeddingModel: "embedding-local",
		Embedding:      []float64{1, 0, 0},
		Limit:          5,
	})
	if err != nil {
		t.Fatalf("search vectors failed: %v", err)
	}
	if len(results) != 1 || results[0].Vector.ID != "vec_invoice" || results[0].Score != 1 {
		t.Fatalf("unexpected search results %#v", results)
	}

	if err := store.Upsert(ctx, []domainknowledge.KnowledgeVector{
		{
			ID:             "vec_invoice",
			KBID:           "kb_finance",
			DocID:          "doc_finance",
			CollectionName: "finance_docs",
			ChunkID:        "chunk_invoice",
			ChunkIndex:     2,
			Content:        "报销发票更新",
			EmbeddingModel: "embedding-local",
			Embedding:      []float64{0, 0, 1},
		},
	}); err != nil {
		t.Fatalf("update vector failed: %v", err)
	}
	results, err = store.Search(ctx, domainknowledge.VectorSearchRequest{
		CollectionName: "finance_docs",
		EmbeddingModel: "embedding-local",
		Embedding:      []float64{0, 0, 1},
		Limit:          1,
	})
	if err != nil {
		t.Fatalf("search updated vector failed: %v", err)
	}
	if len(results) != 1 || results[0].Vector.ChunkIndex != 2 || results[0].Vector.Content != "报销发票更新" {
		t.Fatalf("unexpected updated vector result %#v", results)
	}

	if err := store.Delete(ctx, []string{"vec_invoice"}); err != nil {
		t.Fatalf("delete vector failed: %v", err)
	}
	results, err = store.Search(ctx, domainknowledge.VectorSearchRequest{
		CollectionName: "finance_docs",
		EmbeddingModel: "embedding-local",
		Embedding:      []float64{0, 0, 1},
		Limit:          1,
	})
	if err != nil {
		t.Fatalf("search after delete failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected deleted vector to be absent, got %#v", results)
	}
}
