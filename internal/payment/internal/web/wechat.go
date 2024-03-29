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

package web

import (
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/payment/internal/service/wechat"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
)

var _ ginx.Handler = &WechatHandler{}

type WechatHandler struct {
	handler   *notify.Handler
	l         *elog.Component
	nativeSvc *wechat.NativePaymentService
}

func NewWechatHandler(handler *notify.Handler, nativeSvc *wechat.NativePaymentService) *WechatHandler {
	return &WechatHandler{
		handler:   handler,
		nativeSvc: nativeSvc,
		l:         elog.DefaultLogger}
}

func (h *WechatHandler) PrivateRoutes(_ *gin.Engine) {}

func (h *WechatHandler) PublicRoutes(server *gin.Engine) {
	server.Any("/pay/callback", ginx.W(h.HandleNativeCallBack))
}

func (h *WechatHandler) HandleNativeCallBack(ctx *ginx.Context) (ginx.Result, error) {
	transaction := &payments.Transaction{}
	_, err := h.handler.ParseNotifyRequest(ctx, ctx.Request, transaction)
	if err != nil {
		return ginx.Result{}, err
	}
	err = h.nativeSvc.HandleCallback(ctx, transaction)
	return ginx.Result{}, err
}
