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
	"strconv"
	"testing"
	"time"

	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"

	eveMocks "github.com/ecodeclub/webook/internal/question/internal/event/mocks"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/webook/internal/question/internal/domain"

	"github.com/ecodeclub/webook/internal/pkg/middleware"

	"github.com/ecodeclub/ecache"
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

const uid = 123

type HandlerTestSuite struct {
	suite.Suite
	server         *egin.Component
	db             *egorm.Component
	rdb            ecache.Cache
	dao            dao.QuestionDAO
	questionSetDAO dao.QuestionSetDAO
	ctrl           *gomock.Controller
	producer       *eveMocks.MockSyncEventProducer
}

func (s *HandlerTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `publish_answer_elements`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `publish_questions`").Error
	require.NoError(s.T(), err)

	err = s.db.Exec("TRUNCATE TABLE `question_sets`").Error
	require.NoError(s.T(), err)

	err = s.db.Exec("TRUNCATE TABLE `question_set_questions`").Error
	require.NoError(s.T(), err)

	err = s.db.Exec("TRUNCATE TABLE `question_results`").Error
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())
	s.producer = eveMocks.NewMockSyncEventProducer(s.ctrl)

	intrSvc := intrmocks.NewMockService(s.ctrl)
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

	module.Hdl.PublicRoutes(server.Engine)
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
	module.Hdl.MemberRoutes(server.Engine)

	s.server = server
	s.db = testioc.InitDB()
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewGORMQuestionDAO(s.db)
	s.questionSetDAO = dao.NewGORMQuestionSetDAO(s.db)
	s.rdb = testioc.InitCache()
}

func (s *HandlerTestSuite) TestPubList() {
	// 插入一百条
	data := make([]dao.PublishQuestion, 0, 100)
	for idx := 0; idx < 100; idx++ {
		data = append(data, dao.PublishQuestion{
			Uid:     uid,
			Status:  domain.UnPublishedStatus.ToUint8(),
			Title:   fmt.Sprintf("这是标题 %d", idx),
			Content: fmt.Sprintf("这是解析 %d", idx),
			Utime:   123,
		})
	}
	err := s.db.Create(&data).Error
	require.NoError(s.T(), err)
	testCases := []struct {
		name string
		req  web.Page

		wantCode int
		wantResp test.Result[[]web.Question]
	}{
		{
			name: "获取成功",
			req: web.Page{
				Limit:  2,
				Offset: 0,
			},
			wantCode: 200,
			wantResp: test.Result[[]web.Question]{
				Data: []web.Question{
					{
						Id:      100,
						Title:   "这是标题 99",
						Content: "这是解析 99",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Utime:   123,
						Interactive: web.Interactive{
							ViewCnt:    101,
							LikeCnt:    102,
							CollectCnt: 103,
							Liked:      false,
							Collected:  true,
						},
					},
					{
						Id:      99,
						Title:   "这是标题 98",
						Content: "这是解析 98",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Utime:   123,
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
		{
			name: "获取部分",
			req: web.Page{
				Limit:  2,
				Offset: 99,
			},
			wantCode: 200,
			wantResp: test.Result[[]web.Question]{
				Data: []web.Question{
					{
						Id:      1,
						Title:   "这是标题 0",
						Content: "这是解析 0",
						Status:  domain.UnPublishedStatus.ToUint8(),
						Utime:   123,
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
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/question/pub/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[[]web.Question]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = s.rdb.Delete(ctx, "question:total")
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) TestPubDetail() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	// 插入一百条
	data := make([]dao.PublishQuestion, 0, 2)
	results := make([]dao.QuestionResult, 0, 2)
	for idx := 0; idx < 2; idx++ {
		data = append(data, dao.PublishQuestion{
			Id:      int64(idx + 1),
			Uid:     uid,
			Status:  domain.PublishedStatus.ToUint8(),
			Title:   fmt.Sprintf("这是标题 %d", idx),
			Content: fmt.Sprintf("这是解析 %d", idx),
		})

		results = append(results, dao.QuestionResult{
			Uid:    uid,
			Qid:    int64(idx + 1),
			Result: domain.ResultIntermediate.ToUint8(),
		})
	}
	err := s.db.WithContext(ctx).Create(&data).Error
	require.NoError(s.T(), err)
	// 插入对应的评分数据
	s.db.WithContext(ctx).Create(&results)
	testCases := []struct {
		name string

		req      web.Qid
		wantCode int
		wantResp test.Result[web.Question]
	}{
		{
			name: "查询到了数据",
			req: web.Qid{
				Qid: 2,
			},
			wantCode: 200,
			wantResp: test.Result[web.Question]{
				Data: web.Question{
					Id:      2,
					Title:   "这是标题 1",
					Status:  domain.PublishedStatus.ToUint8(),
					Content: "这是解析 1",
					Utime:   0,
					Interactive: web.Interactive{
						ViewCnt:    3,
						LikeCnt:    4,
						CollectCnt: 5,
						Liked:      false,
						Collected:  true,
					},
					ExamineResult: domain.ResultIntermediate.ToUint8(),
				},
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/question/pub/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Question]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
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

func (s *HandlerTestSuite) mockInteractive(biz string, id int64) interactive.Interactive {
	liked := id%2 == 1
	collected := id%2 == 0
	return interactive.Interactive{
		Biz:        biz,
		BizId:      id,
		ViewCnt:    int(id + 1),
		LikeCnt:    int(id + 2),
		CollectCnt: int(id + 3),
		Liked:      liked,
		Collected:  collected,
	}
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
