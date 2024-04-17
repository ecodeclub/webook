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

type SKUSNReq struct {
	SN string `json:"sn"`
}

type SNReq struct {
	SN string `json:"sn"`
}

type SPU struct {
	SN   string `json:"sn"`
	Name string `json:"name"`
	Desc string `json:"desc"`
	SKUs []SKU  `json:"skus"`
}

type SKU struct {
	SN         string `json:"sn"`
	Name       string `json:"name"`
	Desc       string `json:"desc"`
	Price      int64  `json:"price"`
	Stock      int64  `json:"stock"`
	StockLimit int64  `json:"stockLimit"`
	SaleType   uint8  `json:"saleType"`
	Attrs      string `json:"attrs,omitempty"`
	Image      string `json:"image"`
	// SaleStart  int64  `json:"saleStart"`
	// SaleEnd    int64  `json:"saleEnd"`
}

type SPUSNReq struct {
	SN string `json:"sn"`
}
