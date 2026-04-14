package auth

import (
	"errors"
	"time"

	domainuser "github.com/AmazingCYJ/AgentRAG/internal/domain/user"
	"github.com/golang-jwt/jwt/v5"
)

// Claims 定义当前系统使用的最小 JWT 载荷。
type Claims struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Avatar   string `json:"avatar,omitempty"`
	jwt.RegisteredClaims
}

// TokenManager 负责签发和解析 JWT。
type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenManager 创建 JWT 管理器。
func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	if secret == "" {
		secret = "agentrag-dev-secret"
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// Issue 为指定用户签发访问令牌。
func (m *TokenManager) Issue(user domainuser.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   user.UserID,
		Username: user.Username,
		Role:     user.Role,
		Avatar:   user.Avatar,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Parse 解析并校验访问令牌。
func (m *TokenManager) Parse(token string) (*Claims, error) {
	if token == "" {
		return nil, errors.New("empty token")
	}
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
