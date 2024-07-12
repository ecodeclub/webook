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

//go:build e2e

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"
	"github.com/ecodeclub/webook/internal/pkg/middleware"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/event"
	eveMocks "github.com/ecodeclub/webook/internal/question/internal/event/mocks"
	"github.com/ecodeclub/webook/internal/question/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/question/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/question/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type SetHandlerTestSuite struct {
	BaseTestSuite
	server         *egin.Component
	rdb            ecache.Cache
	dao            dao.QuestionDAO
	questionSetDAO dao.QuestionSetDAO
	ctrl           *gomock.Controller
	producer       *eveMocks.MockSyncEventProducer
}

func (s *SetHandlerTestSuite) SetupSuite() {
	ctrl := gomock.NewController(s.T())
	s.producer = eveMocks.NewMockSyncEventProducer(ctrl)

	intrSvc := intrmocks.NewMockService(ctrl)
	intrModule := &interactive.Module{
		Svc: intrSvc,
	}

	// 模拟返回的数据
	// 使用如下规律:
	// 1. liked == id % 2 == 1 (奇数为 true)
	// 2. collected = id %2 == 0 (偶数为 true)
	// 3. viewCnt = id + 1
	// 4. likeCnt = id + 2
	// 5. collectCnt = id + 3
	intrSvc.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().DoAndReturn(func(ctx context.Context,
		biz string, id int64, uid int64) (interactive.Interactive, error) {
		intr := s.mockInteractive(biz, id)
		return intr, nil
	})
	intrSvc.EXPECT().GetByIds(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context,
		biz string, ids []int64) (map[int64]interactive.Interactive, error) {
		res := make(map[int64]interactive.Interactive, len(ids))
		for _, id := range ids {
			intr := s.mockInteractive(biz, id)
			res[id] = intr
		}
		return res, nil
	}).AnyTimes()

	module, err := startup.InitModule(s.producer, intrModule)
	require.NoError(s.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()

	module.QsHdl.PublicRoutes(server.Engine)
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: uid,
			Data: map[string]string{
				"creator":   "true",
				"memberDDL": strconv.FormatInt(time.Now().Add(time.Hour).UnixMilli(), 10),
			},
		}))
	})
	module.QsHdl.PrivateRoutes(server.Engine)
	server.Use(middleware.NewCheckMembershipMiddlewareBuilder(nil).Build())

	s.server = server
	s.db = testioc.InitDB()
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewGORMQuestionDAO(s.db)
	s.questionSetDAO = dao.NewGORMQuestionSetDAO(s.db)
	s.rdb = testioc.InitCache()
}

func (s *SetHandlerTestSuite) TestQuestionSet_Save() {
	var testCases = []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)
		req    web.SaveQuestionSetReq

		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "创建成功1",
			before: func(t *testing.T) {
				s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)
			},
			after: func(t *testing.T) {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				qs, err := s.questionSetDAO.GetByID(ctx, 1)
				assert.NoError(t, err)

				s.assertQuestionSetEqual(t, dao.QuestionSet{
					Uid:         uid,
					Title:       "mysql",
					Description: "mysql相关面试题",
				}, qs)
			},
			req: web.SaveQuestionSetReq{
				Title:       "mysql",
				Description: "mysql相关面试题",
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		},
		{
			name: "创建成功2",
			before: func(t *testing.T) {
				s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(dao.QuestionSet{
					Id:          2,
					Uid:         uid,
					Title:       "老的 MySQL",
					Description: "老的 Desc",
					Ctime:       123,
					Utime:       123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				qs, err := s.questionSetDAO.GetByID(ctx, 2)
				assert.NoError(t, err)
				s.assertQuestionSetEqual(t, dao.QuestionSet{
					Uid:         uid,
					Title:       "mq",
					Description: "mq相关面试题",
				}, qs)
			},
			req: web.SaveQuestionSetReq{
				Id:          2,
				Title:       "mq",
				Description: "mq相关面试题",
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 2,
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			targeURL := "/question-sets/save"
			req, err := http.NewRequest(http.MethodPost, targeURL, iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)

			recorder := test.NewJSONResponseRecorder[int64]()

			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
		})
	}
}

