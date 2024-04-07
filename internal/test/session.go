package test

import (
	"github.com/ecodeclub/ginx/gctx"
	"github.com/ecodeclub/ginx/session"
)

// 初始化一下 session
func init() {
	session.SetDefaultProvider(&SessionProvider{})
}

type SessionProvider struct {
}

func (s *SessionProvider) UpdateClaims(ctx *gctx.Context, claims session.Claims) error {
	return nil
}

func (s *SessionProvider) RenewAccessToken(ctx *gctx.Context) error {
	//TODO implement me
	panic("implement me")
}

func (s *SessionProvider) NewSession(ctx *gctx.Context, uid int64, jwtData map[string]string, sessData map[string]any) (session.Session, error) {
	return nil, nil
}

func (s *SessionProvider) Get(ctx *gctx.Context) (session.Session, error) {
	val, _ := ctx.Get("_session")
	return val.(session.Session), nil
}
