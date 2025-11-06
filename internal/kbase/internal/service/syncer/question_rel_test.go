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
	"errors"
	"testing"

	"github.com/ecodeclub/webook/internal/kbase/internal/domain"
	kbasemocks "github.com/ecodeclub/webook/internal/kbase/mocks"
	"github.com/ecodeclub/webook/internal/roadmap"
	roadmapmocks "github.com/ecodeclub/webook/internal/roadmap/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestQuestionRelSyncer_Upsert(t *testing.T) {
	testCases := []struct {
		name    string
		id      int64
		setup   func(*roadmapmocks.MockAdminService, *kbasemocks.MockService)
		wantErr error
	}{
		{
			name: "成功",
			id:   123,
			setup: func(roadmapSvc *roadmapmocks.MockAdminService, kbaseSvc *kbasemocks.MockService) {
				roadmapSvc.EXPECT().Detail(gomock.Any(), int64(123)).
					Return(roadmap.Roadmap{
						Id:    123,
						Title: "测试路线图",
						Biz:   roadmap.Biz{Biz: "questionSet", BizId: 456},
						Edges: []roadmap.Edge{
							{
								Id:    1,
								Type:  "prerequisite",
								Attrs: `{"weight": 1}`,
								Src: roadmap.Node{
									ID:    100,
									Title: "源节点",
									Rid:   123,
									Biz: roadmap.Biz{
										Biz:   "question",
										BizId: 100,
										Title: "源题目",
									},
								},
								Dst: roadmap.Node{
									ID:    200,
									Title: "目标节点",
									Rid:   123,
									Biz: roadmap.Biz{
										Biz:   "question",
										BizId: 200,
										Title: "目标题目",
									},
								},
							},
						},
					}, nil).Times(1)
				kbaseSvc.EXPECT().BulkUpsert(gomock.Any(), "question_rel_index", gomock.Any()).
					DoAndReturn(func(ctx context.Context, indexName string, docs []domain.Document) error {
						require.Len(t, docs, 1)
						doc := docs[0]
						assert.Equal(t, "1", doc.ID)
						assert.Equal(t, int64(123), doc.Body["rid"])
						assert.Equal(t, "questionSet", doc.Body["biz"])
						assert.Equal(t, int64(456), doc.Body["biz_id"])
						assert.Equal(t, "prerequisite", doc.Body["type"])
						return nil
					}).Times(1)
			},
		},
		{
			name: "roadmap不存在",
			id:   123,
			setup: func(roadmapSvc *roadmapmocks.MockAdminService, kbaseSvc *kbasemocks.MockService) {
				roadmapSvc.EXPECT().Detail(gomock.Any(), int64(123)).
					Return(roadmap.Roadmap{}, errors.New("roadmap not found")).Times(1)
			},
			wantErr: errors.New("roadmap not found"),
		},
		{
			name: "kbase service错误",
			id:   123,
			setup: func(roadmapSvc *roadmapmocks.MockAdminService, kbaseSvc *kbasemocks.MockService) {
				roadmapSvc.EXPECT().Detail(gomock.Any(), int64(123)).
					Return(roadmap.Roadmap{
						Id:  123,
						Biz: roadmap.Biz{Biz: "questionSet", BizId: 456},
						Edges: []roadmap.Edge{
							{Id: 1},
						},
					}, nil).Times(1)
				kbaseSvc.EXPECT().BulkUpsert(gomock.Any(), "question_rel_index", gomock.Any()).
					Return(errors.New("ES错误")).Times(1)
			},
			wantErr: errors.New("ES错误"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			roadmapSvc := roadmapmocks.NewMockAdminService(ctrl)
			kbaseSvc := kbasemocks.NewMockService(ctrl)

			syncer := NewQuestionRelSyncer("question_rel_index", 100, roadmapSvc, kbaseSvc)

			tc.setup(roadmapSvc, kbaseSvc)

			err := syncer.Upsert(t.Context(), tc.id)
			if tc.wantErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestQuestionRelSyncer_UpsertSince(t *testing.T) {
	testCases := []struct {
		name      string
		startTime int64
		setup     func(*roadmapmocks.MockAdminService, *kbasemocks.MockService)
		wantErr   error
	}{
		{
			name:      "空数据",
			startTime: 1000,
			setup: func(roadmapSvc *roadmapmocks.MockAdminService, kbaseSvc *kbasemocks.MockService) {
				roadmapSvc.EXPECT().ListSince(gomock.Any(), int64(1000), 0, 100).
					Return([]roadmap.Roadmap{}, nil).Times(1)
			},
		},
		{
			name:      "单个roadmap的edges小于batchSize",
			startTime: 1000,
			setup: func(roadmapSvc *roadmapmocks.MockAdminService, kbaseSvc *kbasemocks.MockService) {
				roadmapSvc.EXPECT().ListSince(gomock.Any(), int64(1000), 0, 100).
					Return([]roadmap.Roadmap{
						{
							Id:  1,
							Biz: roadmap.Biz{Biz: "questionSet", BizId: 100},
							Edges: []roadmap.Edge{
								{Id: 1, Src: roadmap.Node{ID: 10, Rid: 1}, Dst: roadmap.Node{ID: 20, Rid: 1}},
								{Id: 2, Src: roadmap.Node{ID: 11, Rid: 1}, Dst: roadmap.Node{ID: 21, Rid: 1}},
							},
						},
					}, nil).Times(1)
				kbaseSvc.EXPECT().BulkUpsert(gomock.Any(), "question_rel_index", gomock.Any()).
					DoAndReturn(func(ctx context.Context, indexName string, docs []domain.Document) error {
						require.Len(t, docs, 2)
						return nil
					}).Times(1)
				roadmapSvc.EXPECT().ListSince(gomock.Any(), int64(1000), 1, 100).
					Return([]roadmap.Roadmap{}, nil).Times(1)
			},
		},
		{
			name:      "单个roadmap的edges超过batchSize需要拆分",
			startTime: 1000,
			setup: func(roadmapSvc *roadmapmocks.MockAdminService, kbaseSvc *kbasemocks.MockService) {
				edges := make([]roadmap.Edge, 0, 150)
				for i := 0; i < 150; i++ {
					edges = append(edges, roadmap.Edge{
						Id:  int64(i + 1),
						Src: roadmap.Node{ID: 10, Rid: 1},
						Dst: roadmap.Node{ID: 20, Rid: 1},
					})
				}
				roadmapSvc.EXPECT().ListSince(gomock.Any(), int64(1000), 0, 100).
					Return([]roadmap.Roadmap{
						{
							Id:    1,
							Biz:   roadmap.Biz{Biz: "questionSet", BizId: 100},
							Edges: edges,
						},
					}, nil).Times(1)
				// 第一次批次：100个
				kbaseSvc.EXPECT().BulkUpsert(gomock.Any(), "question_rel_index", gomock.Any()).
					DoAndReturn(func(ctx context.Context, indexName string, docs []domain.Document) error {
						require.Len(t, docs, 100)
						return nil
					}).Times(1)
				// 第二次批次：50个
				kbaseSvc.EXPECT().BulkUpsert(gomock.Any(), "question_rel_index", gomock.Any()).
					DoAndReturn(func(ctx context.Context, indexName string, docs []domain.Document) error {
						require.Len(t, docs, 50)
						return nil
					}).Times(1)
				roadmapSvc.EXPECT().ListSince(gomock.Any(), int64(1000), 1, 100).
					Return([]roadmap.Roadmap{}, nil).Times(1)
			},
		},
		{
			name:      "多个roadmap的edges合并批次",
			startTime: 1000,
			setup: func(roadmapSvc *roadmapmocks.MockAdminService, kbaseSvc *kbasemocks.MockService) {
				roadmapSvc.EXPECT().ListSince(gomock.Any(), int64(1000), 0, 100).
					Return([]roadmap.Roadmap{
						{
							Id:  1,
							Biz: roadmap.Biz{Biz: "questionSet", BizId: 100},
							Edges: []roadmap.Edge{
								{Id: 1, Src: roadmap.Node{ID: 10, Rid: 1}, Dst: roadmap.Node{ID: 20, Rid: 1}},
								{Id: 2, Src: roadmap.Node{ID: 11, Rid: 1}, Dst: roadmap.Node{ID: 21, Rid: 1}},
							},
						},
						{
							Id:  2,
							Biz: roadmap.Biz{Biz: "questionSet", BizId: 200},
							Edges: []roadmap.Edge{
								{Id: 3, Src: roadmap.Node{ID: 12, Rid: 2}, Dst: roadmap.Node{ID: 22, Rid: 2}},
							},
						},
					}, nil).Times(1)
				kbaseSvc.EXPECT().BulkUpsert(gomock.Any(), "question_rel_index", gomock.Any()).
					DoAndReturn(func(ctx context.Context, indexName string, docs []domain.Document) error {
						require.Len(t, docs, 3)
						return nil
					}).Times(1)
				roadmapSvc.EXPECT().ListSince(gomock.Any(), int64(1000), 2, 100).
					Return([]roadmap.Roadmap{}, nil).Times(1)
			},
		},
		{
			name:      "查询错误",
			startTime: 1000,
			setup: func(roadmapSvc *roadmapmocks.MockAdminService, kbaseSvc *kbasemocks.MockService) {
				roadmapSvc.EXPECT().ListSince(gomock.Any(), int64(1000), 0, 100).
					Return(nil, errors.New("查询失败")).Times(1)
			},
			wantErr: errors.New("查询失败"),
		},
		{
			name:      "BulkUpsert错误-在循环内部触发",
			startTime: 1000,
			setup: func(roadmapSvc *roadmapmocks.MockAdminService, kbaseSvc *kbasemocks.MockService) {
				// 创建 101 个 edges 来触发 batchSize 检查（需要超过 100 才能在循环内部触发）
				edges := make([]roadmap.Edge, 0, 101)
				for i := 0; i < 101; i++ {
					edges = append(edges, roadmap.Edge{
						Id:  int64(i + 1),
						Src: roadmap.Node{ID: 10, Rid: 1},
						Dst: roadmap.Node{ID: 20, Rid: 1},
					})
				}
				roadmapSvc.EXPECT().ListSince(gomock.Any(), int64(1000), 0, 100).
					Return([]roadmap.Roadmap{
						{
							Id:    1,
							Biz:   roadmap.Biz{Biz: "questionSet", BizId: 100},
							Edges: edges,
						},
					}, nil).Times(1)
				// 当处理第 101 个 edge 时，batchDocs 已经有 100 个，会触发 BulkUpsert
				kbaseSvc.EXPECT().BulkUpsert(gomock.Any(), "question_rel_index", gomock.Any()).
					DoAndReturn(func(ctx context.Context, indexName string, docs []domain.Document) error {
						require.Len(t, docs, 100)
						return errors.New("ES错误")
					}).Times(1)
			},
			wantErr: errors.New("ES错误"),
		},
		{
			name:      "BulkUpsert错误-在处理剩余batchDocs时触发",
			startTime: 1000,
			setup: func(roadmapSvc *roadmapmocks.MockAdminService, kbaseSvc *kbasemocks.MockService) {
				roadmapSvc.EXPECT().ListSince(gomock.Any(), int64(1000), 0, 100).
					Return([]roadmap.Roadmap{
						{
							Id:  1,
							Biz: roadmap.Biz{Biz: "questionSet", BizId: 100},
							Edges: []roadmap.Edge{
								{Id: 1, Src: roadmap.Node{ID: 10, Rid: 1}, Dst: roadmap.Node{ID: 20, Rid: 1}},
							},
						},
					}, nil).Times(1)
				roadmapSvc.EXPECT().ListSince(gomock.Any(), int64(1000), 1, 100).
					Return([]roadmap.Roadmap{}, nil).Times(1)
				// 处理完所有 roadmaps 后，调用 BulkUpsert 处理剩余的 batchDocs
				kbaseSvc.EXPECT().BulkUpsert(gomock.Any(), "question_rel_index", gomock.Any()).
					Return(errors.New("ES错误")).Times(1)
			},
			wantErr: errors.New("ES错误"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			roadmapSvc := roadmapmocks.NewMockAdminService(ctrl)
			kbaseSvc := kbasemocks.NewMockService(ctrl)

			syncer := NewQuestionRelSyncer("question_rel_index", 100, roadmapSvc, kbaseSvc)

			tc.setup(roadmapSvc, kbaseSvc)

			err := syncer.UpsertSince(t.Context(), tc.startTime)
			if tc.wantErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestQuestionRelSyncer_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	roadmapSvc := roadmapmocks.NewMockAdminService(ctrl)
	kbaseSvc := kbasemocks.NewMockService(ctrl)

	syncer := NewQuestionRelSyncer("question_rel_index", 100, roadmapSvc, kbaseSvc)

	kbaseSvc.EXPECT().BulkDelete(gomock.Any(), "question_rel_index", []string{"123"}).
		Return(nil).Times(1)

	err := syncer.Delete(t.Context(), 123)
	assert.NoError(t, err)
}

func TestQuestionRelSyncer_toKbaseDocuments(t *testing.T) {
	syncer := NewQuestionRelSyncer("question_rel_index", 100, nil, nil)

	edges := []roadmap.Edge{
		{
			Id:    1,
			Type:  "prerequisite",
			Attrs: `{"weight": 1}`,
			Src: roadmap.Node{
				ID:    100,
				Title: "源节点",
				Rid:   123,
				Biz: roadmap.Biz{
					Biz:   "question",
					BizId: 100,
					Title: "源题目",
				},
			},
			Dst: roadmap.Node{
				ID:    200,
				Title: "目标节点",
				Rid:   123,
				Biz: roadmap.Biz{
					Biz:   "question",
					BizId: 200,
					Title: "目标题目",
				},
			},
		},
		{
			Id:    2,
			Type:  "related",
			Attrs: `{"weight": 2}`,
			Src: roadmap.Node{
				ID:    300,
				Title: "源节点2",
				Rid:   123,
				Biz: roadmap.Biz{
					Biz:   "question",
					BizId: 300,
					Title: "源题目2",
				},
			},
			Dst: roadmap.Node{
				ID:    400,
				Title: "目标节点2",
				Rid:   123,
				Biz: roadmap.Biz{
					Biz:   "question",
					BizId: 400,
					Title: "目标题目2",
				},
			},
		},
	}

	docs := syncer.toKbaseDocuments("questionSet", 456, edges)

	require.Len(t, docs, 2)

	// 验证第一个文档
	doc1 := docs[0]
	assert.Equal(t, "1", doc1.ID)
	assert.Equal(t, int64(123), doc1.Body["rid"])
	assert.Equal(t, "questionSet", doc1.Body["biz"])
	assert.Equal(t, int64(456), doc1.Body["biz_id"])
	assert.Equal(t, "prerequisite", doc1.Body["type"])
	assert.Equal(t, `{"weight": 1}`, doc1.Body["attrs"])
	assert.Equal(t, int64(100), doc1.Body["src_id"])
	assert.Equal(t, "源节点", doc1.Body["src_title"])
	assert.Equal(t, int64(200), doc1.Body["dst_id"])
	assert.Equal(t, "目标节点", doc1.Body["dst_title"])

	// 验证第二个文档
	doc2 := docs[1]
	assert.Equal(t, "2", doc2.ID)
	assert.Equal(t, "related", doc2.Body["type"])
	assert.Equal(t, `{"weight": 2}`, doc2.Body["attrs"])
}
