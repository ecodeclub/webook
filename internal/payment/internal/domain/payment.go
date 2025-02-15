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

package domain

type ChannelType uint8

func (c ChannelType) ToUnit8() uint8 {
	return uint8(c)
}

const (
	ChannelTypeCredit   ChannelType = 1
	ChannelTypeWechat   ChannelType = 2
	ChannelTypeWechatJS ChannelType = 3
)

type PaymentStatus uint8

func (s PaymentStatus) ToUint8() uint8 {
	return uint8(s)
}

const (
	PaymentStatusUnpaid        PaymentStatus = 1
	PaymentStatusProcessing    PaymentStatus = 2
	PaymentStatusPaidSuccess   PaymentStatus = 3
	PaymentStatusPaidFailed    PaymentStatus = 4
	PaymentStatusRefund        PaymentStatus = 5
	PaymentStatusTimeoutClosed PaymentStatus = 6
)

type Amount struct {
	// 如果要支持国际化，那么这个是不能少的
	Currency string
	// 这里我们遵循微信的做法，就用 int64 来记录分数。
	// 那么对于不同的货币来说，这个字段的含义就不同。
	// 比如说一些货币没有分，只有整数。
	Total int64
}

type Payment struct {
	ID      int64
	SN      string
	PayerID int64
	// BizTradeNO, 就是OrderSN
	OrderID int64
	OrderSN string
	// 订单的描述,冗余
	OrderDescription string
	TotalAmount      int64
	PaidAt           int64
	Status           PaymentStatus
	Records          []PaymentRecord
	Ctime            int64
}

type PaymentChannel struct {
	Type ChannelType
	Desc string
}

type PaymentRecord struct {
	PaymentID int64
	// 第三方那边返回的 ID TxnID string
	PaymentNO3rd string
	Description  string
	Channel      ChannelType
	Amount       int64
	PaidAt       int64
	Status       PaymentStatus
	// Native 支付方式使用
	WechatCodeURL string
	// JSAPI 支付方式使用
	WechatJsAPIResp WechatJsAPIPrepayResponse
}

type WechatJsAPIPrepayResponse struct {
	// 预支付交易会话标识
	PrepayId string
	// 应用ID
	Appid string
	// 时间戳
	TimeStamp string
	// 随机字符串
	NonceStr string
	// 订单详情扩展字符串
	Package string
	// 签名方式
	SignType string
	// 签名
	PaySign string
}
