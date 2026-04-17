package httpx

import (
	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	domainchat "github.com/AmazingCYJ/AgentRAG/internal/domain/chat"
	domainconversation "github.com/AmazingCYJ/AgentRAG/internal/domain/conversation"
	domainsamplequestion "github.com/AmazingCYJ/AgentRAG/internal/domain/samplequestion"
	domainsettings "github.com/AmazingCYJ/AgentRAG/internal/domain/settings"
	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/httpx/handlers"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type serverDeps struct {
	authService           *domainauth.Service
	chatService           *domainchat.Service
	conversationService   *domainconversation.Service
	settingsService       *domainsettings.Service
	sampleQuestionService *domainsamplequestion.Service
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
	authService := deps.authService
	if authService == nil {
		authService = domainauth.NewService(cfg.Auth)
	}
	conversationService := deps.conversationService
	if conversationService == nil {
		conversationService = domainconversation.NewService()
	}
	chatService := deps.chatService
	if chatService == nil {
		chatService = domainchat.NewService(conversationService)
	}
	settingsService := deps.settingsService
	if settingsService == nil {
		settingsService = domainsettings.NewService()
	}
	sampleQuestionService := deps.sampleQuestionService
	if sampleQuestionService == nil {
		sampleQuestionService = domainsamplequestion.NewService()
	}
	authHandler := handlers.NewAuthHandler(authService)
	chatHandler := handlers.NewChatHandler(authService, chatService)
	conversationHandler := handlers.NewConversationHandler(authService, conversationService)
	settingsHandler := handlers.NewSettingsHandler(authService, settingsService)
	sampleQuestionHandler := handlers.NewSampleQuestionHandler(authService, sampleQuestionService)
	server.BindHandler("/health", handlers.Health)
	server.BindHandler("/auth/login", authHandler.Login)
	server.BindHandler("/auth/logout", authHandler.Logout)
	server.BindHandler("/user/me", authHandler.CurrentUser)
	server.BindHandler("/rag/settings", settingsHandler.Get)
	server.BindHandler("/conversations", conversationHandler.ListConversations)
	server.BindHandler("PUT:/conversations/{conversationId}", conversationHandler.Rename)
	server.BindHandler("DELETE:/conversations/{conversationId}", conversationHandler.Delete)
	server.BindHandler("/conversations/{conversationId}/messages", conversationHandler.ListMessages)
	server.BindHandler("POST:/conversations/messages/{messageId}/feedback", conversationHandler.SubmitFeedback)
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
