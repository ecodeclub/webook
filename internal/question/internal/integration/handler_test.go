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

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/question/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/question/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/question/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HandlerTestSuite struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
	dao    dao.QuestionDAO
}

func (s *HandlerTestSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `answer_elements`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("DROP TABLE `questions`").Error
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) SetupSuite() {
	handler, err := startup.InitHandler()
	require.NoError(s.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: 123,
		}))
	})
	handler.PrivateRoutes(server.Engine)
	s.server = server
	s.db = testioc.InitDB()
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewGORMQuestionDAO(s.db)
}

func (s *HandlerTestSuite) TestSave() {
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

			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				q, eles, err := s.dao.GetByID(ctx, 1)
				require.NoError(t, err)
				s.assertQuestion(t, dao.Question{
					Uid:     123,
					Title:   "面试题1",
					Content: "面试题内容",
				}, q)
				assert.Equal(t, 4, len(eles))
			},
			req: web.SaveReq{
				Question: web.Question{
					Title:   "面试题1",
					Content: "面试题内容",
					Answer: web.Answer{
						Analysis:     s.buildAnswerEle(0),
						Basic:        s.buildAnswerEle(1),
						Intermediate: s.buildAnswerEle(2),
						Advanced:     s.buildAnswerEle(3),
					},
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
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Question{
					Id:      2,
					Uid:     123,
					Title:   "老的标题",
					Content: "老的内容",
					Ctime:   123,
					Utime:   234,
				}).Error
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
					Uid:     123,
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
						Id:      2,
						Title:   "面试题1",
						Content: "新的内容",
						Answer: web.Answer{
							Analysis:     analysis,
							Basic:        s.buildAnswerEle(1),
							Intermediate: s.buildAnswerEle(2),
							Advanced:     s.buildAnswerEle(3),
						},
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
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/question", iox.NewJSONReader(tc.req))
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

func (s *HandlerTestSuite) buildAnswerEle(idx int64) web.AnswerElement {
	return web.AnswerElement{
		Content:   fmt.Sprintf("这是解析 %d", idx),
		Keywords:  fmt.Sprintf("关键字 %d", idx),
		Shorthand: fmt.Sprintf("快速记忆法 %d", idx),
		Highlight: fmt.Sprintf("亮点 %d", idx),
		Guidance:  fmt.Sprintf("引导点 %d", idx),
	}
}

// assertQuestion 不比较 id
func (s *HandlerTestSuite) assertQuestion(t *testing.T, expect dao.Question, q dao.Question) {
	assert.True(t, q.Id > 0)
	assert.True(t, q.Ctime > 0)
	assert.True(t, q.Utime > 0)
	q.Id = 0
	q.Ctime = 0
	q.Utime = 0
	assert.Equal(t, expect, q)
}

// assertAnswerElement 不包括 Id
func (s *HandlerTestSuite) assertAnswerElement(
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

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
