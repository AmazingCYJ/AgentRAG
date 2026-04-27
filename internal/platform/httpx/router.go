package httpx

import (
	"context"
	"database/sql"
	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	domainchat "github.com/AmazingCYJ/AgentRAG/internal/domain/chat"
	domainconversation "github.com/AmazingCYJ/AgentRAG/internal/domain/conversation"
	domaindashboard "github.com/AmazingCYJ/AgentRAG/internal/domain/dashboard"
	domainingestion "github.com/AmazingCYJ/AgentRAG/internal/domain/ingestion"
	domainintenttree "github.com/AmazingCYJ/AgentRAG/internal/domain/intenttree"
	domainknowledge "github.com/AmazingCYJ/AgentRAG/internal/domain/knowledge"
	domainquerymapping "github.com/AmazingCYJ/AgentRAG/internal/domain/querymapping"
	domainragtrace "github.com/AmazingCYJ/AgentRAG/internal/domain/ragtrace"
	domainsamplequestion "github.com/AmazingCYJ/AgentRAG/internal/domain/samplequestion"
	domainsettings "github.com/AmazingCYJ/AgentRAG/internal/domain/settings"
	domainusermgmt "github.com/AmazingCYJ/AgentRAG/internal/domain/usermgmt"
	"github.com/AmazingCYJ/AgentRAG/internal/mcpserver"
	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	platformdb "github.com/AmazingCYJ/AgentRAG/internal/platform/db"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/httpx/handlers"
	platformstate "github.com/AmazingCYJ/AgentRAG/internal/platform/state"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"strings"
)

type serverDeps struct {
	authService           *domainauth.Service
	chatService           *domainchat.Service
	conversationService   *domainconversation.Service
	dashboardService      *domaindashboard.Service
	intentTreeService     *domainintenttree.Service
	ingestionService      *domainingestion.Service
	knowledgeService      *domainknowledge.Service
	queryMappingService   *domainquerymapping.Service
	ragTraceService       *domainragtrace.Service
	settingsService       *domainsettings.Service
	sampleQuestionService *domainsamplequestion.Service
	userService           *domainusermgmt.Service
}

// NewServer 构造当前阶段的最小 HTTP 服务，并注册基础路由。
func NewServer(cfg *appconfig.Config) *ghttp.Server {
	return newServerWithDeps(cfg, "agentrag-api", serverDeps{})
}

func newServer(cfg *appconfig.Config, name string) *ghttp.Server {
	return newServerWithDeps(cfg, name, serverDeps{})
}

