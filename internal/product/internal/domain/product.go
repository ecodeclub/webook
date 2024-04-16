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

type Status uint8

func (s Status) ToUint8() uint8 {
	return uint8(s)
}

const (
	StatusOffShelf Status = 1 // 下架
	StatusOnShelf  Status = 2 // 上架
)

type SaleType uint8

func (s SaleType) ToUint8() uint8 {
	return uint8(s)
}

const (
	SaleTypeUnlimited SaleType = 1 // 无限期
	SaleTypePromotion SaleType = 2 // 限时促销
	SaleTypePresale   SaleType = 3 // 预售
)

type SPU struct {
	ID     int64
	SN     string
	Name   string
	Desc   string
	SKUs   []SKU
	Status Status
}

type SKU struct {
	ID   int64
	SN   string
	Name string
	Desc string

	Price      int64
	Stock      int64
	StockLimit int64

	SaleType SaleType
	// SaleStart int64
	// SaleEnd   int64
	Attrs  string
	Image  string
	Status Status
}