func (s *SetHandlerTestSuite) TestQuestionSet_UpdateQuestions() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)
		req    web.UpdateQuestionsOfQuestionSetReq

		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "空题集_添加多个问题",
			before: func(t *testing.T) {
				t.Helper()
				s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)

				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				// 创建一个空题集
				id, err := s.questionSetDAO.Create(ctx, dao.QuestionSet{
					Id:          5,
					Uid:         uid,
					Title:       "oss",
					Description: "oss题集",
				})
				require.NoError(t, err)
				require.Equal(t, int64(5), id)

				// 创建问题
				questions := []dao.Question{
					{
						Id:      4,
						Uid:     uid + 1,
						Title:   "oss问题1",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Content: "oss问题1",
						Ctime:   123,
						Utime:   234,
					},
					{
						Id:      5,
						Uid:     uid + 2,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "oss问题2",
						Content: "oss问题2",
						Ctime:   1234,
						Utime:   2345,
					},
				}
				for _, q := range questions {
					require.NoError(t, s.db.WithContext(ctx).Create(&q).Error)
				}

				// 题集中题目为0
				qs, err := s.questionSetDAO.GetQuestionsByID(ctx, id)
				require.NoError(t, err)
				require.Equal(t, 0, len(qs))
			},
			after: func(t *testing.T) {
				t.Helper()

				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				expected := []dao.Question{
					{
						Uid:     uid + 1,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "oss问题1",
						Content: "oss问题1",
					},
					{
						Uid:     uid + 2,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "oss问题2",
						Content: "oss问题2",
					},
				}

				actual, err := s.questionSetDAO.GetQuestionsByID(ctx, 5)
				require.NoError(t, err)
				require.Equal(t, len(expected), len(actual))

				for i := 0; i < len(expected); i++ {
					s.assertQuestion(t, expected[i], actual[i])
				}

			},
			req: web.UpdateQuestionsOfQuestionSetReq{
				QSID: 5,
				QIDs: []int64{4, 5},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{},
		},
		{
			name: "非空题集_添加多个问题",
			before: func(t *testing.T) {
				t.Helper()
				s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)

				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				// 创建一个空题集
				id, err := s.questionSetDAO.Create(ctx, dao.QuestionSet{
					Id:          7,
					Uid:         uid,
					Title:       "Go",
					Description: "Go题集",
				})
				require.NoError(t, err)
				require.Equal(t, int64(7), id)

				// 创建问题
				questions := []dao.Question{
					{
						Id:      14,
						Uid:     uid + 1,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题1",
						Content: "Go问题1",
						Ctime:   123,
						Utime:   234,
					},
					{
						Id:      15,
						Uid:     uid + 2,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题2",
						Content: "Go问题2",
						Ctime:   1234,
						Utime:   2345,
					},
					{
						Id:      16,
						Uid:     uid + 3,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题3",
						Content: "Go问题3",
						Ctime:   1234,
						Utime:   2345,
					},
				}
				for _, q := range questions {
					require.NoError(t, s.db.WithContext(ctx).Create(&q).Error)
				}

				require.NoError(t, s.questionSetDAO.UpdateQuestionsByID(ctx, id, []int64{14}))

				// 题集中题目为1
				qs, err := s.questionSetDAO.GetQuestionsByID(ctx, id)
				require.NoError(t, err)
				require.Equal(t, 1, len(qs))
			},
			after: func(t *testing.T) {
				t.Helper()

				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				expected := []dao.Question{
					{
						Uid:     uid + 1,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题1",
						Content: "Go问题1",
					},
					{
						Uid:     uid + 2,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题2",
						Content: "Go问题2",
					},
					{
						Uid:     uid + 3,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题3",
						Content: "Go问题3",
					},
				}

				actual, err := s.questionSetDAO.GetQuestionsByID(ctx, 7)
				require.NoError(t, err)
				require.Equal(t, len(expected), len(actual))

				for i := 0; i < len(expected); i++ {
					s.assertQuestion(t, expected[i], actual[i])
				}

			},
			req: web.UpdateQuestionsOfQuestionSetReq{
				QSID: 7,
				QIDs: []int64{14, 15, 16},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{},
		},
		{
			name: "非空题集_删除全部问题",
			before: func(t *testing.T) {
				t.Helper()
				s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)

				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				// 创建一个空题集
				id, err := s.questionSetDAO.Create(ctx, dao.QuestionSet{
					Id:          217,
					Uid:         uid,
					Title:       "Go",
					Description: "Go题集",
				})
				require.Equal(t, int64(217), id)
				require.NoError(t, err)

				// 创建问题
				questions := []dao.Question{
					{
						Id:      214,
						Uid:     uid + 1,
						Title:   "Go问题1",
						Content: "Go问题1",
						Ctime:   123,
						Utime:   234,
					},
					{
						Id:      215,
						Uid:     uid + 2,
						Title:   "Go问题2",
						Content: "Go问题2",
						Ctime:   1234,
						Utime:   2345,
					},
					{
						Id:      216,
						Uid:     uid + 2,
						Title:   "Go问题3",
						Content: "Go问题3",
						Ctime:   1234,
						Utime:   2345,
					},
				}
				for _, q := range questions {
					require.NoError(t, s.db.WithContext(ctx).Create(&q).Error)
				}

				require.NoError(t, s.questionSetDAO.UpdateQuestionsByID(ctx, id, []int64{214, 215, 216}))

				qs, err := s.questionSetDAO.GetQuestionsByID(ctx, id)
				require.NoError(t, err)
				require.Equal(t, len(questions), len(qs))

			},
			after: func(t *testing.T) {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				qs, err := s.questionSetDAO.GetQuestionsByID(ctx, 217)
				require.NoError(t, err)
				require.Equal(t, 0, len(qs))
			},
			req: web.UpdateQuestionsOfQuestionSetReq{
				QSID: 217,
				QIDs: []int64{},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{},
		},
		{
			name: "非空题集_删除部分问题",
			before: func(t *testing.T) {
				t.Helper()
				s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)

				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				// 创建一个空题集
				id, err := s.questionSetDAO.Create(ctx, dao.QuestionSet{
					Id:          218,
					Uid:         uid,
					Title:       "Go",
					Description: "Go题集",
				})
				require.Equal(t, int64(218), id)
				require.NoError(t, err)

				// 创建问题
				questions := []dao.Question{
					{
						Id:      314,
						Uid:     uid + 1,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题1",
						Content: "Go问题1",
						Ctime:   123,
						Utime:   234,
					},
					{
						Id:      315,
						Uid:     uid + 2,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题2",
						Content: "Go问题2",
						Ctime:   1234,
						Utime:   2345,
					},
					{
						Id:      316,
						Uid:     uid + 2,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题3",
						Content: "Go问题3",
						Ctime:   1234,
						Utime:   2345,
					},
				}
				for _, q := range questions {
					require.NoError(t, s.db.WithContext(ctx).Create(&q).Error)
				}

				require.NoError(t, s.questionSetDAO.UpdateQuestionsByID(ctx, id, []int64{314, 315, 316}))

				qs, err := s.questionSetDAO.GetQuestionsByID(ctx, id)
				require.NoError(t, err)
				require.Equal(t, len(questions), len(qs))
			},
			after: func(t *testing.T) {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				qs, err := s.questionSetDAO.GetQuestionsByID(ctx, 218)
				require.NoError(t, err)
				require.Equal(t, 1, len(qs))
				s.assertQuestion(t, dao.Question{
					Uid:     uid + 2,
					Status:  domain.UnPublishedStatus.ToUint8(),
					Title:   "Go问题2",
					Content: "Go问题2",
				}, qs[0])

			},
			req: web.UpdateQuestionsOfQuestionSetReq{
				QSID: 218,
				QIDs: []int64{315},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{},
		},
		{
			name: "同时添加/删除部分问题",
			before: func(t *testing.T) {
				t.Helper()
				s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)

				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				// 创建一个空题集
				id, err := s.questionSetDAO.Create(ctx, dao.QuestionSet{
					Id:          219,
					Uid:         uid,
					Title:       "Go",
					Description: "Go题集",
				})
				require.Equal(t, int64(219), id)
				require.NoError(t, err)

				// 创建问题
				questions := []dao.Question{
					{
						Id:      414,
						Uid:     uid + 1,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题1",
						Content: "Go问题1",
						Ctime:   123,
						Utime:   234,
					},
					{
						Id:      415,
						Uid:     uid + 2,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题2",
						Content: "Go问题2",
						Ctime:   1234,
						Utime:   2345,
					},
					{
						Id:      416,
						Uid:     uid + 3,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题3",
						Content: "Go问题3",
						Ctime:   1234,
						Utime:   2345,
					},
					{
						Id:      417,
						Uid:     uid + 4,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题4",
						Content: "Go问题4",
						Ctime:   1234,
						Utime:   2345,
					},
				}
				for _, q := range questions {
					require.NoError(t, s.db.WithContext(ctx).Create(&q).Error)
				}

				qids := []int64{414, 415}
				require.NoError(t, s.questionSetDAO.UpdateQuestionsByID(ctx, id, qids))

				qs, err := s.questionSetDAO.GetQuestionsByID(ctx, id)
				require.NoError(t, err)
				require.Equal(t, len(qids), len(qs))
			},
			after: func(t *testing.T) {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				expected := []dao.Question{
					{
						Uid:     uid + 2,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题2",
						Content: "Go问题2",
					},
					{
						Uid:     uid + 3,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题3",
						Content: "Go问题3",
					},
					{
						Uid:     uid + 4,
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题4",
						Content: "Go问题4",
					},
				}

				qs, err := s.questionSetDAO.GetQuestionsByID(ctx, 219)
				require.NoError(t, err)

				require.Equal(t, len(expected), len(qs))

				for i, e := range expected {
					s.assertQuestion(t, e, qs[i])
				}
			},
			req: web.UpdateQuestionsOfQuestionSetReq{
				QSID: 219,
				QIDs: []int64{415, 416, 417},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{},
		},
		{
			name: "题集不存在",
			before: func(t *testing.T) {
				t.Helper()
			},
			after: func(t *testing.T) {
				t.Helper()
			},
			req: web.UpdateQuestionsOfQuestionSetReq{
				QSID: 10000,
				QIDs: []int64{},
			},
			wantCode: 500,
			wantResp: test.Result[int64]{Code: 502001, Msg: "系统错误"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/question-sets/questions/save", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
		})
	}
}

