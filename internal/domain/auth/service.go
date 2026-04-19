package auth

import (
	"errors"
	"strings"

	domainuser "github.com/AmazingCYJ/AgentRAG/internal/domain/user"
	domainusermgmt "github.com/AmazingCYJ/AgentRAG/internal/domain/usermgmt"
	platformauth "github.com/AmazingCYJ/AgentRAG/internal/platform/auth"
	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
)

var (
	// ErrInvalidCredentials 表示用户名或密码不正确。
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	// ErrUnauthorized 表示当前请求没有有效登录态。
	ErrUnauthorized = errors.New("未登录")
	// ErrInvalidCurrentPassword 表示当前密码不正确。
	ErrInvalidCurrentPassword = errors.New("当前密码错误")
	// ErrNewPasswordRequired 表示新密码不能为空。
	ErrNewPasswordRequired = errors.New("新密码不能为空")
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
	userService  *domainusermgmt.Service
	tokenManager *platformauth.TokenManager
}

// NewService 基于配置创建认证服务。
func NewService(cfg appconfig.AuthConfig, userService *domainusermgmt.Service) *Service {
	if userService == nil {
		userService = domainusermgmt.NewService(cfg)
	}

	return &Service{
		userService:  userService,
		tokenManager: platformauth.NewTokenManager(cfg.JWTSecret, cfg.TokenTTL),
	}
}

// Login 校验当前阶段引导账号并签发令牌。
func (s *Service) Login(username, password string) (*LoginResult, error) {
	user, ok := s.userService.Authenticate(username, password)
	if !ok {
		return nil, ErrInvalidCredentials
	}
	token, err := s.tokenManager.Issue(domainuser.User{
		UserID:   user.ID,
		Username: user.Username,
		Password: user.Password,
		Role:     user.Role,
		Avatar:   user.Avatar,
	})
	if err != nil {
		return nil, err
	}
	return &LoginResult{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		Token:    token,
		Avatar:   user.Avatar,
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

// ChangePassword 修改当前登录用户密码。
func (s *Service) ChangePassword(token, currentPassword, newPassword string) error {
	profile, err := s.CurrentUser(token)
	if err != nil {
		return ErrUnauthorized
	}
	err = s.userService.ChangePassword(profile.UserID, currentPassword, newPassword)
	switch {
	case errors.Is(err, domainusermgmt.ErrInvalidCurrentPassword):
		return ErrInvalidCurrentPassword
	case errors.Is(err, domainusermgmt.ErrNewPasswordRequired):
		return ErrNewPasswordRequired
	case errors.Is(err, domainusermgmt.ErrUserNotFound):
		return ErrUnauthorized
	default:
		return err
	}
}
