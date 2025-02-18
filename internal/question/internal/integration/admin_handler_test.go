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

	"github.com/ecodeclub/webook/internal/member"

	"github.com/ecodeclub/webook/internal/ai"

	"github.com/ecodeclub/webook/internal/permission"

	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"

	"github.com/ecodeclub/webook/internal/question/internal/event"
	eveMocks "github.com/ecodeclub/webook/internal/question/internal/event/mocks"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/webook/internal/question/internal/domain"

	"gorm.io/gorm"

	"github.com/ecodeclub/ekit/sqlx"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
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
)

type AdminHandlerTestSuite struct {
	BaseTestSuite
	server                *egin.Component
	rdb                   ecache.Cache
	dao                   dao.QuestionDAO
	producer              *eveMocks.MockSyncEventProducer
	knowledgeBaseProducer *eveMocks.MockKnowledgeBaseEventProducer
}

func (s *AdminHandlerTestSuite) SetupSuite() {
	s.BaseTestSuite.db = testioc.InitDB()

	ctrl := gomock.NewController(s.T())
	s.producer = eveMocks.NewMockSyncEventProducer(ctrl)
	s.knowledgeBaseProducer = eveMocks.NewMockKnowledgeBaseEventProducer(ctrl)
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

	module, err := startup.InitModule(s.producer,
		s.knowledgeBaseProducer, intrModule,
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
	module.AdminHdl.PrivateRoutes(server.Engine)

	s.server = server
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewGORMQuestionDAO(s.db)
	s.rdb = testioc.InitCache()
}

func (s *AdminHandlerTestSuite) TestSave() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)
		req    web.SaveReq

		wantCode int
		wantResp test.Result[int64]
	}{
		{
			//
			name: "全部新建",
			before: func(t *testing.T) {
				s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				q, eles, err := s.dao.GetByID(ctx, 1)
				require.NoError(t, err)
				s.assertQuestion(t, dao.Question{
					Uid:     uid,
					Title:   "面试题1",
					Content: "面试题内容",
					Biz:     "project",
					BizId:   1,
					Status:  domain.UnPublishedStatus.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
				}, q)
				assert.Equal(t, 4, len(eles))
				wantEles := []dao.AnswerElement{
					s.buildDAOAnswerEle(1, 0, dao.AnswerElementTypeAnalysis),
					s.buildDAOAnswerEle(1, 1, dao.AnswerElementTypeBasic),
					s.buildDAOAnswerEle(1, 2, dao.AnswerElementTypeIntermedia),
					s.buildDAOAnswerEle(1, 3, dao.AnswerElementTypeAdvanced),
				}
				for i := range eles {
					ele := &(eles[i])
					assert.True(t, ele.Id > 0)
					assert.True(t, ele.Ctime > 0)
					assert.True(t, ele.Utime > 0)
					ele.Id = 0
					ele.Ctime = 0
					ele.Utime = 0
				}
				assert.ElementsMatch(t, wantEles, eles)
			},
			req: web.SaveReq{
				Question: web.Question{
					Title:        "面试题1",
					Content:      "面试题内容",
					Biz:          "project",
					BizId:        1,
					Labels:       []string{"MySQL"},
					Analysis:     s.buildAnswerEle(0),
					Basic:        s.buildAnswerEle(1),
					Intermediate: s.buildAnswerEle(2),
					Advanced:     s.buildAnswerEle(3),
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		},
		{
			//
			name: "部分更新",
			before: func(t *testing.T) {
				s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Question{
					Id:    2,
					Uid:   uid,
					Title: "老的标题",
					Biz:   "project",
					BizId: 1,
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
					Content: "老的内容",
					Status:  domain.UnPublishedStatus.ToUint8(),
					Ctime:   123,
					Utime:   234,
				}).Error
				require.NoError(t, err)
				err = s.db.Create(&dao.AnswerElement{
					Id:        1,
					Qid:       2,
					Type:      dao.AnswerElementTypeAnalysis,
					Content:   "老的分析",
					Keywords:  "老的 keyword",
					Shorthand: "老的速记",
					Highlight: "老的亮点",
					Guidance:  "老的引导点",
					Ctime:     123,
					Utime:     123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				q, eles, err := s.dao.GetByID(ctx, 2)
				require.NoError(t, err)
				s.assertQuestion(t, dao.Question{
					Uid:     uid,
					Status:  domain.UnPublishedStatus.ToUint8(),
					Title:   "面试题1",
					Biz:     "roadmap",
					BizId:   2,
					Content: "新的内容",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"sqlite"},
					},
				}, q)
				assert.Equal(t, 4, len(eles))
				analysis := eles[0]
				s.assertAnswerElement(t, dao.AnswerElement{
					Content:   "新的分析",
					Type:      dao.AnswerElementTypeAnalysis,
					Qid:       2,
					Keywords:  "新的 keyword",
					Shorthand: "新的速记",
					Highlight: "新的亮点",
					Guidance:  "新的引导点",
				}, analysis)
			},
			req: func() web.SaveReq {
				analysis := web.AnswerElement{
					Id:        1,
					Content:   "新的分析",
					Keywords:  "新的 keyword",
					Shorthand: "新的速记",
					Highlight: "新的亮点",
					Guidance:  "新的引导点",
				}
				return web.SaveReq{
					Question: web.Question{
						Id:           2,
						Title:        "面试题1",
						Content:      "新的内容",
						Biz:          "roadmap",
						BizId:        2,
						Labels:       []string{"sqlite"},
						Analysis:     analysis,
						Basic:        s.buildAnswerEle(1),
						Intermediate: s.buildAnswerEle(2),
						Advanced:     s.buildAnswerEle(3),
					},
				}
			}(),
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
			req, err := http.NewRequest(http.MethodPost,
				"/question/save", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
			// 清理掉 123 的数据
			err = s.db.Exec("TRUNCATE table `questions`").Error
			require.NoError(t, err)
			err = s.db.Exec("TRUNCATE table `answer_elements`").Error
			require.NoError(t, err)
		})
	}
}

