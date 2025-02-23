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
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ecodeclub/ekit/sqlx"

	"github.com/ecodeclub/webook/internal/member"

	"github.com/ecodeclub/webook/internal/ai"

	"github.com/ecodeclub/webook/internal/permission"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
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
	intrSvc.EXPECT().GetByIds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context,
		biz string, uid int64, ids []int64) (map[int64]interactive.Interactive, error) {
		res := make(map[int64]interactive.Interactive, len(ids))
		for _, id := range ids {
			intr := s.mockInteractive(biz, id)
			res[id] = intr
		}
		return res, nil
	}).AnyTimes()

	module, err := startup.InitModule(s.producer, nil, intrModule, &permission.Module{}, &ai.Module{},
		session.DefaultProvider(),
		&member.Module{})
	require.NoError(s.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		notlogin := ctx.GetHeader("not_login") == "1"
		nuid := uid
		data := map[string]string{
			"creator": "true",
		}
		if notlogin {
			return
		}
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid:  int64(nuid),
			Data: data,
		}))
	})
	module.QsHdl.PublicRoutes(server.Engine)

	s.server = server
	s.db = testioc.InitDB()
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewGORMQuestionDAO(s.db)
	s.questionSetDAO = dao.NewGORMQuestionSetDAO(s.db)
	s.rdb = testioc.InitCache()
}