func (s *SetHandlerTestSuite) TestQuestionSet_Detail() {

	var now int64 = 123

	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)
		req    web.QuestionSetID

		wantCode int
		wantResp test.Result[web.QuestionSet]
	}{
		{
			name: "空题集",
			before: func(t *testing.T) {
				t.Helper()

				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				// 创建一个空题集
				id, err := s.questionSetDAO.Create(ctx, dao.QuestionSet{
					Id:          321,
					Uid:         uid,
					Title:       "Go",
					Description: "Go题集",
					Utime:       now,
				})
				require.NoError(t, err)
				require.Equal(t, int64(321), id)
			},
			after: func(t *testing.T) {
				t.Helper()
			},
			req: web.QuestionSetID{
				QSID: 321,
			},
			wantCode: 200,
			wantResp: test.Result[web.QuestionSet]{
				Data: web.QuestionSet{
					Id:          321,
					Title:       "Go",
					Description: "Go题集",
					Utime:       now,
					Interactive: web.Interactive{
						ViewCnt:    322,
						LikeCnt:    323,
						CollectCnt: 324,
						Liked:      true,
						Collected:  false,
					},
				},
			},
		},
		{
			name: "非空题集",
			before: func(t *testing.T) {
				t.Helper()

				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				// 创建一个空题集
				id, err := s.questionSetDAO.Create(ctx, dao.QuestionSet{
					Id:          322,
					Uid:         uid,
					Title:       "Go",
					Description: "Go题集",
					Utime:       now,
				})
				require.NoError(t, err)
				require.Equal(t, int64(322), id)

				// 添加问题
				questions := []dao.Question{
					{
						Id:      614,
						Uid:     uid + 1,
						Title:   "Go问题1",
						Content: "Go问题1",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:      615,
						Uid:     uid + 2,
						Title:   "Go问题2",
						Content: "Go问题2",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:      616,
						Uid:     uid + 3,
						Title:   "Go问题3",
						Content: "Go问题3",
						Ctime:   now,
						Utime:   now,
					},
				}
				err = s.db.WithContext(ctx).Create(&questions).Error
				require.NoError(t, err)
				qids := []int64{614, 615, 616}
				require.NoError(t, s.questionSetDAO.UpdateQuestionsByID(ctx, id, qids))

				// 添加用户答题记录，只需要添加一个就可以
				err = s.db.WithContext(ctx).Create(&dao.QuestionResult{
					Uid:    uid,
					Qid:    614,
					Result: domain.ResultAdvanced.ToUint8(),
					Ctime:  now,
					Utime:  now,
				}).Error
				require.NoError(t, err)

				// 题集中题目为1
				qs, err := s.questionSetDAO.GetQuestionsByID(ctx, id)
				require.NoError(t, err)
				require.Equal(t, len(qids), len(qs))
			},
			after: func(t *testing.T) {
			},
			req: web.QuestionSetID{
				QSID: 322,
			},
			wantCode: 200,
			wantResp: test.Result[web.QuestionSet]{
				Data: web.QuestionSet{
					Id:          322,
					Title:       "Go",
					Description: "Go题集",
					Interactive: web.Interactive{
						ViewCnt:    323,
						LikeCnt:    324,
						CollectCnt: 325,
						Liked:      false,
						Collected:  true,
					},
					Questions: []web.Question{
						{
							Id:            614,
							Title:         "Go问题1",
							Content:       "Go问题1",
							ExamineResult: domain.ResultAdvanced.ToUint8(),
							Utime:         now,
						},
						{
							Id:      615,
							Title:   "Go问题2",
							Content: "Go问题2",
							Utime:   now,
						},
						{
							Id:      616,
							Title:   "Go问题3",
							Content: "Go问题3",
							Utime:   now,
						},
					},
					Utime: now,
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/question-sets/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.QuestionSet]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
		})
	}
}

