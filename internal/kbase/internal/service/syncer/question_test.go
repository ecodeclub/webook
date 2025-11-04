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
	"time"

	"github.com/ecodeclub/webook/internal/kbase/internal/domain"
	kbasemocks "github.com/ecodeclub/webook/internal/kbase/mocks"
	baguwen "github.com/ecodeclub/webook/internal/question"
	quemocks "github.com/ecodeclub/webook/internal/question/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestQuestionSyncer_Upsert(t *testing.T) {
	testCases := []struct {
		name    string
		id      int64
		setup   func(*quemocks.MockService, *kbasemocks.MockService)
		wantErr error
	}{
		{
			name: "成功",
			id:   123,
			setup: func(queSvc *quemocks.MockService, kbaseSvc *kbasemocks.MockService) {
				queSvc.EXPECT().PubDetailWithoutCntView(gomock.Any(), int64(123)).
					Return(baguwen.Question{
						Id:      123,
						Title:   "测试题目",
						Biz:     "baguwen",
						BizId:   0,
						Labels:  []string{"test"},
						Content: "测试内容",
						Status:  2, // PublishedStatus
						Answer: baguwen.Answer{
							Analysis: baguwen.AnswerElement{
								Id:        1,
								Content:   "分析内容",
								Keywords:  "关键词",
								Shorthand: "速记",
								Highlight: "亮点",
								Guidance:  "引导",
							},
							Utime: time.Unix(1000, 0),
						},
						Utime: time.Unix(2000, 0),
					}, nil).Times(1)
				kbaseSvc.EXPECT().BulkUpsert(gomock.Any(), "question_index", gomock.Any()).
					DoAndReturn(func(ctx context.Context, indexName string, docs []domain.Document) error {
						require.Len(t, docs, 1)
						doc := docs[0]
						assert.Equal(t, "123", doc.ID)
						assert.Equal(t, int64(123), doc.Body["id"])
						assert.Equal(t, "测试题目", doc.Body["title"])
						assert.Equal(t, "baguwen", doc.Body["biz"])
						return nil
					}).Times(1)
			},
		},
		{
			name: "question不存在",
			id:   123,
			setup: func(queSvc *quemocks.MockService, kbaseSvc *kbasemocks.MockService) {
				queSvc.EXPECT().PubDetailWithoutCntView(gomock.Any(), int64(123)).
					Return(baguwen.Question{}, errors.New("question not found")).Times(1)
			},
			wantErr: errors.New("question not found"),
		},
		{
			name: "kbase service错误",
			id:   123,
			setup: func(queSvc *quemocks.MockService, kbaseSvc *kbasemocks.MockService) {
				queSvc.EXPECT().PubDetailWithoutCntView(gomock.Any(), int64(123)).
					Return(baguwen.Question{
						Id:    123,
						Title: "测试题目",
					}, nil).Times(1)
				kbaseSvc.EXPECT().BulkUpsert(gomock.Any(), "question_index", gomock.Any()).
					Return(errors.New("ES错误")).Times(1)
			},
			wantErr: errors.New("ES错误"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			queSvc := quemocks.NewMockService(ctrl)
			kbaseSvc := kbasemocks.NewMockService(ctrl)

			syncer := NewQuestionSyncer("question_index", 100, queSvc, kbaseSvc)

			tc.setup(queSvc, kbaseSvc)

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

func TestQuestionSyncer_UpsertSince(t *testing.T) {
	testCases := []struct {
		name      string
		startTime int64
		setup     func(*quemocks.MockService, *kbasemocks.MockService)
		wantErr   error
	}{
		{
			name:      "空数据",
			startTime: 1000,
			setup: func(queSvc *quemocks.MockService, kbaseSvc *kbasemocks.MockService) {
				queSvc.EXPECT().ListPubSince(gomock.Any(), int64(1000), 0, 100).
					Return([]baguwen.Question{}, nil).Times(1)
			},
		},
		{
			name:      "单页数据",
			startTime: 1000,
			setup: func(queSvc *quemocks.MockService, kbaseSvc *kbasemocks.MockService) {
				queSvc.EXPECT().ListPubSince(gomock.Any(), int64(1000), 0, 100).
					Return([]baguwen.Question{
						{Id: 1, Title: "题目1", Utime: time.Unix(1001, 0)},
						{Id: 2, Title: "题目2", Utime: time.Unix(1002, 0)},
					}, nil).Times(1)
				kbaseSvc.EXPECT().BulkUpsert(gomock.Any(), "question_index", gomock.Any()).
					DoAndReturn(func(ctx context.Context, indexName string, docs []domain.Document) error {
						require.Len(t, docs, 2)
						return nil
					}).Times(1)
				// 继续查询下一页（空）
				queSvc.EXPECT().ListPubSince(gomock.Any(), int64(1000), 2, 100).
					Return([]baguwen.Question{}, nil).Times(1)
			},
		},
		{
			name:      "多页数据",
			startTime: 1000,
			setup: func(queSvc *quemocks.MockService, kbaseSvc *kbasemocks.MockService) {
				// 第一页
				queSvc.EXPECT().ListPubSince(gomock.Any(), int64(1000), 0, 100).
					Return([]baguwen.Question{
						{Id: 1, Title: "题目1", Utime: time.Unix(1001, 0)},
					}, nil).Times(1)
				kbaseSvc.EXPECT().BulkUpsert(gomock.Any(), "question_index", gomock.Any()).
					DoAndReturn(func(ctx context.Context, indexName string, docs []domain.Document) error {
						require.Len(t, docs, 1)
						return nil
					}).Times(1)
				// 第二页
				queSvc.EXPECT().ListPubSince(gomock.Any(), int64(1000), 1, 100).
					Return([]baguwen.Question{
						{Id: 2, Title: "题目2", Utime: time.Unix(1002, 0)},
					}, nil).Times(1)
				kbaseSvc.EXPECT().BulkUpsert(gomock.Any(), "question_index", gomock.Any()).
					DoAndReturn(func(ctx context.Context, indexName string, docs []domain.Document) error {
						require.Len(t, docs, 1)
						return nil
					}).Times(1)
				// 第三页（空）
				queSvc.EXPECT().ListPubSince(gomock.Any(), int64(1000), 2, 100).
					Return([]baguwen.Question{}, nil).Times(1)
			},
		},
		{
			name:      "查询错误",
			startTime: 1000,
			setup: func(queSvc *quemocks.MockService, kbaseSvc *kbasemocks.MockService) {
				queSvc.EXPECT().ListPubSince(gomock.Any(), int64(1000), 0, 100).
					Return(nil, errors.New("查询失败")).Times(1)
			},
			wantErr: errors.New("查询失败"),
		},
		{
			name:      "BulkUpsert错误后继续",
			startTime: 1000,
			setup: func(queSvc *quemocks.MockService, kbaseSvc *kbasemocks.MockService) {
				// 第一页失败（continue，offset 不更新）
				queSvc.EXPECT().ListPubSince(gomock.Any(), int64(1000), 0, 100).
					Return([]baguwen.Question{
						{Id: 1, Title: "题目1", Utime: time.Unix(1001, 0)},
					}, nil).Times(1)
				kbaseSvc.EXPECT().BulkUpsert(gomock.Any(), "question_index", gomock.Any()).
					Return(errors.New("ES错误")).Times(1)
				// 继续查询下一页（offset 仍然是 0，因为 continue 没有更新 offset）
				queSvc.EXPECT().ListPubSince(gomock.Any(), int64(1000), 0, 100).
					Return([]baguwen.Question{}, nil).Times(1)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			queSvc := quemocks.NewMockService(ctrl)
			kbaseSvc := kbasemocks.NewMockService(ctrl)

			syncer := NewQuestionSyncer("question_index", 100, queSvc, kbaseSvc)

			tc.setup(queSvc, kbaseSvc)

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

func TestQuestionSyncer_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	queSvc := quemocks.NewMockService(ctrl)
	kbaseSvc := kbasemocks.NewMockService(ctrl)

	syncer := NewQuestionSyncer("question_index", 100, queSvc, kbaseSvc)

	kbaseSvc.EXPECT().BulkDelete(gomock.Any(), "question_index", []string{"123"}).
		Return(nil).Times(1)

	err := syncer.Delete(t.Context(), 123)
	assert.NoError(t, err)
}

func TestQuestionSyncer_toKbaseDocument(t *testing.T) {
	syncer := NewQuestionSyncer("question_index", 100, nil, nil)

	que := baguwen.Question{
		Id:      123,
		Title:   "测试题目",
		Biz:     "baguwen",
		BizId:   0,
		Labels:  []string{"test", "java"},
		Content: "测试内容",
		Status:  2, // PublishedStatus
		Answer: baguwen.Answer{
			Analysis: baguwen.AnswerElement{
				Id:        1,
				Content:   "分析内容",
				Keywords:  "关键词",
				Shorthand: "速记",
				Highlight: "亮点",
				Guidance:  "引导",
			},
			Basic: baguwen.AnswerElement{
				Id:        2,
				Content:   "基础内容",
				Keywords:  "关键词2",
				Shorthand: "速记2",
				Highlight: "亮点2",
				Guidance:  "引导2",
			},
			Intermediate: baguwen.AnswerElement{
				Id:        3,
				Content:   "中级内容",
				Keywords:  "关键词3",
				Shorthand: "速记3",
				Highlight: "亮点3",
				Guidance:  "引导3",
			},
			Advanced: baguwen.AnswerElement{
				Id:        4,
				Content:   "高级内容",
				Keywords:  "关键词4",
				Shorthand: "速记4",
				Highlight: "亮点4",
				Guidance:  "引导4",
			},
			Utime: time.Unix(1000, 0),
		},
		Utime: time.Unix(2000, 0),
	}

	doc := syncer.toKbaseDocument(que)

	assert.Equal(t, "123", doc.ID)
	assert.Equal(t, int64(123), doc.Body["id"])
	assert.Equal(t, "测试题目", doc.Body["title"])
	assert.Equal(t, "baguwen", doc.Body["biz"])
	assert.Equal(t, int64(0), doc.Body["biz_id"])
	assert.Equal(t, []string{"test", "java"}, doc.Body["labels"])
	assert.Equal(t, "测试内容", doc.Body["content"])
	// Status 字段存储的是 QuestionStatus 类型（底层是 uint8）
	// 需要先断言为 QuestionStatus，再转换为 uint8
	statusVal := doc.Body["status"]
	// 使用反射或直接比较值
	statusUint8 := uint8(statusVal.(interface{ ToUint8() uint8 }).ToUint8())
	assert.Equal(t, uint8(2), statusUint8) // PublishedStatus

	// 验证 answer
	answer, ok := doc.Body["answer"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, time.Unix(1000, 0), answer["utime"])

	// 验证 analysis
	analysis, ok := answer["analysis"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "1", analysis["id"])
	assert.Equal(t, "分析内容", analysis["content"])
	assert.Equal(t, "关键词", analysis["keywords"])
	assert.Equal(t, "速记", analysis["shorthand"])
	assert.Equal(t, "亮点", analysis["highlight"])
	assert.Equal(t, "引导", analysis["guidance"])

	// 验证 utime
	assert.Equal(t, time.Unix(2000, 0), doc.Body["utime"])
}

func TestQuestionSyncer_convertAnswerElement2Map(t *testing.T) {
	syncer := NewQuestionSyncer("question_index", 100, nil, nil)

	element := baguwen.AnswerElement{
		Id:        123,
		Content:   "内容",
		Keywords:  "关键词",
		Shorthand: "速记",
		Highlight: "亮点",
		Guidance:  "引导",
	}

	result := syncer.convertAnswerElement2Map(element)

	assert.Equal(t, "123", result["id"])
	assert.Equal(t, "内容", result["content"])
	assert.Equal(t, "关键词", result["keywords"])
	assert.Equal(t, "速记", result["shorthand"])
	assert.Equal(t, "亮点", result["highlight"])
	assert.Equal(t, "引导", result["guidance"])
}
