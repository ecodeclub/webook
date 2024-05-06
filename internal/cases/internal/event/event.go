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
	"encoding/json"

	"github.com/ecodeclub/webook/internal/cases/internal/domain"
)

type CaseEvent struct {
	Biz   string `json:"biz"`
	BizID int    `json:"bizID"`
	Data  string `json:"data"`
}
type Case struct {
	Id        int64    `json:"id"`
	Uid       int64    `json:"uid"`
	Labels    []string `json:"labels"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	CodeRepo  string   `json:"code_repo"`
	Keywords  string   `json:"keywords"`
	Shorthand string   `json:"shorthand"`
	Highlight string   `json:"highlight"`
	Guidance  string   `json:"guidance"`
	Status    uint8    `json:"status"`
	Ctime     int64    `json:"ctime"`
	Utime     int64    `json:"utime"`
}

func NewCaseEvent(ca *domain.Case) CaseEvent {
	qByte, _ := json.Marshal(newCase(ca))
	return CaseEvent{
		Biz:   "case",
		BizID: int(ca.Id),
		Data:  string(qByte),
	}
}

func newCase(ca *domain.Case) Case {
	return Case{
		Id:        ca.Id,
		Uid:       ca.Uid,
		Labels:    ca.Labels,
		Title:     ca.Title,
		Content:   ca.Content,
		CodeRepo:  ca.CodeRepo,
		Keywords:  ca.Keywords,
		Shorthand: ca.Shorthand,
		Highlight: ca.Highlight,
		Guidance:  ca.Guidance,
		Status:    ca.Status.ToUint8(),
		Ctime:     ca.Ctime.UnixMilli(),
		Utime:     ca.Utime.UnixMilli(),
	}
}
