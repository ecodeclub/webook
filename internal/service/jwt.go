package service

import "github.com/golang-jwt/jwt/v5"

type EmailClaims struct {
	Email string
	jwt.RegisteredClaims
}

var JWTKey = []byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0")
