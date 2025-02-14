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

package payment

import (
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/job"
	"github.com/ecodeclub/webook/internal/payment/internal/service"
	"github.com/ecodeclub/webook/internal/payment/internal/web"
)

type (
	Handler                   = web.Handler
	Payment                   = domain.Payment
	WechatJsAPIPrepayResponse = domain.WechatJsAPIPrepayResponse
	Record                    = domain.PaymentRecord
	Channel                   = domain.PaymentChannel
	ChannelType               = domain.ChannelType
	Service                   = service.Service
	SyncWechatOrderJob        = job.SyncWechatOrderJob
)

const (
	ChannelTypeCredit   = domain.ChannelTypeCredit
	ChannelTypeWechat   = domain.ChannelTypeWechat
	ChannelTypeWechatJS = domain.ChannelTypeWechatJS

	StatusUnpaid      = domain.PaymentStatusUnpaid
	StatusProcessing  = domain.PaymentStatusProcessing
	StatusPaidSuccess = domain.PaymentStatusPaidSuccess
	StatusPaidFailed  = domain.PaymentStatusPaidFailed
)

type Module struct {
	Hdl                *Handler
	Svc                Service
	SyncWechatOrderJob *SyncWechatOrderJob
}