func (s *AdminHandlerTestSuite) TestSync() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)
		req    web.SaveReq

		wantCode int
		wantResp test.Result[int64]
	}{
		{
			//
			name: "全部新建",
			before: func(t *testing.T) {
				s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)
				s.knowledgeBaseProducer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				q, eles, err := s.dao.GetPubByID(ctx, 1)
				require.NoError(t, err)
				s.assertQuestion(t, dao.Question{
					Uid:    uid,
					Title:  "面试题1",
					Biz:    "project",
					BizId:  1,
					Status: domain.PublishedStatus.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
					Content: "面试题内容",
				}, dao.Question(q))
				assert.Equal(t, 4, len(eles))
				s.cacheAssertQuestion(domain.Question{
					Id:      1,
					Uid:     uid,
					Title:   "面试题1",
					Biz:     "project",
					BizId:   1,
					Status:  domain.PublishedStatus,
					Labels:  []string{"MySQL"},
					Content: "面试题内容",
					Answer: domain.Answer{
						Analysis:     s.buildDomainAnswerEle(0, 1),
						Basic:        s.buildDomainAnswerEle(1, 2),
						Intermediate: s.buildDomainAnswerEle(2, 3),
						Advanced:     s.buildDomainAnswerEle(3, 4),
					},
				})
				s.cacheAssertQuestionList("project", []domain.Question{
					{
						Id:      1,
						Uid:     uid,
						Title:   "面试题1",
						Biz:     "project",
						BizId:   1,
						Status:  domain.PublishedStatus,
						Labels:  []string{"MySQL"},
						Content: "面试题内容",
					},
				})

			},
			req: web.SaveReq{
				Question: web.Question{
					Title:        "面试题1",
					Content:      "面试题内容",
					Biz:          "project",
					BizId:        1,
					Labels:       []string{"MySQL"},
					Analysis:     s.buildAnswerEle(0),
					Basic:        s.buildAnswerEle(1),
					Intermediate: s.buildAnswerEle(2),
					Advanced:     s.buildAnswerEle(3),
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		},
		{
			//
			name: "部分更新",
			before: func(t *testing.T) {
				s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)
				s.knowledgeBaseProducer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Question{
					Id:      2,
					Uid:     uid,
					Title:   "老的标题",
					Content: "老的内容",
					Biz:     "project",
					BizId:   1,
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
					Ctime: 123,
					Utime: 234,
				}).Error
				require.NoError(t, err)
				err = s.db.Create(&dao.AnswerElement{
					Id:        1,
					Qid:       2,
					Type:      dao.AnswerElementTypeAnalysis,
					Content:   "老的分析",
					Keywords:  "老的 keyword",
					Shorthand: "老的速记",
					Highlight: "老的亮点",
					Guidance:  "老的引导点",
					Ctime:     123,
					Utime:     123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				q, eles, err := s.dao.GetByID(ctx, 2)
				require.NoError(t, err)
				s.assertQuestion(t, dao.Question{
					Uid:    uid,
					Status: domain.PublishedStatus.ToUint8(),
					Biz:    "roadmap",
					BizId:  2,
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"sqlite"},
					},
					Title:   "面试题1",
					Content: "新的内容",
				}, q)
				assert.Equal(t, 4, len(eles))
				analysis := eles[0]
				s.assertAnswerElement(t, dao.AnswerElement{
					Content:   "新的分析",
					Type:      dao.AnswerElementTypeAnalysis,
					Qid:       2,
					Keywords:  "新的 keyword",
					Shorthand: "新的速记",
					Highlight: "新的亮点",
					Guidance:  "新的引导点",
				}, analysis)

				pq, pEles, err := s.dao.GetPubByID(ctx, 2)

				s.assertQuestion(t, dao.Question{
					Uid:    uid,
					Status: domain.PublishedStatus.ToUint8(),
					Title:  "面试题1",
					Biz:    "roadmap",
					BizId:  2,
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"sqlite"},
					},
					Content: "新的内容",
				}, dao.Question(pq))
				assert.Equal(t, 4, len(pEles))
				pAnalysis := pEles[0]
				s.assertAnswerElement(t, dao.AnswerElement{
					Content:   "新的分析",
					Type:      dao.AnswerElementTypeAnalysis,
					Qid:       2,
					Keywords:  "新的 keyword",
					Shorthand: "新的速记",
					Highlight: "新的亮点",
					Guidance:  "新的引导点",
				}, dao.AnswerElement(pAnalysis))

				s.cacheAssertQuestion(domain.Question{
					Id:      2,
					Uid:     uid,
					Status:  domain.PublishedStatus,
					Biz:     "roadmap",
					BizId:   2,
					Labels:  []string{"sqlite"},
					Title:   "面试题1",
					Content: "新的内容",
					Answer: domain.Answer{
						Analysis: domain.AnswerElement{
							Id:        1,
							Content:   "新的分析",
							Keywords:  "新的 keyword",
							Shorthand: "新的速记",
							Highlight: "新的亮点",
							Guidance:  "新的引导点",
						},
						Basic:        s.buildDomainAnswerEle(1, 2),
						Intermediate: s.buildDomainAnswerEle(2, 3),
						Advanced:     s.buildDomainAnswerEle(3, 4),
					},
				})
			},
			req: func() web.SaveReq {
				analysis := web.AnswerElement{
					Id:        1,
					Content:   "新的分析",
					Keywords:  "新的 keyword",
					Shorthand: "新的速记",
					Highlight: "新的亮点",
					Guidance:  "新的引导点",
				}
				return web.SaveReq{
					Question: web.Question{
						Id:           2,
						Title:        "面试题1",
						Content:      "新的内容",
						Biz:          "roadmap",
						BizId:        2,
						Labels:       []string{"sqlite"},
						Analysis:     analysis,
						Basic:        s.buildAnswerEle(1),
						Intermediate: s.buildAnswerEle(2),
						Advanced:     s.buildAnswerEle(3),
					},
				}
			}(),
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 2,
			},
		},
		{
			name: "更新缓存",
			before: func(t *testing.T) {
				s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)
				s.knowledgeBaseProducer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)
				ques := []domain.Question{
					{
						Id:      8,
						Uid:     uid,
						Title:   "面试题1",
						Biz:     "case",
						BizId:   1,
						Status:  domain.PublishedStatus,
						Labels:  []string{"MySQL"},
						Content: "面试题内容",
					},
					{
						Id:      9,
						Uid:     uid,
						Title:   "面试题1",
						Biz:     "case",
						BizId:   1,
						Status:  domain.PublishedStatus,
						Labels:  []string{"MySQL"},
						Content: "面试题内容",
					},
				}
				quesByte, _ := json.Marshal(ques)
				err := s.rdb.Set(context.Background(), "question:list:case", string(quesByte), 24*time.Hour)
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				s.cacheAssertQuestionList("case", []domain.Question{
					{
						Id:      1,
						Uid:     uid,
						Title:   "面试题1",
						Biz:     "case",
						BizId:   1,
						Status:  domain.PublishedStatus,
						Labels:  []string{"MySQL"},
						Content: "面试题内容",
					},
				})
			},
			req: web.SaveReq{
				Question: web.Question{
					Title:        "面试题1",
					Content:      "面试题内容",
					Biz:          "case",
					BizId:        1,
					Labels:       []string{"MySQL"},
					Analysis:     s.buildAnswerEle(0),
					Basic:        s.buildAnswerEle(1),
					Intermediate: s.buildAnswerEle(2),
					Advanced:     s.buildAnswerEle(3),
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/question/publish", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
			// 清理掉 123 的数据
			err = s.db.Exec("TRUNCATE table `questions`").Error
			require.NoError(t, err)
			err = s.db.Exec("TRUNCATE table `answer_elements`").Error
			require.NoError(t, err)
		})
	}
}

