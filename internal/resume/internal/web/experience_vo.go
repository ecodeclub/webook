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

import (
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/resume/internal/domain"
)

type Experience struct {
	Id               int64            `json:"id"`
	Start            time.Time        `json:"start"`
	End              time.Time        `json:"end"`
	Title            string           `json:"title"`
	CompanyName      string           `json:"company_name"`
	Location         string           `json:"location"`
	Responsibilities []Responsibility `json:"responsibilities"`
	Accomplishments  []Accomplishment `json:"accomplishments"`
	Skills           []string         `json:"skills"`
}

type Responsibility struct {
	// Type 是类型，比如说核心研发、团队管理
	// 用 string 来作为枚举
	Type    string `json:"type"`
	Content string `json:"content"`
}

type Accomplishment struct {
	// Type 是类型，比如说性能优化，获奖啥的
	Type    string `json:"type"`
	Content string `json:"content"`
}

type ListResp struct {
	Experiences []Experience `json:"experiences"`
}

func newExperience(experience domain.Experience) Experience {
	return Experience{
		Id:          experience.Id,
		Start:       experience.Start,
		End:         experience.End,
		Title:       experience.Title,
		CompanyName: experience.CompanyName,
		Location:    experience.Location,
		Responsibilities: slice.Map(experience.Responsibilities, func(idx int, src domain.Responsibility) Responsibility {
			return Responsibility{
				Type:    src.Type,
				Content: src.Content,
			}
		}),
		Accomplishments: slice.Map(experience.Accomplishments, func(idx int, src domain.Accomplishment) Accomplishment {
			return Accomplishment{
				Type:    src.Type,
				Content: src.Content,
			}
		}),
		Skills: experience.Skills,
	}

}
