package service

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/han139242/course-recommend-agents/backend/internal/config"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/entity"
)

// Claims 定义 JWT 载荷。
type Claims struct {
	UserID int64  `json:"userId"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTService 负责 JWT 签发与校验。
type JWTService struct {
	secret   []byte
	ttl      time.Duration
	issuer   string
	audience string
}

// NewJWTService 创建 JWT 服务。
func NewJWTService(cfg config.AuthConfig, appCfg config.AppConfig) *JWTService {
	return &JWTService{
		secret:   []byte(cfg.JWTSecret),
		ttl:      cfg.TokenTTL,
		issuer:   appCfg.Name,
		audience: appCfg.Name,
	}
}

// Sign 为用户签发访问令牌。
// 当前 claim 只保留鉴权链路所需的最小字段，避免把可频繁变更或敏感的信息长期放入 token。
func (s *JWTService) Sign(user *entity.User) (string, time.Time, error) {
	expiredAt := time.Now().Add(s.ttl)
	issuedAt := time.Now()
	claims := Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiredAt),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			Subject:   fmt.Sprintf("%d", user.ID),
			Issuer:    s.issuer,
			Audience:  jwt.ClaimStrings{s.audience},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signedToken, expiredAt, nil
}

// Parse 解析并校验 JWT。
func (s *JWTService) Parse(rawToken string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(rawToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("非法签名算法: %s", token.Method.Alg())
		}
		return s.secret, nil
	}, jwt.WithIssuer(s.issuer), jwt.WithAudience(s.audience))
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}