func (s *AdminHandlerTestSuite) TestDelete() {
	testCases := []struct {
		name string

		qid    int64
		before func(t *testing.T)
		after  func(t *testing.T)

		wantCode int
		wantResp test.Result[any]
	}{
		{
			name: "删除成功",
			qid:  123,
			before: func(t *testing.T) {
				originQs := []domain.Question{
					{
						Id:  234,
						Biz: "xx",
					},
					{
						Id:  123,
						Biz: "xx",
					},
				}
				qsByte, err := json.Marshal(originQs)
				require.NoError(t, err)
				s.rdb.Set(context.Background(), "question:list:xx", string(qsByte), 24*time.Hour)

				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				var qid int64 = 123
				// prepare data
				s.db.Model(&dao.Question{}).Create(&dao.Question{
					Id:  qid,
					Biz: "xx",
				})
				_, err = s.dao.Sync(ctx, dao.Question{
					Id:  qid,
					Biz: "xx",
				}, []dao.AnswerElement{
					{
						Qid: qid,
					},
				})
				require.NoError(t, err)

				s.db.Model(&dao.Question{}).Create(&dao.Question{
					Id:    234,
					Biz:   "xx",
					Utime: 1739779178000,
				})
				_, err = s.dao.Sync(ctx, dao.Question{
					Id:    234,
					Biz:   "xx",
					Utime: 1739779178000,
				}, []dao.AnswerElement{
					{
						Qid: 234,
					},
				})
				require.NoError(t, err)
				err = s.db.Create(&dao.QuestionSetQuestion{
					QID: qid,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				var qid int64 = 123
				_, _, err := s.dao.GetPubByID(ctx, qid)
				assert.Equal(t, err, gorm.ErrRecordNotFound)
				_, _, err = s.dao.GetByID(ctx, qid)
				assert.Equal(t, err, gorm.ErrRecordNotFound)
				var res []dao.QuestionSetQuestion
				err = s.db.Model(&dao.QuestionSetQuestion{}).Where("qid = ?", qid).Find(&res).Error
				assert.NoError(t, err)
				assert.Equal(t, 0, len(res))
				s.cacheAssertQuestionList("xx", []domain.Question{
					{
						Id:  234,
						Biz: "xx",
					},
				})
			},
			wantCode: 200,
			wantResp: test.Result[any]{},
		},
		{
			name: "删除不存在的 Question",
			qid:  124,
			before: func(t *testing.T) {
			},
			after: func(t *testing.T) {
			},
			wantCode: 200,
			wantResp: test.Result[any]{},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/question/delete", iox.NewJSONReader(web.Qid{Qid: tc.qid}))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
		})
	}

}

// assertAnswerElement 不包括 Id
func (s *AdminHandlerTestSuite) assertAnswerElement(
	t *testing.T,
	expect dao.AnswerElement,
	ele dao.AnswerElement) {
	assert.True(t, ele.Id > 0)
	ele.Id = 0
	assert.True(t, ele.Ctime > 0)
	ele.Ctime = 0
	assert.True(t, ele.Utime > 0)
	ele.Utime = 0
	assert.Equal(t, expect, ele)
}

func (s *AdminHandlerTestSuite) TestQuestionEvent() {
	t := s.T()
	ans := make([]event.Question, 0, 16)
	mu := sync.RWMutex{}
	s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, questionEvent event.QuestionEvent) error {
		var eve event.Question
		err := json.Unmarshal([]byte(questionEvent.Data), &eve)
		if err != nil {
			return err
		}
		mu.Lock()
		ans = append(ans, eve)
		mu.Unlock()
		return nil
	}).Times(2)
	s.knowledgeBaseProducer.EXPECT().Produce(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, knowledgeBaseEvent event.KnowledgeBaseEvent) error {
		assert.Equal(t, "question", knowledgeBaseEvent.Biz)
		var que domain.Question
		json.Unmarshal(knowledgeBaseEvent.Data, &que)
		que.Answer.Basic.Id = 0
		que.Answer.Advanced.Id = 0
		que.Answer.Analysis.Id = 0
		que.Answer.Intermediate.Id = 0
		que.Utime = time.UnixMilli(123)
		assert.Equal(t, que.Id, knowledgeBaseEvent.BizID)
		assert.Equal(t, fmt.Sprintf("question_%d", que.Id), knowledgeBaseEvent.Name)
		assert.Equal(t, ai.RepositoryBaseTypeRetrieval, knowledgeBaseEvent.Type)
		assert.Equal(t, s.getWantQuestion(que.Id), que)
		return nil
	})
	// 保存
	saveReq := web.SaveReq{
		Question: web.Question{
			Title:        "面试题1",
			Content:      "新的内容",
			Biz:          "project",
			BizId:        1,
			Analysis:     s.buildAnswerEle(1),
			Basic:        s.buildAnswerEle(2),
			Intermediate: s.buildAnswerEle(3),
			Advanced:     s.buildAnswerEle(4),
		},
	}
	req, err := http.NewRequest(http.MethodPost,
		"/question/save", iox.NewJSONReader(saveReq))
	req.Header.Set("content-type", "application/json")
	require.NoError(t, err)
	recorder := test.NewJSONResponseRecorder[int64]()
	s.server.ServeHTTP(recorder, req)

	require.Equal(t, 200, recorder.Code)

	// 发布
	syncReq := &web.SaveReq{
		Question: web.Question{
			Title:        "面试题2",
			Content:      "面试题内容",
			Biz:          "project",
			BizId:        2,
			Analysis:     s.buildAnswerEle(0),
			Basic:        s.buildAnswerEle(1),
			Intermediate: s.buildAnswerEle(2),
			Advanced:     s.buildAnswerEle(3),
		},
	}
	req2, err := http.NewRequest(http.MethodPost,
		"/question/publish", iox.NewJSONReader(syncReq))
	req2.Header.Set("content-type", "application/json")
	require.NoError(t, err)
	recorder = test.NewJSONResponseRecorder[int64]()
	s.server.ServeHTTP(recorder, req2)
	require.Equal(t, 200, recorder.Code)
	time.Sleep(1 * time.Second)
	for idx := range ans {
		ans[idx].ID = 0
		ans[idx].Utime = 0
		ans[idx].Answer = event.Answer{
			Analysis:     s.removeId(ans[idx].Answer.Analysis),
			Basic:        s.removeId(ans[idx].Answer.Basic),
			Intermediate: s.removeId(ans[idx].Answer.Intermediate),
			Advanced:     s.removeId(ans[idx].Answer.Advanced),
		}
	}
	assert.Equal(t, []event.Question{
		{
			Title:   "面试题1",
			Content: "新的内容",
			UID:     uid,
			Status:  1,
			Biz:     "project",
			BizId:   1,
			Answer: event.Answer{
				Analysis:     s.buildEventEle(1),
				Basic:        s.buildEventEle(2),
				Intermediate: s.buildEventEle(3),
				Advanced:     s.buildEventEle(4),
			},
		},
		{
			Title:   "面试题2",
			UID:     uid,
			Content: "面试题内容",
			Status:  2,
			Biz:     "project",
			BizId:   2,
			Answer: event.Answer{
				Analysis:     s.buildEventEle(0),
				Basic:        s.buildEventEle(1),
				Intermediate: s.buildEventEle(2),
				Advanced:     s.buildEventEle(3),
			},
		},
	}, ans)

}

