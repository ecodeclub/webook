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

type AuthParams struct {
	InvitationCode string `json:"invitation,omitempty"`
}

type CallbackParams struct {
	Code  string
	State string
}

//go:generate mockgen -source=./wechat_service.go -package=svcmocks -typed=true -destination=mocks/wechat_service.mock.go OAuth2Service
type OAuth2Service interface {
	AuthURL(ctx context.Context, a AuthParams) (string, error)
	Verify(ctx context.Context, c CallbackParams) (domain.WechatInfo, error)
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

func (s *WechatOAuth2Service) AuthURL(ctx context.Context, a AuthParams) (string, error) {
	return fmt.Sprintf(authURLPattern, s.appId, s.redirectURL, s.getState(ctx, a)), nil
}

func (s *WechatOAuth2Service) getState(ctx context.Context, a AuthParams) string {
	state := uuid.New()
	if a.InvitationCode != "" {
		// 尽最大努力建立映射但不阻碍主流程
		data, err := json.Marshal(a)
		if err != nil {
			s.logger.Warn("序列化认证参数失败", elog.FieldErr(err))
			return ""
		}
		err = s.cache.Set(ctx, state, string(data), time.Minute*30)
		if err != nil {
			s.logger.Warn("缓存认证参数失败", elog.FieldErr(err))
			return ""
		}
	}
	return state
}

func (s *WechatOAuth2Service) Verify(ctx context.Context, c CallbackParams) (domain.WechatInfo, error) {
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
		OpenId:         res.OpenId,
		UnionId:        res.UnionId,
		InvitationCode: s.getInviterCode(ctx, c.State),
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
	var a AuthParams
	err = json.Unmarshal(data, &a)
	if err != nil {
		s.logger.Warn("反序列化认证参数失败", elog.FieldErr(err))
		return ""
	}
	return a.InvitationCode
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
