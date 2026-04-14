package auth

import (
	"errors"
	"strings"

	domainuser "github.com/AmazingCYJ/AgentRAG/internal/domain/user"
	platformauth "github.com/AmazingCYJ/AgentRAG/internal/platform/auth"
	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
)

var (
	// ErrInvalidCredentials 表示用户名或密码不正确。
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	// ErrUnauthorized 表示当前请求没有有效登录态。
	ErrUnauthorized = errors.New("未登录")
)

// LoginResult 定义登录成功后的返回载荷。
type LoginResult struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Token    string `json:"token"`
	Avatar   string `json:"avatar,omitempty"`
}

// Service 提供当前阶段最小可用的认证能力。
type Service struct {
	bootstrap    domainuser.User
	tokenManager *platformauth.TokenManager
}

// NewService 基于配置创建认证服务。
func NewService(cfg appconfig.AuthConfig) *Service {
	bootstrap := domainuser.User{
		UserID:   cfg.Bootstrap.UserID,
		Username: cfg.Bootstrap.Username,
		Password: cfg.Bootstrap.Password,
		Role:     cfg.Bootstrap.Role,
		Avatar:   cfg.Bootstrap.Avatar,
	}
	if bootstrap.UserID == "" {
		bootstrap.UserID = "u_admin"
	}
	if bootstrap.Username == "" {
		bootstrap.Username = "admin"
	}
	if bootstrap.Password == "" {
		bootstrap.Password = "admin123"
	}
	if bootstrap.Role == "" {
		bootstrap.Role = "admin"
	}

	return &Service{
		bootstrap:    bootstrap,
		tokenManager: platformauth.NewTokenManager(cfg.JWTSecret, cfg.TokenTTL),
	}
}

// Login 校验当前阶段引导账号并签发令牌。
func (s *Service) Login(username, password string) (*LoginResult, error) {
	if !strings.EqualFold(strings.TrimSpace(username), s.bootstrap.Username) || password != s.bootstrap.Password {
		return nil, ErrInvalidCredentials
	}
	token, err := s.tokenManager.Issue(s.bootstrap)
	if err != nil {
		return nil, err
	}
	return &LoginResult{
		UserID:   s.bootstrap.UserID,
		Username: s.bootstrap.Username,
		Role:     s.bootstrap.Role,
		Token:    token,
		Avatar:   s.bootstrap.Avatar,
	}, nil
}

// CurrentUser 根据令牌返回当前用户资料。
func (s *Service) CurrentUser(token string) (*domainuser.Profile, error) {
	claims, err := s.tokenManager.Parse(strings.TrimSpace(strings.TrimPrefix(token, "Bearer ")))
	if err != nil {
		return nil, ErrUnauthorized
	}
	profile := domainuser.Profile{
		UserID:   claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
		Avatar:   claims.Avatar,
	}
	return &profile, nil
}
