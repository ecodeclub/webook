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

package syncer

import (
	"context"
	"fmt"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/kbase/internal/domain"
	"github.com/ecodeclub/webook/internal/kbase/internal/service"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/gotomicro/ego/core/elog"
)

type QuestionSyncer struct {
	indexName  string
	batchSize  int
	baguwenSvc baguwen.Service
	svc        service.Service
	logger     *elog.Component
}

func NewQuestionSyncer(indexName string, batchSize int, baguwenSvc baguwen.Service, svc service.Service) *QuestionSyncer {
	return &QuestionSyncer{
		indexName:  indexName,
		batchSize:  batchSize,
		baguwenSvc: baguwenSvc,
		svc:        svc,
		logger:     elog.DefaultLogger.With(elog.FieldComponent("service.QuestionSyncer")),
	}
}

func (q *QuestionSyncer) Upsert(ctx context.Context, id int64) error {
	que, err := q.baguwenSvc.PubDetailWithoutCntView(ctx, id)
	if err != nil {
		return err
	}
	return q.svc.BulkUpsert(ctx, q.indexName, []domain.Document{
		q.toKbaseDocument(que),
	})
}

func (q *QuestionSyncer) toKbaseDocument(que baguwen.Question) domain.Document {
	return domain.Document{
		ID: q.esID(que.Id),
		Body: map[string]any{
			"id":      que.Id,
			"title":   que.Title,
			"biz":     que.Biz,
			"biz_id":  que.BizId,
			"labels":  que.Labels,
			"content": que.Content,
			"status":  que.Status,
			"answer": map[string]any{
				"analysis":     q.convertAnswerElement2Map(que.Answer.Analysis),
				"basic":        q.convertAnswerElement2Map(que.Answer.Basic),
				"intermediate": q.convertAnswerElement2Map(que.Answer.Intermediate),
				"advanced":     q.convertAnswerElement2Map(que.Answer.Advanced),
				"utime":        que.Answer.Utime,
			},
			"utime": que.Utime,
		},
	}
}

func (q *QuestionSyncer) esID(id int64) string {
	return fmt.Sprintf("%d", id)
}

func (q *QuestionSyncer) convertAnswerElement2Map(a baguwen.AnswerElement) map[string]any {
	return map[string]any{
		"id":        q.esID(a.Id),
		"content":   a.Content,
		"keywords":  a.Keywords,
		"shorthand": a.Shorthand,
		"highlight": a.Highlight,
		"guidance":  a.Guidance,
	}
}

func (q *QuestionSyncer) UpsertSince(ctx context.Context, startTime int64) error {
	offset := 0
	for {
		questions, err := q.baguwenSvc.ListPubSince(ctx, startTime, offset, q.batchSize)
		if err != nil {
			return err
		}
		if len(questions) == 0 {
			break
		}

		err = q.svc.BulkUpsert(ctx, q.indexName, slice.Map(questions, func(_ int, src baguwen.Question) domain.Document {
			return q.toKbaseDocument(src)
		}))

		if err != nil {
			q.logger.Error("同步到Kbase失败", elog.FieldErr(err))
			continue
		}

		offset += len(questions)
	}
	return nil
}

func (q *QuestionSyncer) Delete(ctx context.Context, id int64) error {
	return q.svc.BulkDelete(ctx, q.indexName, []string{q.esID(id)})
}
