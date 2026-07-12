package apitokens

import (
	"errors"
	"time"
)

var (
	ErrTokenNotFound   = errors.New("apitokens: token not found")
	ErrTokenInvalid    = errors.New("apitokens: invalid token")
	ErrTokenRevoked    = errors.New("apitokens: token revoked")
	ErrTokenExpired    = errors.New("apitokens: token expired")
	ErrInvalidInput    = errors.New("apitokens: invalid input")
	ErrScopeNotAllowed = errors.New("apitokens: scope not allowed for user")
)

// Token 是列表/详情用的公开视图（永不含明文）。
type Token struct {
	ID         int64      `json:"id"`
	PublicID   string     `json:"publicId"`
	Name       string     `json:"name"`
	Scopes     []string   `json:"scopes"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	// Prefix 展示用短前缀，便于用户辨认。
	Prefix string `json:"prefix"`
}

// CreatedToken 创建/轮换时一次性返回明文。
type CreatedToken struct {
	Token
	// Plaintext 仅此次响应出现，格式 sft_<publicId>_<secret>
	Plaintext string `json:"token"`
}

type CreateInput struct {
	Name      string
	Scopes    []string
	ExpiresAt *time.Time
}

type Record struct {
	ID         int64
	UserID     int64
	PublicID   string
	TokenHash  string
	Name       string
	Scopes     []string
	LastUsedAt *time.Time
	ExpiresAt  *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}
