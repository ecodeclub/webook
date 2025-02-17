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

	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/ai"
	"github.com/gin-gonic/gin"

	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/event"
	eveMocks "github.com/ecodeclub/webook/internal/cases/internal/event/mocks"
	"github.com/ecodeclub/webook/internal/interactive"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/cases/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/cases/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type AdminCaseHandlerTestSuite struct {
	suite.Suite
	server                *egin.Component
	db                    *egorm.Component
	rdb                   ecache.Cache
	dao                   dao.CaseDAO
	ctrl                  *gomock.Controller
	producer              *eveMocks.MockSyncEventProducer
	knowledgeBaseProducer *eveMocks.MockKnowledgeBaseEventProducer
}

func (s *AdminCaseHandlerTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `cases`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `publish_cases`").Error
	require.NoError(s.T(), err)
}

func (s *AdminCaseHandlerTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())
	s.producer = eveMocks.NewMockSyncEventProducer(s.ctrl)
	s.knowledgeBaseProducer = eveMocks.NewMockKnowledgeBaseEventProducer(s.ctrl)
	intrModule := &interactive.Module{}

	module, err := startup.InitModule(s.producer,
		s.knowledgeBaseProducer, &ai.Module{},
		&member.Module{}, session.DefaultProvider(), intrModule)
	require.NoError(s.T(), err)
	handler := module.AdminHandler
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
	handler.PrivateRoutes(server.Engine)
	s.server = server
	s.db = testioc.InitDB()
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewCaseDao(s.db)
	s.rdb = testioc.InitCache()
}

func (s *AdminCaseHandlerTestSuite) TestSave() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.SaveReq
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "新建",
			before: func(t *testing.T) {
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				ca, err := s.dao.GetCaseByID(ctx, 1)
				require.NoError(t, err)
				s.assertCase(t, dao.Case{
					Uid:          uid,
					Title:        "案例1",
					Content:      "案例1内容",
					Introduction: "案例1介绍",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
					BizId:      11,
					Biz:        "question",
					Status:     domain.UnPublishedStatus.ToUint8(),
					GithubRepo: "www.github.com",
					GiteeRepo:  "www.gitee.com",
					Keywords:   "mysql_keywords",
					Shorthand:  "mysql_shorthand",
					Highlight:  "mysql_highlight",
					Guidance:   "mysql_guidance",
				}, ca)
			},
			req: web.SaveReq{
				Case: web.Case{
					Title:        "案例1",
					Content:      "案例1内容",
					Introduction: "案例1介绍",
					Labels:       []string{"MySQL"},
					GithubRepo:   "www.github.com",
					GiteeRepo:    "www.gitee.com",
					Keywords:     "mysql_keywords",
					Shorthand:    "mysql_shorthand",
					Highlight:    "mysql_highlight",
					BizId:        11,
					Biz:          "question",
					Guidance:     "mysql_guidance",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		},
		{
			name: "部分更新",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Case{
					Id:           2,
					Uid:          uid,
					Title:        "老的案例标题",
					Introduction: "老的案例介绍",
					Content:      "老的案例内容",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"old-MySQL"},
					},
					BizId:      12,
					Biz:        "xxx",
					GithubRepo: "old-github.com",
					GiteeRepo:  "old-gitee.com",
					Keywords:   "old_mysql_keywords",
					Shorthand:  "old_mysql_shorthand",
					Highlight:  "old_mysql_highlight",
					Guidance:   "old_mysql_guidance",
					Ctime:      123,
					Utime:      234,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				ca, err := s.dao.GetCaseByID(ctx, 2)
				require.NoError(t, err)
				assert.True(t, ca.Utime > 234)
				assert.Equal(t, int64(123), ca.Ctime)
				s.assertCase(t, dao.Case{
					Uid:          uid,
					Title:        "案例2",
					Introduction: "案例2介绍",
					Content:      "案例2内容",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
					Status:     domain.UnPublishedStatus.ToUint8(),
					GithubRepo: "www.github.com",
					GiteeRepo:  "www.gitee.com",
					Keywords:   "mysql_keywords",
					Shorthand:  "mysql_shorthand",
					Highlight:  "mysql_highlight",
					Guidance:   "mysql_guidance",
					BizId:      11,
					Biz:        "question",
				}, ca)
			},
			req: web.SaveReq{
				Case: web.Case{
					Id:           2,
					Title:        "案例2",
					Introduction: "案例2介绍",
					Content:      "案例2内容",
					Labels:       []string{"MySQL"},
					GithubRepo:   "www.github.com",
					GiteeRepo:    "www.gitee.com",
					Keywords:     "mysql_keywords",
					Shorthand:    "mysql_shorthand",
					Highlight:    "mysql_highlight",
					Guidance:     "mysql_guidance",
					BizId:        11,
					Biz:          "question",
				},
			},
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
				"/cases/save", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
			// 清理掉 123 的数据
			err = s.db.Exec("TRUNCATE table `cases`").Error
			require.NoError(t, err)
			err = s.db.Exec("TRUNCATE table `publish_cases`").Error
			require.NoError(t, err)
		})
	}
}