func (s *AdminHandlerTestSuite) removeId(ele event.AnswerElement) event.AnswerElement {
	require.True(s.T(), ele.ID != 0)
	ele.ID = 0
	return ele
}

func (s *AdminHandlerTestSuite) buildEventEle(idx int64) event.AnswerElement {
	return event.AnswerElement{
		Content:   fmt.Sprintf("这是解析 %d", idx),
		Keywords:  fmt.Sprintf("关键字 %d", idx),
		Shorthand: fmt.Sprintf("快速记忆法 %d", idx),
		Highlight: fmt.Sprintf("亮点 %d", idx),
		Guidance:  fmt.Sprintf("引导点 %d", idx),
	}
}

func TestAdminHandler(t *testing.T) {
	suite.Run(t, new(AdminHandlerTestSuite))
}

func (s *AdminHandlerTestSuite) getWantQuestion(id int64) domain.Question {
	que := domain.Question{
		Id:      id,
		Uid:     uid,
		Biz:     "project",
		BizId:   id,
		Title:   fmt.Sprintf("面试题%d", id),
		Content: "面试题内容",
		Status:  domain.PublishedStatus,
		Answer: domain.Answer{
			Analysis:     s.getAnswerElement(0),
			Basic:        s.getAnswerElement(1),
			Intermediate: s.getAnswerElement(2),
			Advanced:     s.getAnswerElement(3),
		},
		Utime: time.UnixMilli(123),
	}
	return que
}

