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
	server.Use(ghttp.MiddlewareCORS)
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
		conversationService = newConversationService(stateStore, database)
	}
	ragTraceService := deps.ragTraceService
	if ragTraceService == nil {
		ragTraceService = newRagTraceService(stateStore, database)
	}
	dashboardService := deps.dashboardService
	if dashboardService == nil {
		dashboardService = domaindashboard.NewService(userService, conversationService, ragTraceService)
	}
	knowledgeService := deps.knowledgeService
	if knowledgeService == nil {
		knowledgeService = newKnowledgeService(stateStore, database)
	}
	intentTreeService := deps.intentTreeService
	if intentTreeService == nil {
		intentTreeService = newIntentTreeService(stateStore, database)
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
		queryMappingService = newQueryMappingService(stateStore, database)
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
	bindAPIHandler(server, "/health", handlers.Health)
	bindAPIHandler(server, "/auth/login", authHandler.Login)
	bindAPIHandler(server, "/auth/logout", authHandler.Logout)
	bindAPIHandler(server, "/user/me", authHandler.CurrentUser)
	bindAPIHandler(server, "PUT:/user/password", userAdminHandler.ChangePassword)
	bindAPIHandler(server, "GET:/users", userAdminHandler.Page)
	bindAPIHandler(server, "POST:/users", userAdminHandler.Create)
	bindAPIHandler(server, "PUT:/users/{id}", userAdminHandler.Update)
	bindAPIHandler(server, "DELETE:/users/{id}", userAdminHandler.Delete)
	bindAPIHandler(server, "GET:/admin/dashboard/overview", dashboardHandler.Overview)
	bindAPIHandler(server, "GET:/admin/dashboard/performance", dashboardHandler.Performance)
	bindAPIHandler(server, "GET:/admin/dashboard/trends", dashboardHandler.Trends)
	bindAPIHandler(server, "/rag/settings", settingsHandler.Get)
	bindAPIHandler(server, "/rag/traces/runs", ragTraceHandler.PageRuns)
	bindAPIHandler(server, "GET:/rag/traces/runs/{traceId}", ragTraceHandler.Detail)
	bindAPIHandler(server, "GET:/rag/traces/runs/{traceId}/nodes", ragTraceHandler.Nodes)
	bindAPIHandler(server, "POST:/ingestion/pipelines", ingestionPipelineHandler.Create)
	bindAPIHandler(server, "PUT:/ingestion/pipelines/{id}", ingestionPipelineHandler.Update)
	bindAPIHandler(server, "GET:/ingestion/pipelines/{id}", ingestionPipelineHandler.Get)
	bindAPIHandler(server, "GET:/ingestion/pipelines", ingestionPipelineHandler.Page)
	bindAPIHandler(server, "DELETE:/ingestion/pipelines/{id}", ingestionPipelineHandler.Delete)
	bindAPIHandler(server, "POST:/ingestion/tasks", ingestionTaskHandler.Create)
	bindAPIHandler(server, "POST:/ingestion/tasks/upload", ingestionTaskHandler.Upload)
	bindAPIHandler(server, "GET:/ingestion/tasks/{id}", ingestionTaskHandler.Get)
	bindAPIHandler(server, "GET:/ingestion/tasks/{id}/nodes", ingestionTaskHandler.Nodes)
	bindAPIHandler(server, "GET:/ingestion/tasks", ingestionTaskHandler.Page)
	bindAPIHandler(server, "GET:/knowledge-base/chunk-strategies", knowledgeBaseHandler.ChunkStrategies)
	bindAPIHandler(server, "POST:/knowledge-base", knowledgeBaseHandler.Create)
	bindAPIHandler(server, "PUT:/knowledge-base/{kb-id}", knowledgeBaseHandler.Update)
	bindAPIHandler(server, "DELETE:/knowledge-base/{kb-id}", knowledgeBaseHandler.Delete)
	bindAPIHandler(server, "GET:/knowledge-base/{kb-id}", knowledgeBaseHandler.Get)
	bindAPIHandler(server, "GET:/knowledge-base", knowledgeBaseHandler.Page)
	bindAPIHandler(server, "POST:/knowledge-base/{kb-id}/docs/upload", knowledgeDocumentHandler.Upload)
	bindAPIHandler(server, "POST:/knowledge-base/docs/{doc-id}/chunk", knowledgeDocumentHandler.StartChunk)
	bindAPIHandler(server, "DELETE:/knowledge-base/docs/{doc-id}", knowledgeDocumentHandler.Delete)
	bindAPIHandler(server, "GET:/knowledge-base/docs/{docId}", knowledgeDocumentHandler.Get)
	bindAPIHandler(server, "PUT:/knowledge-base/docs/{docId}", knowledgeDocumentHandler.Update)
	bindAPIHandler(server, "GET:/knowledge-base/{kb-id}/docs", knowledgeDocumentHandler.Page)
	bindAPIHandler(server, "GET:/knowledge-base/docs/search", knowledgeDocumentHandler.Search)
	bindAPIHandler(server, "PATCH:/knowledge-base/docs/{docId}/enable", knowledgeDocumentHandler.Enable)
	bindAPIHandler(server, "GET:/knowledge-base/docs/{docId}/chunk-logs", knowledgeDocumentHandler.ChunkLogs)
	bindAPIHandler(server, "GET:/knowledge-base/docs/{doc-id}/chunks", knowledgeChunkHandler.Page)
	bindAPIHandler(server, "POST:/knowledge-base/docs/{doc-id}/chunks", knowledgeChunkHandler.Create)
	bindAPIHandler(server, "PUT:/knowledge-base/docs/{doc-id}/chunks/{chunk-id}", knowledgeChunkHandler.Update)
	bindAPIHandler(server, "DELETE:/knowledge-base/docs/{doc-id}/chunks/{chunk-id}", knowledgeChunkHandler.Delete)
	bindAPIHandler(server, "PATCH:/knowledge-base/docs/{doc-id}/chunks/{chunk-id}/enable", knowledgeChunkHandler.Enable)
	bindAPIHandler(server, "PATCH:/knowledge-base/docs/{doc-id}/chunks/batch-enable", knowledgeChunkHandler.BatchEnable)
	bindAPIHandler(server, "/intent-tree/trees", intentTreeHandler.Tree)
	bindAPIHandler(server, "POST:/intent-tree", intentTreeHandler.CreateNode)
	bindAPIHandler(server, "PUT:/intent-tree/{id}", intentTreeHandler.UpdateNode)
	bindAPIHandler(server, "DELETE:/intent-tree/{id}", intentTreeHandler.DeleteNode)
	bindAPIHandler(server, "POST:/intent-tree/batch/enable", intentTreeHandler.BatchEnable)
	bindAPIHandler(server, "POST:/intent-tree/batch/disable", intentTreeHandler.BatchDisable)
	bindAPIHandler(server, "POST:/intent-tree/batch/delete", intentTreeHandler.BatchDelete)
	bindAPIHandler(server, "/conversations", conversationHandler.ListConversations)
	bindAPIHandler(server, "PUT:/conversations/{conversationId}", conversationHandler.Rename)
	bindAPIHandler(server, "DELETE:/conversations/{conversationId}", conversationHandler.Delete)
	bindAPIHandler(server, "/conversations/{conversationId}/messages", conversationHandler.ListMessages)
	bindAPIHandler(server, "POST:/conversations/messages/{messageId}/feedback", conversationHandler.SubmitFeedback)
	bindAPIHandler(server, "/mappings", queryMappingHandler.Page)
	bindAPIHandler(server, "GET:/mappings/{id}", queryMappingHandler.Get)
	bindAPIHandler(server, "POST:/mappings", queryMappingHandler.Create)
	bindAPIHandler(server, "PUT:/mappings/{id}", queryMappingHandler.Update)
	bindAPIHandler(server, "DELETE:/mappings/{id}", queryMappingHandler.Delete)
	bindAPIHandler(server, "/rag/sample-questions", sampleQuestionHandler.ListWelcome)
	bindAPIHandler(server, "/sample-questions", sampleQuestionHandler.Page)
	bindAPIHandler(server, "GET:/sample-questions/{id}", sampleQuestionHandler.Get)
	bindAPIHandler(server, "POST:/sample-questions", sampleQuestionHandler.Create)
	bindAPIHandler(server, "PUT:/sample-questions/{id}", sampleQuestionHandler.Update)
	bindAPIHandler(server, "DELETE:/sample-questions/{id}", sampleQuestionHandler.Delete)
	bindAPIHandler(server, "GET:/rag/v3/chat", chatHandler.StreamChat)
	bindAPIHandler(server, "POST:/rag/v3/stop", chatHandler.Stop)
	return server
}

