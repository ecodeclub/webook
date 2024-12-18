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

	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"
	"github.com/lithammer/shortuuid/v4"

	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/cases/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/cases/internal/web"
	"github.com/ecodeclub/webook/internal/pkg/middleware"
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

const uid = 2051

type HandlerTestSuite struct {
	suite.Suite
	server  *egin.Component
	db      *egorm.Component
	rdb     ecache.Cache
	dao     dao.CaseDAO
	examDAO dao.ExamineDAO
	ctrl    *gomock.Controller
}

func (s *HandlerTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `cases`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `publish_cases`").Error
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())
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
	intrSvc.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(ctx context.Context, biz string, id int64, uid int64) (interactive.Interactive, error) {
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
	module, err := startup.InitModule(nil, nil, &ai.Module{}, intrModule)
	require.NoError(s.T(), err)
	handler := module.Hdl
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	handler.PublicRoutes(server.Engine)
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: uid,
			Data: map[string]string{
				"memberDDL": strconv.FormatInt(time.Now().Add(time.Hour).UnixMilli(), 10),
			},
		}))
	})
	server.Use(middleware.NewCheckMembershipMiddlewareBuilder(nil).Build())
	handler.MemberRoutes(server.Engine)

	s.server = server
	s.db = testioc.InitDB()
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewCaseDao(s.db)
	s.examDAO = dao.NewGORMExamineDAO(s.db)
	s.rdb = testioc.InitCache()
}

func (s *HandlerTestSuite) TestPubList() {
	data := make([]dao.PublishCase, 0, 100)
	for idx := 0; idx < 100; idx++ {
		data = append(data, dao.PublishCase{
			Id:           int64(idx + 1),
			Uid:          uid,
			Title:        fmt.Sprintf("这是发布的案例标题 %d", idx),
			Introduction: fmt.Sprintf("这是发布的案例介绍 %d", idx),
			Utime:        123,
		})
	}
	err := s.db.Create(&data).Error
	require.NoError(s.T(), err)
	testCases := []struct {
		name     string
		req      web.Page
		wantCode int
		wantResp test.Result[web.CasesList]
	}{
		{
			name: "获取成功",
			req: web.Page{
				Limit:  2,
				Offset: 0,
			},
			wantCode: 200,
			wantResp: test.Result[web.CasesList]{
				Data: web.CasesList{
					Cases: []web.Case{
						{
							Id:           100,
							Title:        "这是发布的案例标题 99",
							Introduction: fmt.Sprintf("这是发布的案例介绍 %d", 99),
							Utime:        123,
							Interactive: web.Interactive{
								Liked:      false,
								Collected:  true,
								ViewCnt:    101,
								LikeCnt:    102,
								CollectCnt: 103,
							},
						},
						{
							Id:           99,
							Title:        "这是发布的案例标题 98",
							Introduction: fmt.Sprintf("这是发布的案例介绍 %d", 98),
							Utime:        123,
							Interactive: web.Interactive{
								Liked:      true,
								Collected:  false,
								ViewCnt:    100,
								LikeCnt:    101,
								CollectCnt: 102,
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
			wantResp: test.Result[web.CasesList]{
				Data: web.CasesList{
					Cases: []web.Case{
						{
							Id:           1,
							Title:        "这是发布的案例标题 0",
							Introduction: "这是发布的案例介绍 0",
							Utime:        123,
							Interactive: web.Interactive{
								Liked:      true,
								Collected:  false,
								ViewCnt:    2,
								LikeCnt:    3,
								CollectCnt: 4,
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
				"/case/pub/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.CasesList]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *HandlerTestSuite) TestPubDetail() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := s.db.Create(&dao.PublishCase{
		Id:           3,
		Uid:          uid,
		Introduction: "redis案例介绍",
		Labels: sqlx.JsonColumn[[]string]{
			Valid: true,
			Val:   []string{"Redis"},
		},
		Status:     domain.PublishedStatus.ToUint8(),
		Title:      "redis案例标题",
		Content:    "redis案例内容",
		GithubRepo: "redis github仓库",
		GiteeRepo:  "redis gitee仓库",
		Keywords:   "redis_keywords",
		Shorthand:  "redis_shorthand",
		Highlight:  "redis_highlight",
		Guidance:   "redis_guidance",
		Biz:        "ai",
		BizId:      13,
		Utime:      13,
	}).Error
	require.NoError(s.T(), err)
	// 插入测试记录
	err = s.examDAO.SaveResult(ctx, dao.CaseExamineRecord{
		Uid:    uid,
		Cid:    3,
		Tid:    shortuuid.New(),
		Result: 1,
	})
	require.NoError(s.T(), err)

	testCases := []struct {
		name string

		req      web.CaseId
		wantCode int
		wantResp test.Result[web.Case]
	}{
		{
			name: "查询到了数据",
			req: web.CaseId{
				Cid: 3,
			},
			wantCode: 200,
			wantResp: test.Result[web.Case]{
				Data: web.Case{
					Id:           3,
					Labels:       []string{"Redis"},
					Title:        "redis案例标题",
					Introduction: "redis案例介绍",
					Content:      "redis案例内容",
					GithubRepo:   "redis github仓库",
					GiteeRepo:    "redis gitee仓库",
					Status:       domain.PublishedStatus.ToUint8(),
					Keywords:     "redis_keywords",
					Shorthand:    "redis_shorthand",
					Highlight:    "redis_highlight",
					Guidance:     "redis_guidance",
					Biz:          "ai",
					BizId:        13,
					Utime:        13,
					Interactive: web.Interactive{
						Liked:      true,
						Collected:  false,
						ViewCnt:    4,
						LikeCnt:    5,
						CollectCnt: 6,
					},
					ExamineResult: 1,
				},
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/cases/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Case]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
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
