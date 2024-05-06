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
	// StatusInit 初始化状态，对用户是不可见的
	StatusInit          OrderStatus = 1
	StatusProcessing    OrderStatus = 2
	StatusSuccess       OrderStatus = 3
	StatusFailed        OrderStatus = 4
	StatusCanceled      OrderStatus = 5
	StatusTimeoutClosed OrderStatus = 6
)

type Order struct {
	ID               int64
	SN               string
	BuyerID          int64
	Payment          Payment
	OriginalTotalAmt int64
	RealTotalAmt     int64
	Status           OrderStatus
	Items            []OrderItem
	Ctime            int64
	Utime            int64
}

type Payment struct {
	ID int64
	SN string
}

type OrderItem struct {
	SPU SPU
	SKU SKU
}

type Category struct {
	Name string
	Desc string
}

type SPU struct {
	ID       int64
	Category Category
}

type SKU struct {
	ID            int64
	SN            string
	Image         string
	Attrs         string
	Name          string
	Description   string
	OriginalPrice int64
	RealPrice     int64
	Quantity      int64
}