func (s *AdminHandlerTestSuite) getAnswerElement(idx int64) domain.AnswerElement {
	return domain.AnswerElement{
		Content:   fmt.Sprintf("这是解析 %d", idx),
		Keywords:  fmt.Sprintf("关键字 %d", idx),
		Shorthand: fmt.Sprintf("快速记忆法 %d", idx),
		Highlight: fmt.Sprintf("亮点 %d", idx),
		Guidance:  fmt.Sprintf("引导点 %d", idx),
	}
}

// 校验缓存中的数据
func (s *AdminHandlerTestSuite) cacheAssertQuestion(q domain.Question) {
	t := s.T()
	key := fmt.Sprintf("question:publish:%d", q.Id)
	val := s.rdb.Get(context.Background(), key)
	require.NoError(t, val.Err)

	var actual domain.Question
	err := json.Unmarshal([]byte(val.Val.(string)), &actual)
	require.NoError(t, err)

	// 处理时间字段
	require.True(t, actual.Utime.Unix() > 0)
	q.Utime = actual.Utime

	// 清理缓存
	require.True(t, actual.Answer.Basic.Id > 0)
	require.True(t, actual.Answer.Advanced.Id > 0)
	require.True(t, actual.Answer.Intermediate.Id > 0)
	require.True(t, actual.Answer.Analysis.Id > 0)
	actual.Answer.Basic.Id = 0
	actual.Answer.Advanced.Id = 0
	actual.Answer.Intermediate.Id = 0
	actual.Answer.Analysis.Id = 0
	q.Answer.Basic.Id = 0
	q.Answer.Advanced.Id = 0
	q.Answer.Intermediate.Id = 0
	q.Answer.Analysis.Id = 0
	_, err = s.rdb.Delete(context.Background(), key)
	require.NoError(t, err)
	assert.Equal(t, q, actual)
}

func (s *AdminHandlerTestSuite) cacheAssertQuestionList(biz string, questions []domain.Question) {
	key := fmt.Sprintf("question:list:%s", biz)
	val := s.rdb.Get(context.Background(), key)
	require.NoError(s.T(), val.Err)

	var qs []domain.Question
	err := json.Unmarshal([]byte(val.Val.(string)), &qs)
	require.NoError(s.T(), err)
	require.Equal(s.T(), len(questions), len(qs))
	for idx, q := range qs {
		require.True(s.T(), q.Utime.UnixMilli() > 0)
		require.True(s.T(), q.Id > 0)
		//require.True(s.T(), q.Answer.Analysis.Id > 0)
		//require.True(s.T(), q.Answer.Basic.Id > 0)
		//require.True(s.T(), q.Answer.Advanced.Id > 0)
		//require.True(s.T(), q.Answer.Intermediate.Id > 0)
		qs[idx].Id = questions[idx].Id
		qs[idx].Utime = questions[idx].Utime

	}
	assert.Equal(s.T(), questions, qs)
	_, err = s.rdb.Delete(context.Background(), key)
	require.NoError(s.T(), err)
}
