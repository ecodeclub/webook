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

type Payment struct {
	ID          int64
	SN          string
	OrderID     int64
	OrderSN     string
	TotalAmount int64
	Deadline    int64
	PaidAt      int64
	Status      int64
	Records     []PaymentRecord
	Ctime       int64
	Utime       int64
}

type PaymentChannel struct {
	Type int64
	Desc string
}

type PaymentRecord struct {
	PaymentNO3rd  string
	Channel       int64
	Amount        int64
	PaidAt        int64
	Status        int64
	WechatCodeURL string
}
