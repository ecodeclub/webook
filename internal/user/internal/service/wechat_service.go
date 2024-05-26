package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ekit/net/httpx"
	"github.com/ecodeclub/webook/internal/user/internal/domain"
	"github.com/gotomicro/ego/core/elog"
	uuid "github.com/lithammer/shortuuid/v4"
)

const authURLPattern = "https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s#wechat_redire"

type AuthQueryParams struct {
	InviterCode string `json:"invitation,omitempty"`
}

type CallbackQueryParams struct {
	Code  string
	State string
}

type OAuth2Service interface {
	AuthURL(ctx context.Context, a AuthQueryParams) (string, error)
	Verify(ctx context.Context, c CallbackQueryParams) (domain.WechatInfo, error)
}

type WechatOAuth2Service struct {
	cache       ecache.Cache
	appId       string
	appSecret   string
	redirectURL string
	logger      *elog.Component
	client      *http.Client
}

func NewWechatService(cache ecache.Cache, appId, appSecret, redirectURL string) OAuth2Service {
	return &WechatOAuth2Service{
		cache:       cache,
		redirectURL: url.PathEscape(redirectURL),
		logger:      elog.DefaultLogger,
		client:      http.DefaultClient,
		appId:       appId,
		appSecret:   appSecret,
	}
}

func (s *WechatOAuth2Service) AuthURL(ctx context.Context, a AuthQueryParams) (string, error) {
	return fmt.Sprintf(authURLPattern, s.appId, s.redirectURL, s.getState(ctx, a)), nil
}

func (s *WechatOAuth2Service) getState(ctx context.Context, a AuthQueryParams) string {
	state := uuid.New()
	if a.InviterCode != "" {
		// 尽最大努力建立映射但不阻碍主流程
		data, err := json.Marshal(a)
		if err != nil {
			return ""
		}
		err = s.cache.Set(ctx, state, string(data), time.Minute*5)
		if err != nil {
			return ""
		}
	}
	return state
}

func (s *WechatOAuth2Service) Verify(ctx context.Context, c CallbackQueryParams) (domain.WechatInfo, error) {
	const baseURL = "https://api.weixin.qq.com/sns/oauth2/access_token"
	var res Result
	err := httpx.NewRequest(ctx, http.MethodGet, baseURL).
		Client(s.client).
		AddParam("appid", s.appId).
		AddParam("secret", s.appSecret).AddParam("code", c.Code).
		AddParam("grant_type", "authorization_code").Do().
		JSONScan(&res)
	if err != nil {
		return domain.WechatInfo{}, err
	}
	if res.ErrCode != 0 {
		return domain.WechatInfo{},
			fmt.Errorf("换取 access_token 失败 %d, %s", res.ErrCode, res.ErrMsg)
	}
	return domain.WechatInfo{
		OpenId:      res.OpenId,
		UnionId:     res.UnionId,
		InviterCode: s.getInviterCode(ctx, c.State),
	}, nil
}

func (s *WechatOAuth2Service) getInviterCode(ctx context.Context, state string) string {
	val := s.cache.Get(ctx, state)
	if val.KeyNotFound() {
		return ""
	}
	data, err := val.AsBytes()
	if err != nil {
		return ""
	}
	var a AuthQueryParams
	err = json.Unmarshal(data, &a)
	if err != nil {
		return ""
	}
	return a.InviterCode
}

type Result struct {
	ErrCode int64  `json:"errcode"`
	ErrMsg  string `json:"errMsg"`

	Scope string `json:"scope"`

	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`

	OpenId  string `json:"openid"`
	UnionId string `json:"unionid"`
}
