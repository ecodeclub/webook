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

const (
	ChannelTypeCredit = iota + 1
	ChannelTypeWechat
)

const (
	PaymentStatusUnpaid = iota + 1
	PaymentStatusPaid
	PaymentStatusFailed
	PaymentStatusRefund
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
	PayDDL           int64
	PaidAt           int64
	Status           int64
	Records          []PaymentRecord
	Ctime            int64
	Utime            int64
}

type PaymentChannel struct {
	Type int64
	Desc string
}

type PaymentRecord struct {
	PaymentID int64
	// 第三方那边返回的 ID TxnID string
	PaymentNO3rd  string
	Description   string
	Channel       int64
	Amount        int64
	PaidAt        int64
	Status        int64
	WechatCodeURL string
}