func (s *SetHandlerTestSuite) TestQuestionSetDetailByBiz() {
	var now int64 = 123

	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)
		req    web.BizReq

		wantCode int
		wantResp test.Result[web.QuestionSet]
	}{
		{
			name: "空题集",
			before: func(t *testing.T) {
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
			req: web.BizReq{
				Biz:   "roadmap",
				BizId: 2,
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
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				id, err := s.questionSetDAO.Create(ctx, dao.QuestionSet{
					Id:          322,
					Uid:         uid,
					Title:       "Go",
					Description: "Go题集",
					Biz:         "roadmap",
					BizId:       3,
					Utime:       now,
				})
				require.NoError(t, err)
				require.Equal(t, int64(322), id)

				// 添加问题
				questions := []dao.Question{
					{
						Id:    614,
						Uid:   uid + 1,
						Biz:   "project",
						BizId: 1,
						Labels: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val:   []string{"MySQL"},
						},
						Title:   "Go问题1",
						Content: "Go问题1",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:    615,
						Uid:   uid + 2,
						Biz:   "project",
						BizId: 1,
						Labels: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val:   []string{"Redis"},
						},
						Title:   "Go问题2",
						Content: "Go问题2",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:    616,
						Uid:   uid + 3,
						Biz:   "project",
						BizId: 1,
						Labels: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val:   []string{"mongo"},
						},
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
			req: web.BizReq{
				Biz:   "roadmap",
				BizId: 3,
			},
			wantCode: 200,
			wantResp: test.Result[web.QuestionSet]{
				Data: web.QuestionSet{
					Id:          322,
					Biz:         "roadmap",
					BizId:       3,
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
							Id:    614,
							Biz:   "project",
							BizId: 1,
							Labels: []string{
								"MySQL",
							},
							Title:         "Go问题1",
							Content:       "Go问题1",
							ExamineResult: domain.ResultAdvanced.ToUint8(),
							Utime:         now,
							Interactive: web.Interactive{
								ViewCnt:    615,
								LikeCnt:    616,
								CollectCnt: 617,
								Liked:      false,
								Collected:  true,
							},
						},
						{
							Id:    615,
							Biz:   "project",
							BizId: 1,
							Labels: []string{
								"Redis",
							},
							Title:   "Go问题2",
							Content: "Go问题2",
							Utime:   now,
							Interactive: web.Interactive{
								ViewCnt:    616,
								LikeCnt:    617,
								CollectCnt: 618,
								Liked:      true,
								Collected:  false,
							},
						},
						{
							Id:      616,
							Biz:     "project",
							BizId:   1,
							Title:   "Go问题3",
							Labels:  []string{"mongo"},
							Content: "Go问题3",
							Utime:   now,
							Interactive: web.Interactive{
								ViewCnt:    617,
								LikeCnt:    618,
								CollectCnt: 619,
								Liked:      false,
								Collected:  true,
							},
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
				"/question-sets/detail/biz", iox.NewJSONReader(tc.req))
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

func (s *SetHandlerTestSuite) TestQuestionSet_Detail() {

	var now int64 = 123

	testCases := []struct {
		name   string
		before func(t *testing.T, req *http.Request)
		after  func(t *testing.T)
		req    web.QuestionSetID

		wantCode int
		wantResp test.Result[web.QuestionSet]
	}{

		{
			name: "空题集",
			before: func(t *testing.T, req *http.Request) {
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
			before: func(t *testing.T, req *http.Request) {
				t.Helper()

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
				questions := []dao.PublishQuestion{
					{
						Id:    614,
						Uid:   uid + 1,
						Biz:   "project",
						BizId: 1,
						Labels: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val:   []string{"MySQL"},
						},
						Title:   "Go问题1",
						Content: "Go问题1",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:    615,
						Uid:   uid + 2,
						Biz:   "project",
						BizId: 1,
						Labels: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val:   []string{"MySQL"},
						},
						Title:   "Go问题2",
						Content: "Go问题2",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:    616,
						Uid:   uid + 3,
						Biz:   "project",
						BizId: 1,
						Labels: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val:   []string{"MySQL"},
						},
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
				qs, err := s.questionSetDAO.GetPubQuestionsByID(ctx, id)
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
					Interactive: web.Interactive{
						ViewCnt:    323,
						LikeCnt:    324,
						CollectCnt: 325,
						Liked:      false,
						Collected:  true,
					},
					Questions: []web.Question{
						{
							Id:      614,
							Biz:     "project",
							BizId:   1,
							Labels:  []string{"MySQL"},
							Title:   "Go问题1",
							Content: "Go问题1",
							Interactive: web.Interactive{
								ViewCnt:    615,
								LikeCnt:    616,
								CollectCnt: 617,
								Liked:      false,
								Collected:  true,
							},
							ExamineResult: domain.ResultAdvanced.ToUint8(),
							Utime:         now,
						},
						{
							Id:      615,
							Biz:     "project",
							BizId:   1,
							Labels:  []string{"MySQL"},
							Title:   "Go问题2",
							Content: "Go问题2",
							Interactive: web.Interactive{
								ViewCnt:    616,
								LikeCnt:    617,
								CollectCnt: 618,
								Liked:      true,
								Collected:  false,
							},
							Utime: now,
						},
						{
							Id:      616,
							Biz:     "project",
							BizId:   1,
							Title:   "Go问题3",
							Content: "Go问题3",
							Labels:  []string{"MySQL"},
							Interactive: web.Interactive{
								ViewCnt:    617,
								LikeCnt:    618,
								CollectCnt: 619,
								Liked:      false,
								Collected:  true,
							},
							Utime: now,
						},
					},
					Utime: now,
				},
			},
		},
		{
			name: "非空题集-未登录",
			before: func(t *testing.T, req *http.Request) {
				t.Helper()
				req.Header.Set("not_login", "1")
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				// 创建一个空题集
				id, err := s.questionSetDAO.Create(ctx, dao.QuestionSet{
					Id:          333,
					Uid:         uid,
					Title:       "Go",
					Description: "Go题集",
					Biz:         "roadmap",
					BizId:       2,
					Utime:       now,
				})
				require.NoError(t, err)
				require.Equal(t, int64(333), id)

				// 添加问题
				questions := []dao.PublishQuestion{
					{
						Id:    714,
						Uid:   uid + 1,
						Biz:   "project",
						BizId: 1,
						Labels: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val:   []string{"MySQL"},
						},
						Title:   "Go问题1",
						Content: "Go问题1",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:    715,
						Uid:   uid + 2,
						Biz:   "project",
						BizId: 1,
						Labels: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val:   []string{"MySQL"},
						},
						Title:   "Go问题2",
						Content: "Go问题2",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:    716,
						Uid:   uid + 3,
						Biz:   "project",
						BizId: 1,
						Labels: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val:   []string{"MySQL"},
						},
						Title:   "Go问题3",
						Content: "Go问题3",
						Ctime:   now,
						Utime:   now,
					},
				}
				err = s.db.WithContext(ctx).Create(&questions).Error
				require.NoError(t, err)
				qids := []int64{714, 715, 716}
				require.NoError(t, s.questionSetDAO.UpdateQuestionsByID(ctx, id, qids))

				// 添加用户答题记录，只需要添加一个就可以
				err = s.db.WithContext(ctx).Create(&dao.QuestionResult{
					Uid:    uid,
					Qid:    714,
					Result: domain.ResultAdvanced.ToUint8(),
					Ctime:  now,
					Utime:  now,
				}).Error
				require.NoError(t, err)

				// 题集中题目为1
				qs, err := s.questionSetDAO.GetPubQuestionsByID(ctx, id)
				require.NoError(t, err)
				require.Equal(t, len(qids), len(qs))
			},
			after: func(t *testing.T) {
			},
			req: web.QuestionSetID{
				QSID: 333,
			},
			wantCode: 200,
			wantResp: test.Result[web.QuestionSet]{
				Data: web.QuestionSet{
					Id:          333,
					Biz:         "roadmap",
					BizId:       2,
					Title:       "Go",
					Description: "Go题集",
					Interactive: web.Interactive{
						ViewCnt:    334,
						LikeCnt:    335,
						CollectCnt: 336,
						Liked:      true,
						Collected:  false,
					},
					Questions: []web.Question{
						{
							Id:      714,
							Biz:     "project",
							BizId:   1,
							Labels:  []string{"MySQL"},
							Title:   "Go问题1",
							Content: "Go问题1",
							Interactive: web.Interactive{
								ViewCnt:    715,
								LikeCnt:    716,
								CollectCnt: 717,
								Liked:      false,
								Collected:  true,
							},
							//ExamineResult: domain.ResultAdvanced.ToUint8(),
							Utime: now,
						},
						{
							Id:      715,
							Biz:     "project",
							BizId:   1,
							Labels:  []string{"MySQL"},
							Title:   "Go问题2",
							Content: "Go问题2",
							Interactive: web.Interactive{
								ViewCnt:    716,
								LikeCnt:    717,
								CollectCnt: 718,
								Liked:      true,
								Collected:  false,
							},
							Utime: now,
						},
						{
							Id:      716,
							Biz:     "project",
							BizId:   1,
							Labels:  []string{"MySQL"},
							Title:   "Go问题3",
							Content: "Go问题3",
							Interactive: web.Interactive{
								ViewCnt:    717,
								LikeCnt:    718,
								CollectCnt: 719,
								Liked:      false,
								Collected:  true,
							},
							Utime: now,
						},
					},
					Utime: now,
				},
			},
		},
		{
			name: "非空题集-忽略未发布",
			before: func(t *testing.T, req *http.Request) {
				t.Helper()

				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				// 创建一个空题集
				id, err := s.questionSetDAO.Create(ctx, dao.QuestionSet{
					Id:          433,
					Uid:         uid,
					Title:       "Go",
					Description: "Go题集",
					Biz:         "roadmap",
					BizId:       2,
					Utime:       now,
				})
				require.NoError(t, err)
				require.Equal(t, int64(433), id)

				// 添加问题
				questions := []dao.PublishQuestion{
					{
						Id:    814,
						Uid:   uid + 1,
						Biz:   "project",
						BizId: 1,
						Labels: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val:   []string{"MySQL"},
						},
						Title:   "Go问题1",
						Content: "Go问题1",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:    815,
						Uid:   uid + 2,
						Biz:   "project",
						BizId: 1,
						Labels: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val:   []string{"MySQL"},
						},
						Title:   "Go问题2",
						Content: "Go问题2",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:    816,
						Uid:   uid + 3,
						Biz:   "project",
						BizId: 1,
						Labels: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val:   []string{"MySQL"},
						},
						Title:   "Go问题3",
						Content: "Go问题3",
						Ctime:   now,
						Utime:   now,
					},
				}
				err = s.db.WithContext(ctx).Create(&questions).Error
				require.NoError(t, err)

				err = s.db.WithContext(ctx).Create(&dao.Question{
					Id:      817,
					Uid:     uid + 3,
					Biz:     "project",
					BizId:   1,
					Title:   "Go问题3",
					Content: "Go问题3",
					Status:  domain.UnPublishedStatus.ToUint8(),
					Ctime:   now,
					Utime:   now,
				}).Error
				require.NoError(t, err)
				// 817 还没发布
				qids := []int64{814, 815, 816, 817}
				require.NoError(t, s.questionSetDAO.UpdateQuestionsByID(ctx, id, qids))

				// 添加用户答题记录，只需要添加一个就可以
				err = s.db.WithContext(ctx).Create(&dao.QuestionResult{
					Uid:    uid,
					Qid:    814,
					Result: domain.ResultAdvanced.ToUint8(),
					Ctime:  now,
					Utime:  now,
				}).Error
				require.NoError(t, err)

				// 题集中题目为1
				qs, err := s.questionSetDAO.GetPubQuestionsByID(ctx, id)
				require.NoError(t, err)
				require.Equal(t, len(questions), len(qs))
			},
			after: func(t *testing.T) {
			},
			req: web.QuestionSetID{
				QSID: 433,
			},
			wantCode: 200,
			wantResp: test.Result[web.QuestionSet]{
				Data: web.QuestionSet{
					Id:          433,
					Biz:         "roadmap",
					BizId:       2,
					Title:       "Go",
					Description: "Go题集",
					Interactive: web.Interactive{
						ViewCnt:    434,
						LikeCnt:    435,
						CollectCnt: 436,
						Liked:      true,
						Collected:  false,
					},
					Questions: []web.Question{
						{
							Id:      814,
							Biz:     "project",
							BizId:   1,
							Labels:  []string{"MySQL"},
							Title:   "Go问题1",
							Content: "Go问题1",
							Interactive: web.Interactive{
								ViewCnt:    815,
								LikeCnt:    816,
								CollectCnt: 817,
								Liked:      false,
								Collected:  true,
							},
							ExamineResult: domain.ResultAdvanced.ToUint8(),
							Utime:         now,
						},
						{
							Id:      815,
							Biz:     "project",
							BizId:   1,
							Labels:  []string{"MySQL"},
							Title:   "Go问题2",
							Content: "Go问题2",
							Interactive: web.Interactive{
								ViewCnt:    816,
								LikeCnt:    817,
								CollectCnt: 818,
								Liked:      true,
								Collected:  false,
							},
							Utime: now,
						},
						{
							Id:      816,
							Biz:     "project",
							BizId:   1,
							Labels:  []string{"MySQL"},
							Title:   "Go问题3",
							Content: "Go问题3",
							Interactive: web.Interactive{
								ViewCnt:    817,
								LikeCnt:    818,
								CollectCnt: 819,
								Liked:      false,
								Collected:  true,
							},
							Utime: now,
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
			req, err := http.NewRequest(http.MethodPost,
				"/question-sets/detail", iox.NewJSONReader(tc.req))
			tc.before(t, req)
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
	// 这个接口是不会查询到这些数据的
	data = append(data, dao.QuestionSet{
		Uid:         200,
		Title:       fmt.Sprintf("题集标题 %d", 200),
		Description: fmt.Sprintf("题集简介 %d", 200),
		Biz:         "project",
		BizId:       200,
		Utime:       123,
	})
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
					Total: 100,
					QuestionSets: []web.QuestionSet{
						{
							Id:          100,
							Title:       "题集标题 99",
							Description: "题集简介 99",
							Biz:         "baguwen",
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
							Biz:         "baguwen",
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
					Total: 100,
					QuestionSets: []web.QuestionSet{
						{
							Id:          1,
							Title:       "题集标题 0",
							Description: "题集简介 0",
							Biz:         "baguwen",
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

func TestSetHandler(t *testing.T) {
	suite.Run(t, new(SetHandlerTestSuite))
}
