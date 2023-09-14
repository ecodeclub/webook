package generator

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenGenerator generates a token for the specified subject.
type TokenGenerator interface {
	GenerateToken(subject string, expire time.Duration) (string, error)
}

// JWTTokenGen generates a JWT token.
type JWTTokenGen struct {
	key     string
	issuer  string
	nowFunc func() time.Time
}

// NewJWTTokenGen creates a JWTTokenGen.
func NewJWTTokenGen(issuer, key string) TokenGenerator {
	return &JWTTokenGen{
		issuer:  issuer,
		key:     key,
		nowFunc: time.Now,
	}
}

// GenerateToken generates a token.
func (t *JWTTokenGen) GenerateToken(subject string, expire time.Duration) (string, error) {
	nowTime := t.nowFunc()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    t.issuer,
		IssuedAt:  jwt.NewNumericDate(nowTime),
		ExpiresAt: jwt.NewNumericDate(nowTime.Add(expire)),
		Subject:   subject,
	})
	return token.SignedString([]byte(t.key))
}
