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
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ecodeclub/webook/internal/member"

	"github.com/ecodeclub/webook/internal/question/internal/service"

	"github.com/ecodeclub/webook/internal/ai"

	"github.com/ecodeclub/webook/internal/permission"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/interactive"
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

type AdminSetHandlerTestSuite struct {
	BaseTestSuite
	server         *egin.Component
	rdb            ecache.Cache
	dao            dao.QuestionDAO
	questionSetDAO dao.QuestionSetDAO
	producer       *eveMocks.MockSyncEventProducer

	setSvc service.QuestionSetService
}

func (s *AdminSetHandlerTestSuite) SetupSuite() {
	ctrl := gomock.NewController(s.T())
	s.producer = eveMocks.NewMockSyncEventProducer(ctrl)

	intrModule := &interactive.Module{}

	module, err := startup.InitModule(s.producer, nil, intrModule,
		&permission.Module{}, &ai.Module{},
		session.DefaultProvider(),
		&member.Module{})
	require.NoError(s.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: uid,
			Data: map[string]string{
				"creator":   "true",
				"memberDDL": strconv.FormatInt(time.Now().Add(time.Hour).UnixMilli(), 10),
			},
		}))
	})
	module.AdminSetHdl.PrivateRoutes(server.Engine)
	server.Use(middleware.NewCheckMembershipMiddlewareBuilder(nil).Build())
	s.setSvc = module.SetSvc
	s.server = server
	s.db = testioc.InitDB()
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewGORMQuestionDAO(s.db)
	s.questionSetDAO = dao.NewGORMQuestionSetDAO(s.db)
	s.rdb = testioc.InitCache()
}