func newServerWithDeps(cfg *appconfig.Config, name string, deps serverDeps) *ghttp.Server {
	server := g.Server(name)
	server.SetDumpRouterMap(false)
	if cfg != nil && cfg.HTTP.Port > 0 {
		server.SetPort(cfg.HTTP.Port)
	}
	var stateStore *platformstate.FileStore
	if cfg != nil && strings.TrimSpace(cfg.State.File) != "" {
		if store, err := platformstate.NewFileStore(cfg.State.File); err == nil {
			stateStore = store
		}
	}
	database := openConfiguredDatabase(cfg)
	userService := deps.userService
	if userService == nil {
		userService = newUserService(cfg, stateStore, database)
	}
	authService := deps.authService
	if authService == nil {
		authService = domainauth.NewService(cfg.Auth, userService)
	}
	conversationService := deps.conversationService
	if conversationService == nil {
		conversationService = domainconversation.NewService(stateStore)
	}
	ragTraceService := deps.ragTraceService
	if ragTraceService == nil {
		ragTraceService = domainragtrace.NewService(stateStore)
	}
	dashboardService := deps.dashboardService
	if dashboardService == nil {
		dashboardService = domaindashboard.NewService(userService, conversationService, ragTraceService)
	}
	knowledgeService := deps.knowledgeService
	if knowledgeService == nil {
		knowledgeService = domainknowledge.NewService(stateStore)
	}
	intentTreeService := deps.intentTreeService
	if intentTreeService == nil {
		intentTreeService = domainintenttree.NewService(stateStore)
	}
	chatService := deps.chatService
	if chatService == nil {
		mcpRegistry := mcpserver.NewRegistry()
		paramExtractor := domainchat.BuildEinoToolParamExtractor(cfg.AI)
		routeResolver := func(_ context.Context, question string) (domainchat.RouteDecision, error) {
			hint := intentTreeService.MatchQuestion(question)
			if hint.Score <= 0 {
				return domainchat.RouteDecision{Kind: domainchat.RouteKindNone}, nil
			}
			if hint.Kind == 2 && strings.TrimSpace(hint.ToolID) != "" {
				return domainchat.RouteDecision{
					Kind:   domainchat.RouteKindTool,
					ToolID: hint.ToolID,
				}, nil
			}
			return domainchat.RouteDecision{Kind: domainchat.RouteKindKnowledge}, nil
		}
		toolCaller := func(_ context.Context, toolID, question string) (string, error) {
			executor, ok := mcpRegistry.GetExecutor(toolID)
			if !ok {
				return "", nil
			}
			arguments := buildMCPToolArguments(toolID, question)
			if paramExtractor != nil {
				if extracted, err := paramExtractor(context.Background(), domainchat.ToChatToolSchema(executor.Definition()), question); err == nil && len(extracted) > 0 {
					arguments = extracted
				}
			}
			result := executor.Execute(mcpserver.ToolRequest{
				ToolID:     toolID,
				Parameters: arguments,
			})
			if !result.Success {
				return result.ErrorMessage, nil
			}
			return result.TextResult, nil
		}
		chatService = domainchat.NewService(
			conversationService,
			ragTraceService,
			domainchat.NewGeneratorFromConfig(cfg.AI, knowledgeService.BuildPromptContext, routeResolver, toolCaller),
		)
	}
	ingestionService := deps.ingestionService
	if ingestionService == nil {
		ingestionService = domainingestion.NewService(stateStore)
	}
	queryMappingService := deps.queryMappingService
	if queryMappingService == nil {
		queryMappingService = domainquerymapping.NewService(stateStore)
	}
	settingsService := deps.settingsService
	if settingsService == nil {
		settingsService = domainsettings.NewService()
	}
	sampleQuestionService := deps.sampleQuestionService
	if sampleQuestionService == nil {
		sampleQuestionService = newSampleQuestionService(stateStore, database)
	}
	authHandler := handlers.NewAuthHandler(authService)
	chatHandler := handlers.NewChatHandler(authService, chatService)
	conversationHandler := handlers.NewConversationHandler(authService, conversationService)
	dashboardHandler := handlers.NewDashboardHandler(authService, dashboardService)
	intentTreeHandler := handlers.NewIntentTreeHandler(authService, intentTreeService)
	ingestionPipelineHandler := handlers.NewIngestionPipelineHandler(authService, ingestionService)
	ingestionTaskHandler := handlers.NewIngestionTaskHandler(authService, ingestionService)
	knowledgeBaseHandler := handlers.NewKnowledgeBaseHandler(authService, knowledgeService)
	knowledgeDocumentHandler := handlers.NewKnowledgeDocumentHandler(authService, knowledgeService)
	knowledgeChunkHandler := handlers.NewKnowledgeChunkHandler(authService, knowledgeService)
	queryMappingHandler := handlers.NewQueryMappingHandler(authService, queryMappingService)
	ragTraceHandler := handlers.NewRagTraceHandler(authService, ragTraceService)
	settingsHandler := handlers.NewSettingsHandler(authService, settingsService)
	sampleQuestionHandler := handlers.NewSampleQuestionHandler(authService, sampleQuestionService)
	userAdminHandler := handlers.NewUserAdminHandler(authService, userService)
	server.BindHandler("/health", handlers.Health)
	server.BindHandler("/auth/login", authHandler.Login)
	server.BindHandler("/auth/logout", authHandler.Logout)
	server.BindHandler("/user/me", authHandler.CurrentUser)
	server.BindHandler("PUT:/user/password", userAdminHandler.ChangePassword)
	server.BindHandler("GET:/users", userAdminHandler.Page)
	server.BindHandler("POST:/users", userAdminHandler.Create)
	server.BindHandler("PUT:/users/{id}", userAdminHandler.Update)
	server.BindHandler("DELETE:/users/{id}", userAdminHandler.Delete)
	server.BindHandler("GET:/admin/dashboard/overview", dashboardHandler.Overview)
	server.BindHandler("GET:/admin/dashboard/performance", dashboardHandler.Performance)
	server.BindHandler("GET:/admin/dashboard/trends", dashboardHandler.Trends)
	server.BindHandler("/rag/settings", settingsHandler.Get)
	server.BindHandler("/rag/traces/runs", ragTraceHandler.PageRuns)
	server.BindHandler("GET:/rag/traces/runs/{traceId}", ragTraceHandler.Detail)
	server.BindHandler("GET:/rag/traces/runs/{traceId}/nodes", ragTraceHandler.Nodes)
	server.BindHandler("POST:/ingestion/pipelines", ingestionPipelineHandler.Create)
	server.BindHandler("PUT:/ingestion/pipelines/{id}", ingestionPipelineHandler.Update)
	server.BindHandler("GET:/ingestion/pipelines/{id}", ingestionPipelineHandler.Get)
	server.BindHandler("GET:/ingestion/pipelines", ingestionPipelineHandler.Page)
	server.BindHandler("DELETE:/ingestion/pipelines/{id}", ingestionPipelineHandler.Delete)
	server.BindHandler("POST:/ingestion/tasks", ingestionTaskHandler.Create)
	server.BindHandler("POST:/ingestion/tasks/upload", ingestionTaskHandler.Upload)
	server.BindHandler("GET:/ingestion/tasks/{id}", ingestionTaskHandler.Get)
	server.BindHandler("GET:/ingestion/tasks/{id}/nodes", ingestionTaskHandler.Nodes)
	server.BindHandler("GET:/ingestion/tasks", ingestionTaskHandler.Page)
	server.BindHandler("GET:/knowledge-base/chunk-strategies", knowledgeBaseHandler.ChunkStrategies)
	server.BindHandler("POST:/knowledge-base", knowledgeBaseHandler.Create)
	server.BindHandler("PUT:/knowledge-base/{kb-id}", knowledgeBaseHandler.Update)
	server.BindHandler("DELETE:/knowledge-base/{kb-id}", knowledgeBaseHandler.Delete)
	server.BindHandler("GET:/knowledge-base/{kb-id}", knowledgeBaseHandler.Get)
	server.BindHandler("GET:/knowledge-base", knowledgeBaseHandler.Page)
	server.BindHandler("POST:/knowledge-base/{kb-id}/docs/upload", knowledgeDocumentHandler.Upload)
	server.BindHandler("POST:/knowledge-base/docs/{doc-id}/chunk", knowledgeDocumentHandler.StartChunk)
	server.BindHandler("DELETE:/knowledge-base/docs/{doc-id}", knowledgeDocumentHandler.Delete)
	server.BindHandler("GET:/knowledge-base/docs/{docId}", knowledgeDocumentHandler.Get)
	server.BindHandler("PUT:/knowledge-base/docs/{docId}", knowledgeDocumentHandler.Update)
	server.BindHandler("GET:/knowledge-base/{kb-id}/docs", knowledgeDocumentHandler.Page)
	server.BindHandler("GET:/knowledge-base/docs/search", knowledgeDocumentHandler.Search)
	server.BindHandler("PATCH:/knowledge-base/docs/{docId}/enable", knowledgeDocumentHandler.Enable)
	server.BindHandler("GET:/knowledge-base/docs/{docId}/chunk-logs", knowledgeDocumentHandler.ChunkLogs)
	server.BindHandler("GET:/knowledge-base/docs/{doc-id}/chunks", knowledgeChunkHandler.Page)
	server.BindHandler("POST:/knowledge-base/docs/{doc-id}/chunks", knowledgeChunkHandler.Create)
	server.BindHandler("PUT:/knowledge-base/docs/{doc-id}/chunks/{chunk-id}", knowledgeChunkHandler.Update)
	server.BindHandler("DELETE:/knowledge-base/docs/{doc-id}/chunks/{chunk-id}", knowledgeChunkHandler.Delete)
	server.BindHandler("PATCH:/knowledge-base/docs/{doc-id}/chunks/{chunk-id}/enable", knowledgeChunkHandler.Enable)
	server.BindHandler("PATCH:/knowledge-base/docs/{doc-id}/chunks/batch-enable", knowledgeChunkHandler.BatchEnable)
	server.BindHandler("/intent-tree/trees", intentTreeHandler.Tree)
	server.BindHandler("POST:/intent-tree", intentTreeHandler.CreateNode)
	server.BindHandler("PUT:/intent-tree/{id}", intentTreeHandler.UpdateNode)
	server.BindHandler("DELETE:/intent-tree/{id}", intentTreeHandler.DeleteNode)
	server.BindHandler("POST:/intent-tree/batch/enable", intentTreeHandler.BatchEnable)
	server.BindHandler("POST:/intent-tree/batch/disable", intentTreeHandler.BatchDisable)
	server.BindHandler("POST:/intent-tree/batch/delete", intentTreeHandler.BatchDelete)
	server.BindHandler("/conversations", conversationHandler.ListConversations)
	server.BindHandler("PUT:/conversations/{conversationId}", conversationHandler.Rename)
	server.BindHandler("DELETE:/conversations/{conversationId}", conversationHandler.Delete)
	server.BindHandler("/conversations/{conversationId}/messages", conversationHandler.ListMessages)
	server.BindHandler("POST:/conversations/messages/{messageId}/feedback", conversationHandler.SubmitFeedback)
	server.BindHandler("/mappings", queryMappingHandler.Page)
	server.BindHandler("GET:/mappings/{id}", queryMappingHandler.Get)
	server.BindHandler("POST:/mappings", queryMappingHandler.Create)
	server.BindHandler("PUT:/mappings/{id}", queryMappingHandler.Update)
	server.BindHandler("DELETE:/mappings/{id}", queryMappingHandler.Delete)
	server.BindHandler("/rag/sample-questions", sampleQuestionHandler.ListWelcome)
	server.BindHandler("/sample-questions", sampleQuestionHandler.Page)
	server.BindHandler("GET:/sample-questions/{id}", sampleQuestionHandler.Get)
	server.BindHandler("POST:/sample-questions", sampleQuestionHandler.Create)
	server.BindHandler("PUT:/sample-questions/{id}", sampleQuestionHandler.Update)
	server.BindHandler("DELETE:/sample-questions/{id}", sampleQuestionHandler.Delete)
	server.BindHandler("GET:/rag/v3/chat", chatHandler.StreamChat)
	server.BindHandler("POST:/rag/v3/stop", chatHandler.Stop)
	return server
}

