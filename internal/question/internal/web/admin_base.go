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
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
)

type AdminBaseHandler struct {
}

func (h AdminBaseHandler) toQuestionList(data []domain.Question, cnt int64) QuestionList {
	return QuestionList{
		Total: cnt,
		Questions: slice.Map(data, func(idx int, src domain.Question) Question {
			return Question{
				Id:      src.Id,
				Title:   src.Title,
				Content: src.Content,
				Labels:  src.Labels,
				Biz:     src.Biz,
				BizId:   src.BizId,
				Status:  src.Status.ToUint8(),
				Utime:   src.Utime.UnixMilli(),
			}
		}),
	}
}
