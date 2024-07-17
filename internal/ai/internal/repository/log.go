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

	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
)

type LLMLogRepo interface {
	SaveLog(ctx context.Context, l domain.LLMRecord) (int64, error)
}

// 调用日志
type llmLogDAO struct {
	logDao dao.LLMRecordDAO
}

func NewLLMLogRepo(logDao dao.LLMRecordDAO) LLMLogRepo {
	return &llmLogDAO{
		logDao: logDao,
	}
}

func (g *llmLogDAO) SaveLog(ctx context.Context, l domain.LLMRecord) (int64, error) {
	logEntity := g.toEntity(l)
	return g.logDao.Save(ctx, logEntity)
}

func (g *llmLogDAO) toEntity(r domain.LLMRecord) dao.LLMRecord {
	return dao.LLMRecord{
		Id:          r.Id,
		Tid:         r.Tid,
		Uid:         r.Uid,
		Biz:         r.Biz,
		Tokens:      r.Tokens,
		Amount:      r.Amount,
		KnowledgeId: r.KnowledgeId,
		Input: sqlx.JsonColumn[[]string]{
			Valid: true,
			Val:   r.Input,
		},
		Status:         r.Status.ToUint8(),
		PromptTemplate: sqlx.NewNullString(r.PromptTemplate),
		Answer:         sqlx.NewNullString(r.Answer),
	}
}
