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

type RedemptionCodeStatus uint8

const (
	RedemptionCodeStatusUnused RedemptionCodeStatus = 1
	RedemptionCodeStatusUsed   RedemptionCodeStatus = 2
)

func (r RedemptionCodeStatus) ToUint8() uint8 {
	return uint8(r)
}

type RedemptionCode struct {
	ID      int64
	OwnerID int64
	Biz     string
	BizId   int64
	Type    string
	Attrs   CodeAttrs
	Code    string
	Status  RedemptionCodeStatus
	Ctime   int64
	Utime   int64
}

type CodeAttrs struct {
	SKU SKU `json:"sku,omitempty"`
}

type SKU struct {
	ID    int64  `json:"id,omitempty"`
	SN    string `json:"sn,omitempty"`
	Name  string `json:"name,omitempty"`
	Attrs string `json:"attrs,omitempty"`
}