func (s *AdminCaseHandlerTestSuite) TestList() {
	// 插入一百条
	data := make([]dao.Case, 0, 100)
	for idx := 0; idx < 100; idx++ {
		data = append(data, dao.Case{
			Uid:          uid,
			Title:        fmt.Sprintf("这是案例标题 %d", idx),
			Content:      fmt.Sprintf("这是案例内容 %d", idx),
			Introduction: fmt.Sprintf("这是案例介绍 %d", idx),
			Status:       domain.UnPublishedStatus.ToUint8(),
			Utime:        0,
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
					Total: 100,
					Cases: []web.Case{
						{
							Id:           100,
							Title:        "这是案例标题 99",
							Introduction: fmt.Sprintf("这是案例介绍 %d", 99),
							Status:       domain.UnPublishedStatus.ToUint8(),
							Utime:        0,
						},
						{
							Id:           99,
							Title:        "这是案例标题 98",
							Introduction: fmt.Sprintf("这是案例介绍 %d", 98),
							Status:       domain.UnPublishedStatus.ToUint8(),
							Utime:        0,
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
					Total: 100,
					Cases: []web.Case{
						{
							Id:           1,
							Title:        "这是案例标题 0",
							Introduction: fmt.Sprintf("这是案例介绍 %d", 0),
							Status:       domain.UnPublishedStatus.ToUint8(),
							Utime:        0,
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
				"/cases/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.CasesList]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = s.rdb.Delete(ctx, "webook:question:total")
	require.NoError(s.T(), err)
}

func (s *AdminCaseHandlerTestSuite) TestDetail() {
	err := s.db.Create(&dao.Case{
		Id:  3,
		Uid: uid,
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
		Biz:        "case",
		BizId:      11,
		Ctime:      12,
		Utime:      12,
	}).Error
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
					Id:         3,
					Labels:     []string{"Redis"},
					Title:      "redis案例标题",
					Content:    "redis案例内容",
					GithubRepo: "redis github仓库",
					GiteeRepo:  "redis gitee仓库",
					Status:     domain.PublishedStatus.ToUint8(),
					Keywords:   "redis_keywords",
					Shorthand:  "redis_shorthand",
					Highlight:  "redis_highlight",
					Guidance:   "redis_guidance",
					Biz:        "case",
					BizId:      11,
					Utime:      12,
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

func (s *AdminCaseHandlerTestSuite) TestPublish() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.SaveReq
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "新建并发布",
			before: func(t *testing.T) {
				s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).
					MaxTimes(1).
					Return(nil)
				s.knowledgeBaseProducer.EXPECT().Produce(gomock.Any(), gomock.Any()).
					MaxTimes(1).
					Return(nil)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				ca, err := s.dao.GetCaseByID(ctx, 1)
				require.NoError(t, err)
				wantCase := dao.Case{
					Uid:          uid,
					Title:        "案例1",
					Content:      "案例1内容",
					Introduction: "案例1介绍",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
					Status: domain.PublishedStatus.ToUint8(),

					GithubRepo: "www.github.com",
					GiteeRepo:  "www.gitee.com",
					Keywords:   "mysql_keywords",
					Shorthand:  "mysql_shorthand",
					Highlight:  "mysql_highlight",
					Guidance:   "mysql_guidance",
					Biz:        "case",
					BizId:      11,
				}
				s.assertCase(t, wantCase, ca)
				publishCase, err := s.dao.GetPublishCase(ctx, 1)
				require.NoError(t, err)
				s.assertCase(t, wantCase, dao.Case(publishCase))
				s.cacheAssertCase(domain.Case{
					Id:           1,
					Uid:          uid,
					Title:        "案例1",
					Content:      "案例1内容",
					Introduction: "案例1介绍",
					Labels: []string{
						"MySQL",
					},
					Status:     domain.PublishedStatus,
					GithubRepo: "www.github.com",
					GiteeRepo:  "www.gitee.com",
					Keywords:   "mysql_keywords",
					Shorthand:  "mysql_shorthand",
					Highlight:  "mysql_highlight",
					Guidance:   "mysql_guidance",
					Biz:        "case",
					BizId:      11,
				})
				s.cacheAssertCaseList("case", []domain.Case{
					{
						Id:           1,
						Title:        "案例1",
						Introduction: "案例1介绍",
						Labels: []string{
							"MySQL",
						},
						Status: domain.PublishedStatus,
					},
				})

			},
			req: web.SaveReq{
				Case: web.Case{
					Title:        "案例1",
					Content:      "案例1内容",
					Introduction: "案例1介绍",
					Labels:       []string{"MySQL"},
					GithubRepo:   "www.github.com",
					GiteeRepo:    "www.gitee.com",
					Keywords:     "mysql_keywords",
					Shorthand:    "mysql_shorthand",
					Highlight:    "mysql_highlight",
					Guidance:     "mysql_guidance",
					Biz:          "case",
					BizId:        11,
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		},
		{
			name: "更新并发布",
			// publish_case表的ctime不更新
			before: func(t *testing.T) {
				s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).
					MaxTimes(1).
					Return(nil)
				s.knowledgeBaseProducer.EXPECT().Produce(gomock.Any(), gomock.Any()).
					MaxTimes(1).
					Return(nil)
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Case{
					Id:           2,
					Uid:          uid,
					Title:        "老的案例标题",
					Content:      "老的案例内容",
					Introduction: "老的案例介绍",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"old-MySQL"},
					},
					GithubRepo: "old-github.com",
					GiteeRepo:  "old-gitee.com",
					Keywords:   "old_mysql_keywords",
					Shorthand:  "old_mysql_shorthand",
					Highlight:  "old_mysql_highlight",
					Guidance:   "old_mysql_guidance",
					Biz:        "question",
					BizId:      11,
					Ctime:      123,
					Utime:      234,
				}).Error
				require.NoError(t, err)
				cs := []domain.Case{
					{
						Id:           2,
						Title:        "老的案例标题",
						Content:      "老的案例内容",
						Introduction: "老的案例介绍",
						Labels:       []string{"old-MySQL"},
						Status:       domain.PublishedStatus,
					},
				}
				csByte, err := json.Marshal(cs)
				require.NoError(t, err)
				err = s.rdb.Set(ctx, "cases:list:question", string(csByte), 24*time.Hour)
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				ca, err := s.dao.GetCaseByID(ctx, 2)
				require.NoError(t, err)
				wantCase := dao.Case{
					Uid:          uid,
					Title:        "案例2",
					Content:      "案例2内容",
					Introduction: "案例2介绍",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
					Status:     domain.PublishedStatus.ToUint8(),
					GithubRepo: "www.github.com",
					GiteeRepo:  "www.gitee.com",
					Keywords:   "mysql_keywords",
					Shorthand:  "mysql_shorthand",
					Highlight:  "mysql_highlight",
					Guidance:   "mysql_guidance",
					Biz:        "question",
					BizId:      12,
				}
				s.assertCase(t, wantCase, ca)
				publishCase, err := s.dao.GetPublishCase(ctx, 2)
				require.NoError(t, err)
				publishCase.Ctime = 123
				s.assertCase(t, wantCase, dao.Case(publishCase))
				s.cacheAssertCase(domain.Case{
					Id:           2,
					Uid:          uid,
					Title:        "案例2",
					Content:      "案例2内容",
					Introduction: "案例2介绍",
					Labels: []string{
						"MySQL",
					},
					Status:     domain.PublishedStatus,
					GithubRepo: "www.github.com",
					GiteeRepo:  "www.gitee.com",
					Keywords:   "mysql_keywords",
					Shorthand:  "mysql_shorthand",
					Highlight:  "mysql_highlight",
					Guidance:   "mysql_guidance",
					Biz:        "question",
					BizId:      12,
				})
				s.cacheAssertCaseList("question", []domain.Case{
					{
						Id:           2,
						Title:        "案例2",
						Introduction: "案例2介绍",
						Labels: []string{
							"MySQL",
						},
						Status: domain.PublishedStatus,
					},
				})
			},
			req: web.SaveReq{
				Case: web.Case{
					Id:           2,
					Title:        "案例2",
					Content:      "案例2内容",
					Introduction: "案例2介绍",
					Labels:       []string{"MySQL"},
					GithubRepo:   "www.github.com",
					GiteeRepo:    "www.gitee.com",
					Keywords:     "mysql_keywords",
					Shorthand:    "mysql_shorthand",
					Highlight:    "mysql_highlight",
					Guidance:     "mysql_guidance",
					Biz:          "question",
					BizId:        12,
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 2,
			},
		},
		{
			name: "publish表有值发布",
			// publish_case表的ctime不更新
			before: func(t *testing.T) {
				s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).
					MaxTimes(1).
					Return(nil)
				s.knowledgeBaseProducer.EXPECT().Produce(gomock.Any(), gomock.Any()).
					MaxTimes(1).
					Return(nil)
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				oldCase := dao.Case{
					Id:           3,
					Uid:          uid,
					Title:        "老的案例标题",
					Introduction: "老的案例介绍",
					Content:      "老的案例内容",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"old-MySQL"},
					},
					GithubRepo: "old-github.com",
					GiteeRepo:  "old-gitee.com",
					Keywords:   "old_mysql_keywords",
					Shorthand:  "old_mysql_shorthand",
					Highlight:  "old_mysql_highlight",
					Guidance:   "old_mysql_guidance",
					Biz:        "question",
					BizId:      12,
					Ctime:      123,
					Utime:      234,
				}
				err := s.db.WithContext(ctx).Create(&oldCase).Error
				require.NoError(t, err)
				pubCase := dao.PublishCase(oldCase)
				err = s.db.WithContext(ctx).Create(pubCase).Error
				require.NoError(t, err)

			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				ca, err := s.dao.GetCaseByID(ctx, 3)
				require.NoError(t, err)
				wantCase := dao.Case{
					Uid:          uid,
					Title:        "案例2",
					Content:      "案例2内容",
					Introduction: "案例2介绍",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
					Status:     domain.PublishedStatus.ToUint8(),
					GithubRepo: "www.github.com",
					GiteeRepo:  "www.gitee.com",
					Keywords:   "mysql_keywords",
					Shorthand:  "mysql_shorthand",
					Highlight:  "mysql_highlight",
					Guidance:   "mysql_guidance",
					Biz:        "ai",
					BizId:      13,
				}
				s.assertCase(t, wantCase, ca)
				publishCase, err := s.dao.GetPublishCase(ctx, 3)
				require.NoError(t, err)
				publishCase.Ctime = 123
				s.assertCase(t, wantCase, dao.Case(publishCase))
				s.cacheAssertCase(domain.Case{
					Id:           3,
					Uid:          uid,
					Title:        "案例2",
					Content:      "案例2内容",
					Introduction: "案例2介绍",
					Labels:       []string{"MySQL"},
					Status:       domain.PublishedStatus,
					GithubRepo:   "www.github.com",
					GiteeRepo:    "www.gitee.com",
					Keywords:     "mysql_keywords",
					Shorthand:    "mysql_shorthand",
					Highlight:    "mysql_highlight",
					Guidance:     "mysql_guidance",
					Biz:          "ai",
					BizId:        13,
				})
			},
			req: web.SaveReq{
				Case: web.Case{
					Id:           3,
					Title:        "案例2",
					Content:      "案例2内容",
					Introduction: "案例2介绍",
					Labels:       []string{"MySQL"},
					GithubRepo:   "www.github.com",
					GiteeRepo:    "www.gitee.com",
					Keywords:     "mysql_keywords",
					Shorthand:    "mysql_shorthand",
					Highlight:    "mysql_highlight",
					Guidance:     "mysql_guidance",
					Biz:          "ai",
					BizId:        13,
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 3,
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/cases/publish", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
			err = s.db.Exec("TRUNCATE table `cases`").Error
			require.NoError(t, err)
			err = s.db.Exec("TRUNCATE table `publish_cases`").Error
			require.NoError(t, err)
		})
	}
}

func (s *AdminCaseHandlerTestSuite) cacheAssertCase(ca domain.Case) {
	t := s.T()
	key := fmt.Sprintf("cases:publish:%d", ca.Id)
	val := s.rdb.Get(context.Background(), key)
	require.NoError(t, val.Err)
	valStr, err := val.String()
	require.NoError(t, err)
	actualCa := domain.Case{}
	json.Unmarshal([]byte(valStr), &actualCa)
	require.True(t, actualCa.Ctime.Unix() > 0)
	require.True(t, actualCa.Utime.Unix() > 0)
	ca.Ctime = actualCa.Ctime
	ca.Utime = actualCa.Utime
	assert.Equal(t, ca, actualCa)
	_, err = s.rdb.Delete(context.Background(), key)
	require.NoError(t, err)
}

func (s *AdminCaseHandlerTestSuite) TestEvent() {
	t := s.T()
	var evt event.Case
	var wg sync.WaitGroup
	wg.Add(1)
	s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, caseEvent event.CaseEvent) error {
		var eve event.Case
		err := json.Unmarshal([]byte(caseEvent.Data), &eve)
		if err != nil {
			return err
		}
		evt = eve
		wg.Done()
		return nil
	}).Times(1)
	s.knowledgeBaseProducer.EXPECT().Produce(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, baseEvent event.KnowledgeBaseEvent) error {
		var ca domain.Case
		err := json.Unmarshal(baseEvent.Data, &ca)
		require.NoError(t, err)
		assert.Equal(t, domain.BizCase, baseEvent.Biz)
		assert.Equal(t, baseEvent.BizID, ca.Id)
		assert.Equal(t, fmt.Sprintf("case_%d", ca.Id), baseEvent.Name)
		ca.Ctime = time.UnixMilli(123)
		ca.Utime = time.UnixMilli(123)
		assert.Equal(t, domain.Case{
			Id:         baseEvent.BizID,
			Title:      "案例2",
			Uid:        uid,
			Content:    "案例2内容",
			Labels:     []string{"MySQL"},
			GithubRepo: "www.github.com",
			GiteeRepo:  "www.gitee.com",
			Keywords:   "mysql_keywords",
			Shorthand:  "mysql_shorthand",
			Highlight:  "mysql_highlight",
			Guidance:   "mysql_guidance",
			Status:     2,
			Biz:        "bbb",
			Ctime:      time.UnixMilli(123),
			Utime:      time.UnixMilli(123),
		}, ca)
		return nil
	})
	// 发布
	publishReq := web.SaveReq{
		Case: web.Case{
			Title:      "案例2",
			Content:    "案例2内容",
			Labels:     []string{"MySQL"},
			GithubRepo: "www.github.com",
			GiteeRepo:  "www.gitee.com",
			Keywords:   "mysql_keywords",
			Shorthand:  "mysql_shorthand",
			Highlight:  "mysql_highlight",
			Guidance:   "mysql_guidance",
			Biz:        "bbb",
		},
	}
	req2, err := http.NewRequest(http.MethodPost,
		"/cases/publish", iox.NewJSONReader(publishReq))
	req2.Header.Set("content-type", "application/json")
	require.NoError(t, err)
	recorder := test.NewJSONResponseRecorder[int64]()
	s.server.ServeHTTP(recorder, req2)
	require.Equal(t, 200, recorder.Code)
	wg.Wait()
	assert.True(t, evt.Ctime > 0)
	evt.Ctime = 0
	assert.True(t, evt.Utime > 0)
	evt.Utime = 0
	assert.True(t, evt.Id > 0)
	evt.Id = 0
	assert.Equal(t, event.Case{
		Title:      "案例2",
		Uid:        uid,
		Content:    "案例2内容",
		Labels:     []string{"MySQL"},
		GithubRepo: "www.github.com",
		GiteeRepo:  "www.gitee.com",
		Keywords:   "mysql_keywords",
		Shorthand:  "mysql_shorthand",
		Highlight:  "mysql_highlight",
		Guidance:   "mysql_guidance",
		Status:     2,
	}, evt)
	time.Sleep(3 * time.Second)
}

