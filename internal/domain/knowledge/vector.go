package knowledge

import "context"

// EmbeddingService 定义文本向量化能力，后续可替换为 OpenAI、Ollama 或其他 provider。
type EmbeddingService interface {
	Embed(ctx context.Context, model string, texts []string) ([][]float64, error)
}

// VectorStore 定义知识库向量写入与检索能力，后续可由 PgVector 或 Milvus 实现。
type VectorStore interface {
	Upsert(ctx context.Context, vectors []KnowledgeVector) error
	Delete(ctx context.Context, ids []string) error
	Search(ctx context.Context, query VectorSearchRequest) ([]VectorSearchResult, error)
}

// KnowledgeVector 表示可写入向量库的一条知识分块向量。
type KnowledgeVector struct {
	ID             string
	KBID           string
	DocID          string
	CollectionName string
	ChunkID        string
	ChunkIndex     int
	Content        string
	EmbeddingModel string
	Embedding      []float64
}

// VectorSearchRequest 表示向量检索请求。
type VectorSearchRequest struct {
	Query          string
	CollectionName string
	EmbeddingModel string
	Embedding      []float64
	Limit          int
}

// VectorSearchResult 表示向量检索结果。
type VectorSearchResult struct {
	Vector KnowledgeVector
	Score  float64
}

type localEmbeddingService struct{}

func (s localEmbeddingService) Embed(_ context.Context, _ string, texts []string) ([][]float64, error) {
	vectors := make([][]float64, 0, len(texts))
	for _, text := range texts {
		vectors = append(vectors, embedText(text))
	}
	return vectors, nil
}

func newLocalEmbeddingService() EmbeddingService {
	return localEmbeddingService{}
}

func embedOne(ctx context.Context, service EmbeddingService, model, text string) []float64 {
	if service == nil {
		service = newLocalEmbeddingService()
	}
	vectors, err := service.Embed(ctx, model, []string{text})
	if err != nil || len(vectors) == 0 {
		return embedText(text)
	}
	return cloneFloat64Slice(vectors[0])
}
