// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ecodeclub/ekit/net/httpx"
	"github.com/ecodeclub/webook/internal/user/internal/domain"
	"github.com/gotomicro/ego/core/elog"
)

type WechatMiniService struct {
	appId     string
	appSecret string
	logger    *elog.Component
	client    *http.Client
}

func NewWechatMiniService(appId string, appSecret string) *WechatMiniService {
	return &WechatMiniService{
		appId:     appId,
		appSecret: appSecret,
		logger:    elog.DefaultLogger,
		client:    http.DefaultClient,
	}
}

func (s *WechatMiniService) AuthURL(ctx context.Context, a AuthParams) (string, error) {
	panic("小程序登录用不上这个")
}

func (s *WechatMiniService) Verify(ctx context.Context, c CallbackParams) (domain.WechatInfo, error) {
	const baseURL = "https://api.weixin.qq.com/sns/jscode2session"
	var res Result
	err := httpx.NewRequest(ctx, http.MethodGet, baseURL).
		Client(s.client).
		AddParam("appid", s.appId).
		AddParam("secret", s.appSecret).AddParam("js_code", c.Code).
		AddParam("grant_type", "authorization_code").Do().
		JSONScan(&res)
	if err != nil {
		return domain.WechatInfo{}, err
	}
	if res.ErrCode != 0 {
		return domain.WechatInfo{},
			fmt.Errorf("小程序登录失败 失败 %d, %s", res.ErrCode, res.ErrMsg)
	}
	return domain.WechatInfo{
		MiniOpenId: res.OpenId,
		UnionId:    res.UnionId,
	}, nil
}