func openConfiguredDatabase(cfg *appconfig.Config) *sql.DB {
	if cfg == nil {
		return nil
	}
	database, err := platformdb.OpenDatabase(platformdb.Config{
		Driver: cfg.Database.Driver,
		DSN:    cfg.Database.DSN,
	})
	if err != nil {
		return nil
	}
	return database
}

func newUserService(cfg *appconfig.Config, stateStore *platformstate.FileStore, database *sql.DB) *domainusermgmt.Service {
	if cfg == nil {
		return domainusermgmt.NewService(appconfig.AuthConfig{}, stateStore)
	}
	if database != nil {
		repository := platformdb.NewSQLUserRepository(database)
		if err := repository.Bootstrap(); err == nil {
			return domainusermgmt.NewServiceWithRepository(cfg.Auth, repository)
		}
	}
	return domainusermgmt.NewService(cfg.Auth, stateStore)
}

func newSampleQuestionService(stateStore *platformstate.FileStore, database *sql.DB) *domainsamplequestion.Service {
	if database != nil {
		repository := platformdb.NewSQLSampleQuestionRepository(database)
		if err := repository.Bootstrap(); err == nil {
			return domainsamplequestion.NewServiceWithRepository(repository)
		}
	}
	return domainsamplequestion.NewService(stateStore)
}

