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

import (
	"github.com/ecodeclub/webook/internal/permission/internal/domain"
)

const (
	PermissionEventName = "permission_events"
)

type PermissionEvent struct {
	UID    int64   `json:"uid"`
	Biz    string  `json:"biz"` // project,interview
	BizIDs []int64 `json:"biz_ids"`
	Action string  `json:"action"` // 购买项目商品, 兑换项目商品
}

func (p PermissionEvent) toDomain() []domain.PersonalPermission {
	r := make([]domain.PersonalPermission, 0, len(p.BizIDs))
	for _, id := range p.BizIDs {
		r = append(r, domain.PersonalPermission{
			UID:   p.UID,
			Biz:   p.Biz,
			BizID: id,
			Desc:  p.Action,
		})
	}
	return r
}
