package config

import "testing"

func TestLoadParsesHTTPPort(t *testing.T) {
	cfg, err := Load("testdata/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTP.Port != 8080 {
		t.Fatalf("expected 8080, got %d", cfg.HTTP.Port)
	}
}

func TestExampleConfigUsesFrontendProxyPort(t *testing.T) {
	cfg, err := Load("../../../configs/config.example.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTP.Port != 9090 {
		t.Fatalf("expected frontend proxy compatible port 9090, got %d", cfg.HTTP.Port)
	}
}

func TestLoadParsesAIWorkflowConfig(t *testing.T) {
	cfg, err := Load("testdata/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AI.Workflow.RetrievalLimit != 6 {
		t.Fatalf("expected retrieval limit 6, got %d", cfg.AI.Workflow.RetrievalLimit)
	}
	if cfg.AI.Workflow.RewritePrompt != "请把问题改写成适合检索的中文查询。" {
		t.Fatalf("unexpected rewrite prompt %s", cfg.AI.Workflow.RewritePrompt)
	}
	if cfg.AI.Workflow.ToolParamPrompt != "请提取工具参数并输出 JSON。" {
		t.Fatalf("unexpected tool param prompt %s", cfg.AI.Workflow.ToolParamPrompt)
	}
	if cfg.AI.Workflow.KnowledgePrompt != "请基于知识上下文回答。" {
		t.Fatalf("unexpected knowledge prompt %s", cfg.AI.Workflow.KnowledgePrompt)
	}
}

func TestLoadParsesRAGRateLimitConfig(t *testing.T) {
	cfg, err := Load("testdata/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RAG.RateLimit.Global.Enabled == nil || !*cfg.RAG.RateLimit.Global.Enabled {
		t.Fatalf("expected enabled rate limit, got %#v", cfg.RAG.RateLimit.Global.Enabled)
	}
	if cfg.RAG.RateLimit.Global.MaxConcurrent != 3 {
		t.Fatalf("expected max concurrent 3, got %d", cfg.RAG.RateLimit.Global.MaxConcurrent)
	}
	if cfg.RAG.RateLimit.Global.MaxWaitSeconds != 5 {
		t.Fatalf("expected max wait seconds 5, got %d", cfg.RAG.RateLimit.Global.MaxWaitSeconds)
	}
}

func TestLoadParsesDatabaseConfig(t *testing.T) {
	cfg, err := Load("testdata/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Database.Driver != "sqlite" {
		t.Fatalf("expected sqlite driver, got %s", cfg.Database.Driver)
	}
	if cfg.Database.DSN != "data/agentrag.db" {
		t.Fatalf("expected database dsn data/agentrag.db, got %s", cfg.Database.DSN)
	}
}
