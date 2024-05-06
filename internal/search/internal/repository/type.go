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

package repository

import (
	"context"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
)

type CaseRepo interface {
	SearchCase(ctx context.Context, keywords string) ([]domain.Case, error)
}

type QuestionRepo interface {
	SearchQuestion(ctx context.Context, keywords string) ([]domain.Question, error)
}
type QuestionSetRepo interface {
	SearchQuestionSet(ctx context.Context, keywords string) ([]domain.QuestionSet, error)
}

type SkillRepo interface {
	SearchSkill(ctx context.Context, keywords string) ([]domain.Skill, error)
}

type AnyRepo interface {
	Input(ctx context.Context, index string, docID string, data string) error
}
