package validator

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Verifier interface {
	Verify(token string) (string, error)
}

type JWTTokenVerifier struct {
	Key     string
	nowFunc func() time.Time
}

func NewJWTTokenVerifier(key string) Verifier {
	return &JWTTokenVerifier{
		Key: key,
		nowFunc: func() time.Time {
			return time.Now()
		},
	}
}

func (j *JWTTokenVerifier) Verify(token string) (string, error) {
	t, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(j.Key), nil
		},
		jwt.WithTimeFunc(func() time.Time {
			return j.nowFunc()
		}),
	)
	if err != nil {
		return "", fmt.Errorf("cannot parse token: %v", err)
	}

	clm, ok := t.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return "", fmt.Errorf("token claim is not RegisteredClaims")
	}

	if !t.Valid {
		return "", fmt.Errorf("token not valid")
	}

	return clm.Subject, nil
}
