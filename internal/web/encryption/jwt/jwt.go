package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ecodeclub/webook/internal/web/encryption"
)

type TokenClaims struct {
	jwt.RegisteredClaims
	// 这是一个前端采集了用户的登录环境生成的一个码
	Fingerprint string
	//用于查找用户信息的一个字段
	Id int64
}

type Jwt struct {
	//secretCode string
}

func NewJwt() encryption.Handle {
	return &Jwt{}
}

func (j *Jwt) Encryption(arg map[string]string, secretCode string, duration time.Duration) (encryptString string, err error) {
	now := time.Now()
	fingerprint, ok := arg["fingerprint"]
	if !ok {
		return "", errors.New("参数缺失")
	}
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
		Fingerprint: fingerprint,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)

	encryptString, err = token.SignedString([]byte(secretCode))
	if err != nil {
		return "", nil
	}
	return encryptString, nil
}

func (j *Jwt) Decrypt(tokenStr string, secretCode string) (interface{}, error) {
	claims := &TokenClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretCode), nil
	})
	if err != nil {
		return nil, err
	}
	if token == nil || !token.Valid {
		//解析成功  但是 token 以及 claims 不一定合法
		return nil, errors.New("不合法操作")
	}

	return claims, nil
}