// assertCase 不比较 id
func (s *AdminCaseHandlerTestSuite) assertCase(t *testing.T, expect dao.Case, ca dao.Case) {
	assert.True(t, ca.Id > 0)
	assert.True(t, ca.Ctime > 0)
	assert.True(t, ca.Utime > 0)
	ca.Id = 0
	ca.Ctime = 0
	ca.Utime = 0
	assert.Equal(t, expect, ca)
}

func TestAdminCaseHandler(t *testing.T) {
	suite.Run(t, new(AdminCaseHandlerTestSuite))
}

func (s *AdminCaseHandlerTestSuite) cacheAssertCaseList(biz string, cases []domain.Case) {
	key := fmt.Sprintf("cases:list:%s", biz)
	val := s.rdb.Get(context.Background(), key)
	require.NoError(s.T(), val.Err)

	var cs []domain.Case
	err := json.Unmarshal([]byte(val.Val.(string)), &cs)
	require.NoError(s.T(), err)
	require.Equal(s.T(), len(cases), len(cs))
	for idx, q := range cs {
		require.True(s.T(), q.Utime.UnixMilli() > 0)
		require.True(s.T(), q.Id > 0)
		cs[idx].Id = cases[idx].Id
		cs[idx].Utime = cases[idx].Utime
		cs[idx].Ctime = cases[idx].Ctime

	}
	assert.Equal(s.T(), cases, cs)
	_, err = s.rdb.Delete(context.Background(), key)
	require.NoError(s.T(), err)
}
