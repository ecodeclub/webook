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

// RedeemRedemptionCodeReq 使用兑换码
type RedeemRedemptionCodeReq struct {
	Code string `json:"code"`
}

// ListRedemptionCodesReq 分页查询用户所有兑换码
type ListRedemptionCodesReq struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

type ListRedemptionCodesResp struct {
	Total int64            `json:"total"`
	Codes []RedemptionCode `json:"codes"`
}

type RedemptionCode struct {
	Code   string `json:"code"`
	Status uint8  `json:"status"`
	Utime  int64  `json:"utime"`
}
