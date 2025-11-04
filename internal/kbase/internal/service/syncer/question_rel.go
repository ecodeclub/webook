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
	"github.com/ecodeclub/webook/internal/roadmap"
	"github.com/gotomicro/ego/core/elog"
)

type QuestionRelSyncer struct {
	indexName  string
	batchSize  int
	roadmapSvc roadmap.AdminService
	svc        service.Service
	logger     *elog.Component
}

func NewQuestionRelSyncer(indexName string, batchSize int, roadmapSvc roadmap.AdminService, svc service.Service) *QuestionRelSyncer {
	return &QuestionRelSyncer{
		indexName:  indexName,
		batchSize:  batchSize,
		roadmapSvc: roadmapSvc,
		svc:        svc,
		logger:     elog.DefaultLogger.With(elog.FieldComponent("syncer.QuestionRelSyncer")),
	}
}

func (q *QuestionRelSyncer) Upsert(ctx context.Context, id int64) error {
	rd, err := q.roadmapSvc.Detail(ctx, id)
	if err != nil {
		return err
	}
	return q.svc.BulkUpsert(ctx, q.indexName, q.toKbaseDocuments(rd.Biz, rd.BizId, rd.Edges))
}

func (q *QuestionRelSyncer) toKbaseDocuments(biz string, bizID int64, edges []roadmap.Edge) []domain.Document {
	return slice.Map(edges, func(_ int, e roadmap.Edge) domain.Document {
		return domain.Document{
			ID: fmt.Sprintf("%d", e.Id),
			Body: map[string]any{
				"rid":       e.Src.Rid,
				"biz":       biz,
				"biz_id":    bizID,
				"type":      e.Type,
				"attrs":     e.Attrs,
				"src_id":    e.Src.ID,
				"src_title": e.Src.Title,
				"dst_id":    e.Dst.ID,
				"dst_title": e.Dst.Title,
			},
		}
	})
}

func (q *QuestionRelSyncer) UpsertSince(ctx context.Context, startTime int64) error {
	offset := 0
	batchDocs := make([]domain.Document, 0, q.batchSize)
	for {
		roadmaps, err := q.roadmapSvc.ListSince(ctx, startTime, offset, q.batchSize)
		if err != nil {
			return err
		}
		if len(roadmaps) == 0 {
			break
		}

		for i := range roadmaps {
			docs := q.toKbaseDocuments(roadmaps[i].Biz, roadmaps[i].BizId, roadmaps[i].Edges)
			for _, doc := range docs {
				if len(batchDocs) >= q.batchSize {
					err = q.svc.BulkUpsert(ctx, q.indexName, batchDocs)
					if err != nil {
						q.logger.Error("同步到Kbase失败", elog.FieldErr(err))
						return err
					}
					batchDocs = batchDocs[:0]
				}
				batchDocs = append(batchDocs, doc)
			}
		}

		offset += len(roadmaps)
	}

	if len(batchDocs) > 0 {
		err := q.svc.BulkUpsert(ctx, q.indexName, batchDocs)
		if err != nil {
			q.logger.Error("同步到Kbase失败", elog.FieldErr(err))
			return err
		}
	}

	return nil
}

func (q *QuestionRelSyncer) Delete(ctx context.Context, id int64) error {
	return q.svc.BulkDelete(ctx, q.indexName, []string{fmt.Sprintf("%d", id)})
}
