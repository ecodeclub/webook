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

package event

const creditIncreaseEvents = "credit_increase_events"

type CreditIncreaseEvent struct {
	Key    string `json:"key"`
	Uid    int64  `json:"uid"`    // 用户A                用户C
	Amount uint64 `json:"amount"` // 增加100              增加1000
	Biz    int64  `json:"biz"`    // 用户模块            下单
	BizId  int64  `json:"biz_id"` // 通过用户B userB_id    oder_id
	Action string `json:"action"` // 邀请注册             购买商品
}
