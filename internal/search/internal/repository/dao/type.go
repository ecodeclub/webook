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

package dao

import (
	"context"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
)

type CaseDAO interface {
	SearchCase(ctx context.Context, offset, limit int, queryMetas []domain.QueryMeta) ([]Case, error)
}

type QuestionDAO interface {
	SearchQuestion(ctx context.Context, offset, limit int, queryMetas []domain.QueryMeta) ([]Question, error)
}

type SkillDAO interface {
	// ids 为case的id 和question的id
	SearchSkill(ctx context.Context, offset, limit int, queryMetas []domain.QueryMeta) ([]Skill, error)
}

type QuestionSetDAO interface {
	// ids 为case的id 和question的id
	SearchQuestionSet(ctx context.Context, offset, limit int, queryMetas []domain.QueryMeta) ([]QuestionSet, error)
}

type AnyDAO interface {
	Input(ctx context.Context, index string, docID string, data string) error
}
