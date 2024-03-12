package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ecodeclub/ekit/net/httpx"
	"github.com/ecodeclub/webook/internal/user/internal/domain"
	"github.com/gotomicro/ego/core/elog"
	uuid "github.com/lithammer/shortuuid/v4"
)

const authURLPattern = "https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s#wechat_redire"

type OAuth2Service interface {
	AuthURL() (string, error)
	VerifyCode(ctx context.Context, code string) (domain.WechatInfo, error)
}

type WechatOAuth2Service struct {
	appId       string
	appSecret   string
	redirectURL string
	logger      *elog.Component
	client      *http.Client
}

func NewWechatService(appId, appSecret string) OAuth2Service {
	return &WechatOAuth2Service{
		redirectURL: url.PathEscape("https://i.meoying.com/oauth2/wechat/callback"),
		logger:      elog.DefaultLogger,
		client:      http.DefaultClient,
		appId:       appId,
		appSecret:   appSecret,
	}
}

func (s *WechatOAuth2Service) AuthURL() (string, error) {
	state := uuid.New()
	return fmt.Sprintf(authURLPattern, s.appId, s.redirectURL, state), nil
}

func (s *WechatOAuth2Service) VerifyCode(ctx context.Context, code string) (domain.WechatInfo, error) {
	const baseURL = "https://api.weixin.qq.com/sns/oauth2/access_token"
	var res Result
	err := httpx.NewRequest(ctx, http.MethodGet, baseURL).
		Client(s.client).
		AddParam("appid", s.appId).
		AddParam("secret", s.appSecret).AddParam("code", code).
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
		OpenId:  res.OpenId,
		UnionId: res.UnionId,
	}, nil
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