func (s *SetHandlerTestSuite) TestQuestionSet_RetrieveQuestionSetDetail_Failed() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)
		req    web.QuestionSetID

		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "题集ID非法_题集ID不存在",
			before: func(t *testing.T) {
				t.Helper()
			},
			after: func(t *testing.T) {
				t.Helper()
			},
			req: web.QuestionSetID{
				QSID: 10000,
			},
			wantCode: 500,
			wantResp: test.Result[int64]{Code: 502001, Msg: "系统错误"},
		},
	}
	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/question-sets/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
		})
	}
}

func (s *SetHandlerTestSuite) TestQuestionSet_ListAllQuestionSets() {
	// 插入一百条
	total := 100
	data := make([]dao.QuestionSet, 0, total)

	for idx := 0; idx < total; idx++ {
		// 空题集
		data = append(data, dao.QuestionSet{
			Uid:         int64(uid + idx),
			Title:       fmt.Sprintf("题集标题 %d", idx),
			Description: fmt.Sprintf("题集简介 %d", idx),
			Utime:       123,
		})
	}
	err := s.db.Create(&data).Error
	require.NoError(s.T(), err)

	testCases := []struct {
		name string
		req  web.Page

		wantCode int
		wantResp test.Result[web.QuestionSetList]
	}{
		{
			name: "获取成功",
			req: web.Page{
				Limit:  2,
				Offset: 0,
			},
			wantCode: 200,
			wantResp: test.Result[web.QuestionSetList]{
				Data: web.QuestionSetList{
					Total: int64(total),
					QuestionSets: []web.QuestionSet{
						{
							Id:          100,
							Title:       "题集标题 99",
							Description: "题集简介 99",
							Utime:       123,
							Interactive: web.Interactive{
								ViewCnt:    101,
								LikeCnt:    102,
								CollectCnt: 103,
								Liked:      false,
								Collected:  true,
							},
						},
						{
							Id:          99,
							Title:       "题集标题 98",
							Description: "题集简介 98",
							Utime:       123,
							Interactive: web.Interactive{
								ViewCnt:    100,
								LikeCnt:    101,
								CollectCnt: 102,
								Liked:      true,
								Collected:  false,
							},
						},
					},
				},
			},
		},
		{
			name: "获取部分",
			req: web.Page{
				Limit:  2,
				Offset: 99,
			},
			wantCode: 200,
			wantResp: test.Result[web.QuestionSetList]{
				Data: web.QuestionSetList{
					Total: int64(total),
					QuestionSets: []web.QuestionSet{
						{
							Id:          1,
							Title:       "题集标题 0",
							Description: "题集简介 0",
							Utime:       123,
							Interactive: web.Interactive{
								ViewCnt:    2,
								LikeCnt:    3,
								CollectCnt: 4,
								Liked:      true,
								Collected:  false,
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/question-sets/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.QuestionSetList]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *SetHandlerTestSuite) TestQuestionSetEvent() {
	t := s.T()
	ans := make([]event.QuestionSet, 0, 16)
	mu := &sync.RWMutex{}
	s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, questionEvent event.QuestionEvent) error {
		var eve event.QuestionSet
		err := json.Unmarshal([]byte(questionEvent.Data), &eve)
		if err != nil {
			return err
		}
		mu.Lock()
		ans = append(ans, eve)
		mu.Unlock()
		return nil
	}).Times(2)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	_, err := s.dao.Create(ctx, dao.Question{
		Id: 1,
	}, []dao.AnswerElement{
		{
			Content: "ele",
		},
	})
	assert.NoError(t, err)
	_, err = s.dao.Create(ctx, dao.Question{
		Id: 2,
	}, []dao.AnswerElement{
		{
			Content: "ele",
		},
	})
	assert.NoError(t, err)
	// 保存
	saveReq := web.SaveQuestionSetReq{
		Title:       "questionSet1",
		Description: "question_description1",
	}
	req, err := http.NewRequest(http.MethodPost,
		"/question-sets/save", iox.NewJSONReader(saveReq))
	req.Header.Set("content-type", "application/json")
	assert.NoError(t, err)
	recorder := test.NewJSONResponseRecorder[int64]()
	s.server.ServeHTTP(recorder, req)
	assert.Equal(t, 200, recorder.Code)
	// 更新
	syncReq := &web.UpdateQuestionsOfQuestionSetReq{
		QSID: 1,
		QIDs: []int64{1, 2},
	}
	req2, err := http.NewRequest(http.MethodPost,
		"/question-sets/questions/save", iox.NewJSONReader(syncReq))
	req2.Header.Set("content-type", "application/json")
	assert.NoError(t, err)
	recorder = test.NewJSONResponseRecorder[int64]()
	s.server.ServeHTTP(recorder, req2)
	assert.Equal(t, 200, recorder.Code)
	time.Sleep(1 * time.Second)
	for idx := range ans {
		ans[idx].Id = 0
		ans[idx].Utime = 0
	}
	assert.Equal(t, []event.QuestionSet{
		{
			Uid:         uid,
			Title:       "questionSet1",
			Description: "question_description1",
			Questions:   []int64{},
		},
		{
			Uid:         uid,
			Title:       "questionSet1",
			Description: "question_description1",
			Questions:   []int64{1, 2},
		},
	}, ans)
}

func TestSetHandler(t *testing.T) {
	suite.Run(t, new(SetHandlerTestSuite))
}