func bindAPIHandler(server *ghttp.Server, pattern string, handler ghttp.HandlerFunc) {
	server.BindHandler(pattern, handler)
	prefixed := prefixAPIRoutePattern(pattern)
	if prefixed != pattern {
		server.BindHandler(prefixed, handler)
	}
}

func prefixAPIRoutePattern(pattern string) string {
	const apiPrefix = "/api/ragent"
	method, path, found := strings.Cut(pattern, ":")
	if !found || !strings.HasPrefix(path, "/") {
		path = pattern
		method = ""
	}
	if path == apiPrefix || strings.HasPrefix(path, apiPrefix+"/") {
		return pattern
	}
	prefixedPath := apiPrefix + path
	if method == "" {
		return prefixedPath
	}
	return method + ":" + prefixedPath
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

func newIntentTreeService(stateStore *platformstate.FileStore, database *sql.DB) *domainintenttree.Service {
	if database != nil {
		repository := platformdb.NewSQLIntentTreeRepository(database)
		if err := repository.Bootstrap(); err == nil {
			return domainintenttree.NewServiceWithRepository(repository)
		}
	}
	return domainintenttree.NewService(stateStore)
}

func newQueryMappingService(stateStore *platformstate.FileStore, database *sql.DB) *domainquerymapping.Service {
	if database != nil {
		repository := platformdb.NewSQLQueryMappingRepository(database)
		if err := repository.Bootstrap(); err == nil {
			return domainquerymapping.NewServiceWithRepository(repository)
		}
	}
	return domainquerymapping.NewService(stateStore)
}

func newConversationService(stateStore *platformstate.FileStore, database *sql.DB) *domainconversation.Service {
	if database != nil {
		repository := platformdb.NewSQLConversationRepository(database)
		if err := repository.Bootstrap(); err == nil {
			return domainconversation.NewServiceWithRepository(repository)
		}
	}
	return domainconversation.NewService(stateStore)
}

func newRagTraceService(stateStore *platformstate.FileStore, database *sql.DB) *domainragtrace.Service {
	if database != nil {
		repository := platformdb.NewSQLRagTraceRepository(database)
		if err := repository.Bootstrap(); err == nil {
			return domainragtrace.NewServiceWithRepository(repository)
		}
	}
	return domainragtrace.NewService(stateStore)
}

func newKnowledgeService(stateStore *platformstate.FileStore, database *sql.DB) *domainknowledge.Service {
	if database != nil {
		repository := platformdb.NewSQLKnowledgeRepository(database)
		if err := repository.Bootstrap(); err == nil {
			return domainknowledge.NewServiceWithRepository(repository)
		}
	}
	return domainknowledge.NewService(stateStore)
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
