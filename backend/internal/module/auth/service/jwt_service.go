package service

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/han139242/course-recommend-agents/backend/internal/config"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/entity"
)

// Claims 定义 JWT 载荷。
//
// 这里只保留鉴权链路必需的最小身份信息，避免把易变字段直接固化进令牌后造成状态不同步。
type Claims struct {
	UserID int64  `json:"userId"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTService 负责 JWT 签发与校验。
//
// 服务内部统一约束签名密钥、签发方和受众，避免不同调用点自行拼装 token 导致校验规则不一致。
type JWTService struct {
	secret   []byte
	ttl      time.Duration
	issuer   string
	audience string
}

// NewJWTService 创建 JWT 服务。
//
// issuer 与 audience 默认复用应用名，目的是在单体部署场景下保持配置简单，
// 同时让签发端和消费端都能基于同一身份标识完成校验。
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
//
// 这里显式限制签名算法、签发方和受众，防止仅凭密钥校验通过却接受到非预期来源的令牌。
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