func (s *AdminSetHandlerTestSuite) TestQuestionSet_Save() {
	var testCases = []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)
		req    web.QuestionSet

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
					Biz:         "project",
					BizId:       1,
					Description: "mysql相关面试题",
				}, qs)
			},
			req: web.QuestionSet{
				Title:       "mysql",
				Biz:         "project",
				BizId:       1,
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
					Biz:         "project",
					BizId:       1,
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
					Biz:         "roadmap",
					BizId:       2,
					Description: "mq相关面试题",
				}, qs)
			},
			req: web.QuestionSet{
				Id:          2,
				Title:       "mq",
				Biz:         "roadmap",
				BizId:       2,
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

func (s *AdminSetHandlerTestSuite) TestQuestionSet_Candidates() {
	testCases := []struct {
		name string

		before func(t *testing.T)
		req    web.CandidateReq

		wantCode int
		wantResp test.Result[web.QuestionList]
	}{
		{
			name: "获取成功",
			before: func(t *testing.T) {
				// 准备数据
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				// 创建一个空题集
				id, err := s.questionSetDAO.Create(ctx, dao.QuestionSet{
					Id:          1,
					Uid:         uid,
					Title:       "Go",
					Description: "Go题集",
					Biz:         "roadmap",
					BizId:       2,
					Utime:       123,
				})
				require.NoError(t, err)
				// 添加问题
				questions := []dao.Question{
					s.buildQuestion(1),
					s.buildQuestion(2),
					s.buildQuestion(3),
					s.buildQuestion(4),
					s.buildQuestion(5),
					s.buildQuestion(6),
				}
				err = s.db.WithContext(ctx).Create(&questions).Error
				require.NoError(t, err)
				qids := []int64{1, 2, 3}
				require.NoError(t, s.questionSetDAO.UpdateQuestionsByID(ctx, id, qids))
			},
			req: web.CandidateReq{
				QSID:   1,
				Offset: 1,
				Limit:  2,
			},
			wantCode: 200,
			wantResp: test.Result[web.QuestionList]{
				Data: web.QuestionList{
					Total: 3,
					Questions: []web.Question{
						s.buildWebQuestion(5),
						s.buildWebQuestion(4),
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/question-sets/candidate", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.QuestionList]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *AdminSetHandlerTestSuite) TestQuestionSet_UpdateQuestions() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)
		req    web.UpdateQuestions

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
					Biz:         "baguwen",
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
						Biz:     "baguwen",
						Title:   "oss问题1",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Content: "oss问题1",
						Ctime:   123,
						Utime:   234,
					},
					{
						Id:      5,
						Uid:     uid + 2,
						Biz:     "baguwen",
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
						Biz:     "baguwen",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "oss问题1",
						Content: "oss问题1",
					},
					{
						Uid:     uid + 2,
						Biz:     "baguwen",
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
			req: web.UpdateQuestions{
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
					Biz:         "baguwen",
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
						Biz:     "baguwen",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题1",
						Content: "Go问题1",
						Ctime:   123,
						Utime:   234,
					},
					{
						Id:      15,
						Uid:     uid + 2,
						Biz:     "baguwen",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题2",
						Content: "Go问题2",
						Ctime:   1234,
						Utime:   2345,
					},
					{
						Id:      16,
						Uid:     uid + 3,
						Biz:     "baguwen",
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
						Biz:     "baguwen",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题1",
						Content: "Go问题1",
					},
					{
						Uid:     uid + 2,
						Biz:     "baguwen",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题2",
						Content: "Go问题2",
					},
					{
						Uid:     uid + 3,
						Biz:     "baguwen",
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
			req: web.UpdateQuestions{
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
					Biz:         "baguwen",
					Description: "Go题集",
				})
				require.Equal(t, int64(217), id)
				require.NoError(t, err)

				// 创建问题
				questions := []dao.Question{
					{
						Id:      214,
						Biz:     "baguwen",
						Uid:     uid + 1,
						Title:   "Go问题1",
						Content: "Go问题1",
						Ctime:   123,
						Utime:   234,
					},
					{
						Id:      215,
						Uid:     uid + 2,
						Biz:     "baguwen",
						Title:   "Go问题2",
						Content: "Go问题2",
						Ctime:   1234,
						Utime:   2345,
					},
					{
						Id:      216,
						Uid:     uid + 2,
						Biz:     "baguwen",
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
			req: web.UpdateQuestions{
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
					Biz:         "baguwen",
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
						Biz:     "baguwen",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题1",
						Content: "Go问题1",
						Ctime:   123,
						Utime:   234,
					},
					{
						Id:      315,
						Uid:     uid + 2,
						Biz:     "baguwen",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题2",
						Content: "Go问题2",
						Ctime:   1234,
						Utime:   2345,
					},
					{
						Id:      316,
						Uid:     uid + 2,
						Biz:     "baguwen",
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
					Biz:     "baguwen",
					Status:  domain.UnPublishedStatus.ToUint8(),
					Title:   "Go问题2",
					Content: "Go问题2",
				}, qs[0])

			},
			req: web.UpdateQuestions{
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
					Biz:         "baguwen",
					Description: "Go题集",
				})
				require.Equal(t, int64(219), id)
				require.NoError(t, err)

				// 创建问题
				questions := []dao.Question{
					{
						Id:      414,
						Uid:     uid + 1,
						Biz:     "baguwen",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题1",
						Content: "Go问题1",
						Ctime:   123,
						Utime:   234,
					},
					{
						Id:      415,
						Uid:     uid + 2,
						Biz:     "baguwen",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题2",
						Content: "Go问题2",
						Ctime:   1234,
						Utime:   2345,
					},
					{
						Id:      416,
						Uid:     uid + 3,
						Biz:     "baguwen",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题3",
						Content: "Go问题3",
						Ctime:   1234,
						Utime:   2345,
					},
					{
						Id:      417,
						Uid:     uid + 4,
						Biz:     "baguwen",
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
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				expected := []dao.Question{
					{
						Uid:     uid + 2,
						Biz:     "baguwen",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题2",
						Content: "Go问题2",
					},
					{
						Uid:     uid + 3,
						Biz:     "baguwen",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Title:   "Go问题3",
						Content: "Go问题3",
					},
					{
						Uid:     uid + 4,
						Biz:     "baguwen",
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
			req: web.UpdateQuestions{
				QSID: 219,
				QIDs: []int64{415, 416, 417},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{},
		},
		{
			name: "题集不存在",
			before: func(t *testing.T) {
			},
			after: func(t *testing.T) {
			},
			req: web.UpdateQuestions{
				QSID: 10000,
				QIDs: []int64{},
			},
			wantCode: 500,
			wantResp: test.Result[int64]{Code: 502001, Msg: "系统错误"},
		},
	}

	for _, tc := range testCases {
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

func (s *AdminSetHandlerTestSuite) TestQuestionSet_Detail() {

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
					Biz:         "roadmap",
					BizId:       2,
					Description: "Go题集",
					Utime:       now,
				})
				require.NoError(t, err)
				require.Equal(t, int64(321), id)
			},
			after: func(t *testing.T) {
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
					Biz:         "roadmap",
					BizId:       2,
					Utime:       now,
				},
			},
		},
		{
			name: "非空题集",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				// 创建一个空题集
				id, err := s.questionSetDAO.Create(ctx, dao.QuestionSet{
					Id:          322,
					Uid:         uid,
					Title:       "Go",
					Description: "Go题集",
					Biz:         "roadmap",
					BizId:       2,
					Utime:       now,
				})
				require.NoError(t, err)
				require.Equal(t, int64(322), id)

				// 添加问题
				questions := []dao.Question{
					{
						Id:      614,
						Uid:     uid + 1,
						Biz:     "project",
						BizId:   1,
						Title:   "Go问题1",
						Content: "Go问题1",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:      615,
						Uid:     uid + 2,
						Biz:     "project",
						BizId:   1,
						Title:   "Go问题2",
						Content: "Go问题2",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:      616,
						Uid:     uid + 3,
						Biz:     "project",
						BizId:   1,
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
					Biz:         "roadmap",
					BizId:       2,
					Title:       "Go",
					Description: "Go题集",
					Questions: []web.Question{
						{
							Id:      614,
							Biz:     "project",
							BizId:   1,
							Title:   "Go问题1",
							Content: "Go问题1",
							Utime:   now,
						},
						{
							Id:      615,
							Biz:     "project",
							BizId:   1,
							Title:   "Go问题2",
							Content: "Go问题2",
							Utime:   now,
						},
						{
							Id:      616,
							Biz:     "project",
							BizId:   1,
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

func (s *AdminSetHandlerTestSuite) TestQuestionSetEvent() {
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
	saveReq := web.QuestionSet{
		Title:       "questionSet1",
		Biz:         "project",
		BizId:       1,
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
	syncReq := &web.UpdateQuestions{
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
			Biz:         "project",
			BizId:       1,
			Description: "question_description1",
			Questions:   []int64{},
		},
		{
			Uid:         uid,
			Title:       "questionSet1",
			Biz:         "project",
			BizId:       1,
			Description: "question_description1",
			Questions:   []int64{1, 2},
		},
	}, ans)
}

func (s *AdminSetHandlerTestSuite) TestGetQuestionSets() {
	var now int64 = 123
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T, sets []domain.QuestionSet)
		req    []int64
	}{
		{
			name: "非空题集",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				// 创建两个空题集
				id, err := s.questionSetDAO.Create(ctx, dao.QuestionSet{
					Id:          322,
					Uid:         uid,
					Title:       "Go",
					Description: "Go题集",
					Biz:         "roadmap",
					BizId:       2,
					Utime:       now,
				})
				require.NoError(t, err)
				require.Equal(t, int64(322), id)
				id, err = s.questionSetDAO.Create(ctx, dao.QuestionSet{
					Id:          323,
					Uid:         uid,
					Title:       "mysql",
					Description: "mysql题集",
					Biz:         "roadmap",
					BizId:       3,
					Utime:       now,
				})
				require.NoError(t, err)
				require.Equal(t, int64(323), id)

				// 为322添加问题
				questions := []dao.Question{
					{
						Id:      614,
						Uid:     uid + 1,
						Biz:     "project",
						BizId:   1,
						Title:   "Go问题1",
						Content: "Go问题1",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:      615,
						Uid:     uid + 2,
						Biz:     "project",
						BizId:   1,
						Title:   "Go问题2",
						Content: "Go问题2",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:      616,
						Uid:     uid + 3,
						Biz:     "project",
						BizId:   1,
						Title:   "Go问题3",
						Content: "Go问题3",
						Ctime:   now,
						Utime:   now,
					},
				}
				err = s.db.WithContext(ctx).Create(&questions).Error
				require.NoError(t, err)
				qids := []int64{614, 615, 616}
				require.NoError(t, s.questionSetDAO.UpdateQuestionsByID(ctx, 322, qids))
				// 为333添加题目
				questions = []dao.Question{
					{
						Id:      618,
						Uid:     uid + 1,
						Biz:     "project",
						BizId:   1,
						Title:   "Mysql问题1",
						Content: "Mysql问题1",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:      619,
						Uid:     uid + 2,
						Biz:     "project",
						BizId:   1,
						Title:   "Mysql问题2",
						Content: "Mysql问题2",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:      620,
						Uid:     uid + 3,
						Biz:     "project",
						BizId:   1,
						Title:   "Mysql问题4",
						Content: "Mysql问题4",
						Ctime:   now,
						Utime:   now,
					},
				}
				err = s.db.WithContext(ctx).Create(&questions).Error
				require.NoError(t, err)
				qids = []int64{618, 619, 620}
				require.NoError(t, s.questionSetDAO.UpdateQuestionsByID(ctx, 323, qids))

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
				qs, err := s.questionSetDAO.GetQuestionsByID(ctx, 322)
				require.NoError(t, err)
				require.Equal(t, 3, len(qs))
				qs, err = s.questionSetDAO.GetQuestionsByID(ctx, 323)
				require.NoError(t, err)
				require.Equal(t, 3, len(qs))
			},
			after: func(t *testing.T, sets []domain.QuestionSet) {
				assert.Equal(t, []domain.QuestionSet{
					{
						Id:    322,
						Title: "Go",
						Questions: []domain.Question{
							{
								Id: 614,

								Title: "Go问题1",
							},
							{
								Id:    615,
								Title: "Go问题2",
							},
							{
								Id:    616,
								Title: "Go问题3",
							},
						},
					},
					{
						Id:    323,
						Title: "mysql",
						Questions: []domain.Question{
							{
								Id:    618,
								Title: "Mysql问题1",
							},
							{
								Id:    619,
								Title: "Mysql问题2",
							},
							{
								Id:    620,
								Title: "Mysql问题4",
							},
						},
					},
				}, sets)

			},
			req: []int64{322, 323},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			sets, err := s.setSvc.GetByIDsWithQuestion(context.Background(), tc.req)
			require.NoError(t, err)
			tc.after(t, sets)
		})
	}
}

func TestAdminSetHandler(t *testing.T) {
	suite.Run(t, new(AdminSetHandlerTestSuite))
}
