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
	StatusOffShelf = iota // 下架
	StatusOnShelf         // 上架
)

type Product struct {
	SPU SPU
	SKU SKU
}

type SPU struct {
	SN     string
	Name   string
	Desc   string
	Status int64
}

type SKU struct {
	SN   string
	Name string
	Desc string

	Price      int64
	Stock      int64
	StockLimit int64

	SaleType int64
	// SaleStart int64
	// SaleEnd   int64
	Status int64
}
