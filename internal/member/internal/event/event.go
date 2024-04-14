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

const userRegistrationEvents = "user_registration_events"

type RegistrationEvent struct {
	Uid int64 `json:"uid"`
}

// todo: member_events?
const memberUpdateEvents = "member_update_events"

type MemberEvent struct {
	Key    string `json:"key"`
	Uid    int64  `json:"uid"`    // 用户A      用户C
	Days   uint64 `json:"days"`   // 31天会员   366天会员
	Biz    string `json:"biz"`    // user      order  对应的包名
	BizId  int64  `json:"biz_id"` // user_id=A order_id
	Action string `json:"action"` // 首次注册   购买会员
}
