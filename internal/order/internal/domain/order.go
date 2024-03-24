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
	OrderStatusUnpaid    = iota + 1 // 未支付
	OrderStatusCompleted            // 已完成(已支付)
	OrderStatusCanceled             // 已取消
	OrderStatusExpired              // 已超时
)

type Order struct {
	ID                 int64
	SN                 string
	BuyerID            int64
	PaymentID          int64
	PaymentSN          string
	OriginalTotalPrice int64
	RealTotalPrice     int64
	ClosedAt           int64
	Status             int64
	Items              []OrderItem
	Ctime              int64
	Utime              int64
}

type OrderItem struct {
	OrderID          int64
	SPUID            int64
	SKUID            int64
	SKUName          string
	SKUDescription   string
	SKUOriginalPrice int64
	SKURealPrice     int64
	Quantity         int64
}
