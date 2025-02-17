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

package cache

import (
	"context"

	"github.com/ecodeclub/webook/internal/question/internal/domain"
)

type QuestionCache interface {
	GetTotal(ctx context.Context, biz string) (int64, error)
	SetTotal(ctx context.Context, biz string, total int64) error
	SetQuestion(ctx context.Context, question domain.Question) error
	GetQuestion(ctx context.Context, id int64) (domain.Question, error)
	SetQuestions(ctx context.Context, biz string, questions []domain.Question) error
	GetQuestions(ctx context.Context, biz string) ([]domain.Question, error)
	DelQuestion(ctx context.Context, id int64) error
}