func buildMCPToolArguments(toolID, question string) map[string]any {
	args := map[string]any{}
	switch toolID {
	case "weather_query":
		args["city"] = detectCity(question)
		if strings.Contains(question, "预报") || strings.Contains(question, "明天") || strings.Contains(question, "后天") || strings.Contains(question, "未来") {
			args["queryType"] = "forecast"
		} else {
			args["queryType"] = "current"
		}
	case "ticket_query":
		args["region"] = detectRegion(question)
		if strings.Contains(question, "列表") {
			args["queryType"] = "list"
		} else if strings.Contains(question, "统计") || strings.Contains(question, "分析") {
			args["queryType"] = "stats"
		} else {
			args["queryType"] = "summary"
		}
	case "sales_query":
		args["region"] = detectRegion(question)
		if strings.Contains(question, "排名") {
			args["queryType"] = "ranking"
		} else if strings.Contains(question, "趋势") {
			args["queryType"] = "trend"
		} else if strings.Contains(question, "明细") {
			args["queryType"] = "detail"
		} else {
			args["queryType"] = "summary"
		}
		args["period"] = detectPeriod(question)
	}
	return args
}

func detectCity(question string) string {
	for _, city := range []string{"北京", "上海", "广州", "深圳", "杭州", "成都", "武汉", "南京", "西安", "重庆", "长沙", "天津", "苏州", "郑州", "青岛", "大连", "厦门", "昆明", "哈尔滨", "三亚"} {
		if strings.Contains(question, city) {
			return city
		}
	}
	return "北京"
}

func detectRegion(question string) string {
	for _, region := range []string{"华东", "华南", "华北", "西南", "西北"} {
		if strings.Contains(question, region) {
			return region
		}
	}
	return "华东"
}

func detectPeriod(question string) string {
	for _, period := range []string{"本月", "上月", "本季度", "上季度", "本年"} {
		if strings.Contains(question, period) {
			return period
		}
	}
	return "本月"
}
