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

package event

const PaymentEventName = "payment_events"

// PaymentEvent 也是最简设计
// 有一些人会习惯把支付详情也放进来，但是目前来看是没有必要的
// 后续如果要接入大数据之类的，那么就可以考虑提供 payment 详情
type PaymentEvent struct {
	OrderSN string `json:"orderSN"`
	PayerID int64  `json:"payerID"`
	Status  uint8  `json:"status"`
}
