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

type Payment struct {
	SN          string
	OrderID     int64
	OrderSN     string
	TotalAmount int64
	Deadline    int64
	PaidAt      int64
	Status      int64
}

type PayReq struct {
	OrderSN  int64     `json:"order_sn"`
	Channels []Channel `json:"channels"`
}

type Channel struct {
	Type          int64  `json:"type,omitempty"`
	Desc          string `json:"desc,omitempty"`
	Amount        int64  `json:"amount"`
	WechatCodeURL string `json:"wechatCodeURL,omitempty"` // 微信二维码
}
