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

type OrderStatus uint8

func (s OrderStatus) ToUint8() uint8 {
	return uint8(s)
}

const (
	StatusUnpaid    OrderStatus = 1
	StatusCompleted OrderStatus = 2
	StatusCanceled  OrderStatus = 3
	StatusExpired   OrderStatus = 4
)

type Order struct {
	ID                 int64
	SN                 string
	BuyerID            int64
	PaymentID          int64
	PaymentSN          string
	OriginalTotalPrice int64
	RealTotalPrice     int64
	Status             OrderStatus
	Items              []OrderItem
	Ctime              int64
	Utime              int64
}

type OrderItem struct {
	OrderID          int64
	SPUID            int64
	SKUID            int64
	SPUSN            string
	SKUSN            string
	SKUImage         string
	SKUName          string
	SKUDescription   string
	SKUOriginalPrice int64
	SKURealPrice     int64
	Quantity         int64
}
