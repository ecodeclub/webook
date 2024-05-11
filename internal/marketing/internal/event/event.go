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

const (
	MemberUpdateEventName = "member_update_events"
	OrderEventName        = "order_events"
	CreditEventName       = "credit_increase_events"
	PermissionEventName   = "permission_events"
)

type MemberEvent struct {
	Key    string `json:"key"`
	Uid    int64  `json:"uid"`    // 用户A      用户C
	Days   uint64 `json:"days"`   // 31天会员   366天会员
	Biz    string `json:"biz"`    // user      order  对应的包名
	BizId  int64  `json:"biz_id"` // user_id=A order_id
	Action string `json:"action"` // 首次注册   购买会员
}

type OrderEvent struct {
	OrderSN string `json:"orderSN"`
	BuyerID int64  `json:"buyerID"`
	SPUs    []SPU  `json:"spus"`
}

type SPU struct {
	ID       int64  `json:"id"`
	Category string `json:"category"`
	Type     string `json:"type"`
}

func (s SPU) IsProductCategory() bool {
	return s.Category == "product"
}

func (s SPU) IsCodeCategory() bool {
	return s.Category == "code"
}

func (s SPU) IsMemberProduct() bool {
	return s.IsProductCategory() && s.Type == "member"
}

type CreditIncreaseEvent struct {
	Key    string `json:"key"`
	Uid    int64  `json:"uid"`    // 用户A       用户C
	Amount uint64 `json:"amount"` // 增加100     增加1000
	Biz    string `json:"biz"`    // user        order
	BizId  int64  `json:"biz_id"` // user_id=B   order_id
	Action string `json:"action"` // 邀请注册     购买商品
}

type PermissionEvent struct {
	// todo: 权限模块
	Key string `json:"key"`
}
