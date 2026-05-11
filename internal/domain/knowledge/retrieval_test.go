package knowledge

import (
	"math"
	"testing"
)

func TestScoreKnowledgeMatchRequiresLexicalHit(t *testing.T) {
	queryTokens := tokenize("报销流程")
	queryEmbedding := embedTokens(queryTokens)

	score := scoreKnowledgeMatch(queryTokens, queryEmbedding, "账号登录和权限配置")
	if score.matched() {
		t.Fatalf("expected lexical gate to reject unrelated candidate, got %#v", score)
	}
}

func TestCompareRetrievalScorePrioritizesLexicalMatchCount(t *testing.T) {
	strongLexical := retrievalScore{lexical: 2, vector: -1}
	strongVector := retrievalScore{lexical: 1, vector: 1}

	if compareRetrievalScore(strongLexical, strongVector) <= 0 {
		t.Fatalf("expected lexical score to dominate vector score")
	}
}

func TestLocalEmbeddingIsDeterministicAndNormalized(t *testing.T) {
	left := embedText("报销流程 发票")
	right := embedText("报销流程 发票")
	if len(left) != localEmbeddingDimensions || len(right) != localEmbeddingDimensions {
		t.Fatalf("unexpected embedding dimensions left=%d right=%d", len(left), len(right))
	}
	if similarity := cosineSimilarity(left, right); math.Abs(similarity-1) > 0.000001 {
		t.Fatalf("expected deterministic embedding similarity 1, got %f", similarity)
	}
}

func TestVectorScorePrefersFocusedCandidateWhenLexicalScoresTie(t *testing.T) {
	queryTokens := tokenize("报销流程 发票")
	queryEmbedding := embedTokens(queryTokens)
	noisy := scoreKnowledgeMatch(queryTokens, queryEmbedding, "报销流程 发票 账号 密码 登录 审批人 通知 配置 表单 字段 权限 菜单 角色 同步 导出")
	focused := scoreKnowledgeMatch(queryTokens, queryEmbedding, "报销流程 发票")

	if noisy.lexical != focused.lexical {
		t.Fatalf("expected lexical scores to tie, noisy=%#v focused=%#v", noisy, focused)
	}
	if compareRetrievalScore(focused, noisy) <= 0 {
		t.Fatalf("expected vector score to prefer focused candidate, noisy=%#v focused=%#v", noisy, focused)
	}
}
