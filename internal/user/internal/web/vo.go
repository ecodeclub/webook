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

import "github.com/ecodeclub/webook/internal/user/internal/domain"

type Profile struct {
	Id        int64  `json:"id,omitempty"`
	Nickname  string `json:"nickname,omitempty"`
	Avatar    string `json:"avatar,omitempty"`
	SN        string `json:"sn,omitempty"`
	IsCreator bool   `json:"isCreator,omitempty"`
	// 毫秒数
	MemberDDL int64 `json:"memberDDL,omitempty"`
}

func newProfile(u domain.User) Profile {
	return Profile{
		Nickname: u.Nickname,
		Avatar:   u.Avatar,
		SN:       u.SN,
	}
}

type WechatCallback struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type EditReq struct {
	Avatar   string `json:"avatar"`
	Nickname string `json:"nickname"`
}

type SendCodeReq struct {
	Phone string `json:"phone"`
}

type PhoneReq struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}
