//go:build e2e

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

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
	server *egin.Component
	db     *egorm.Component
	rdb    ecache.Cache
	dao    dao.CaseDAO
}

func (s *HandlerTestSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `cases`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("DROP TABLE `publish_cases`").Error
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `cases`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `publish_cases`").Error
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) SetupSuite() {
	handler, err := startup.InitHandler()
	require.NoError(s.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()

	handler.PublicRoutes(server.Engine)
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid:  uid,
			Data: map[string]string{"creator": "true", "memberDDL": "2099-01-01 23:59:59"},
		}))
	})
	handler.PrivateRoutes(server.Engine)
	server.Use(middleware.NewCheckMembershipMiddlewareBuilder(nil).Build())
	handler.MemberRoutes(server.Engine)

	s.server = server
	s.db = testioc.InitDB()
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewCaseDao(s.db)
	s.rdb = testioc.InitCache()
}

func (s *HandlerTestSuite) TestSave() {
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
					Uid:     uid,
					Title:   "案例1",
					Content: "案例1内容",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
					CodeRepo:  "www.github.com",
					Keywords:  "mysql_keywords",
					Shorthand: "mysql_shorthand",
					Highlight: "mysql_highlight",
					Guidance:  "mysql_guidance",
				}, ca)
			},
			req: web.SaveReq{
				Case: web.Case{
					Title:     "案例1",
					Content:   "案例1内容",
					Labels:    []string{"MySQL"},
					CodeRepo:  "www.github.com",
					Keywords:  "mysql_keywords",
					Shorthand: "mysql_shorthand",
					Highlight: "mysql_highlight",
					Guidance:  "mysql_guidance",
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
					Id:      2,
					Uid:     uid,
					Title:   "老的案例标题",
					Content: "老的案例内容",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"old-MySQL"},
					},
					CodeRepo:  "old-github.com",
					Keywords:  "old_mysql_keywords",
					Shorthand: "old_mysql_shorthand",
					Highlight: "old_mysql_highlight",
					Guidance:  "old_mysql_guidance",
					Ctime:     123,
					Utime:     234,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				ca, err := s.dao.GetCaseByID(ctx, 2)
				require.NoError(t, err)
				s.assertCase(t, dao.Case{
					Uid:     uid,
					Title:   "案例2",
					Content: "案例2内容",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
					CodeRepo:  "www.github.com",
					Keywords:  "mysql_keywords",
					Shorthand: "mysql_shorthand",
					Highlight: "mysql_highlight",
					Guidance:  "mysql_guidance",
				}, ca)
			},
			req: web.SaveReq{
				Case: web.Case{
					Id:        2,
					Title:     "案例2",
					Content:   "案例2内容",
					Labels:    []string{"MySQL"},
					CodeRepo:  "www.github.com",
					Keywords:  "mysql_keywords",
					Shorthand: "mysql_shorthand",
					Highlight: "mysql_highlight",
					Guidance:  "mysql_guidance",
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
				"/case/save", iox.NewJSONReader(tc.req))
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

func (s *HandlerTestSuite) TestList() {
	// 插入一百条
	data := make([]dao.Case, 0, 100)
	for idx := 0; idx < 100; idx++ {
		data = append(data, dao.Case{
			Uid:     uid,
			Title:   fmt.Sprintf("这是案例标题 %d", idx),
			Content: fmt.Sprintf("这是案例内容 %d", idx),
			Utime:   0,
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
							Id:      100,
							Title:   "这是案例标题 99",
							Content: "这是案例内容 99",
							Utime:   time.UnixMilli(0).Format(time.DateTime),
						},
						{
							Id:      99,
							Title:   "这是案例标题 98",
							Content: "这是案例内容 98",
							Utime:   time.UnixMilli(0).Format(time.DateTime),
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
							Id:      1,
							Title:   "这是案例标题 0",
							Content: "这是案例内容 0",
							Utime:   time.UnixMilli(0).Format(time.DateTime),
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/case/list", iox.NewJSONReader(tc.req))
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

func (s *HandlerTestSuite) TestDetail() {
	err := s.db.Create(&dao.Case{
		Id:  3,
		Uid: uid,
		Labels: sqlx.JsonColumn[[]string]{
			Valid: true,
			Val:   []string{"Redis"},
		},
		Title:     "redis案例标题",
		Content:   "redis案例内容",
		CodeRepo:  "redis仓库",
		Keywords:  "redis_keywords",
		Shorthand: "redis_shorthand",
		Highlight: "redis_highlight",
		Guidance:  "redis_guidance",
		Ctime:     12,
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
					Id:        3,
					Labels:    []string{"Redis"},
					Title:     "redis案例标题",
					Content:   "redis案例内容",
					CodeRepo:  "redis仓库",
					Keywords:  "redis_keywords",
					Shorthand: "redis_shorthand",
					Highlight: "redis_highlight",
					Guidance:  "redis_guidance",
					Utime:     time.UnixMilli(12).Format(time.DateTime),
				},
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/case/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Case]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}

}

func (s *HandlerTestSuite) TestPublish() {
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

			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				ca, err := s.dao.GetCaseByID(ctx, 1)
				require.NoError(t, err)
				wantCase := dao.Case{
					Uid:     uid,
					Title:   "案例1",
					Content: "案例1内容",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
					CodeRepo:  "www.github.com",
					Keywords:  "mysql_keywords",
					Shorthand: "mysql_shorthand",
					Highlight: "mysql_highlight",
					Guidance:  "mysql_guidance",
				}
				s.assertCase(t, wantCase, ca)
				publishCase, err := s.dao.GetPublishCase(ctx, 1)
				require.NoError(t, err)
				s.assertCase(t, wantCase, dao.Case(publishCase))
			},
			req: web.SaveReq{
				Case: web.Case{
					Title:     "案例1",
					Content:   "案例1内容",
					Labels:    []string{"MySQL"},
					CodeRepo:  "www.github.com",
					Keywords:  "mysql_keywords",
					Shorthand: "mysql_shorthand",
					Highlight: "mysql_highlight",
					Guidance:  "mysql_guidance",
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
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Case{
					Id:      2,
					Uid:     uid,
					Title:   "老的案例标题",
					Content: "老的案例内容",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"old-MySQL"},
					},
					CodeRepo:  "old-github.com",
					Keywords:  "old_mysql_keywords",
					Shorthand: "old_mysql_shorthand",
					Highlight: "old_mysql_highlight",
					Guidance:  "old_mysql_guidance",
					Ctime:     123,
					Utime:     234,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				ca, err := s.dao.GetCaseByID(ctx, 2)
				require.NoError(t, err)
				wantCase := dao.Case{
					Uid:     uid,
					Title:   "案例2",
					Content: "案例2内容",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
					CodeRepo:  "www.github.com",
					Keywords:  "mysql_keywords",
					Shorthand: "mysql_shorthand",
					Highlight: "mysql_highlight",
					Guidance:  "mysql_guidance",
				}
				s.assertCase(t, wantCase, ca)
				publishCase, err := s.dao.GetPublishCase(ctx, 2)
				require.NoError(t, err)
				publishCase.Ctime = 123
				s.assertCase(t, wantCase, dao.Case(publishCase))
			},
			req: web.SaveReq{
				Case: web.Case{
					Id:        2,
					Title:     "案例2",
					Content:   "案例2内容",
					Labels:    []string{"MySQL"},
					CodeRepo:  "www.github.com",
					Keywords:  "mysql_keywords",
					Shorthand: "mysql_shorthand",
					Highlight: "mysql_highlight",
					Guidance:  "mysql_guidance",
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
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Case{
					Id:      3,
					Uid:     uid,
					Title:   "老的案例标题",
					Content: "老的案例内容",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"old-MySQL"},
					},
					CodeRepo:  "old-github.com",
					Keywords:  "old_mysql_keywords",
					Shorthand: "old_mysql_shorthand",
					Highlight: "old_mysql_highlight",
					Guidance:  "old_mysql_guidance",
					Ctime:     123,
					Utime:     234,
				}).Error
				require.NoError(t, err)
				err = s.db.WithContext(ctx).Create(&dao.PublishCase{
					Id:      3,
					Uid:     uid,
					Title:   "老的案例标题",
					Content: "老的案例内容",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"old-MySQL"},
					},
					CodeRepo:  "old-github.com",
					Keywords:  "old_mysql_keywords",
					Shorthand: "old_mysql_shorthand",
					Highlight: "old_mysql_highlight",
					Guidance:  "old_mysql_guidance",
					Ctime:     123,
					Utime:     234,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				ca, err := s.dao.GetCaseByID(ctx, 3)
				require.NoError(t, err)
				wantCase := dao.Case{
					Uid:     uid,
					Title:   "案例2",
					Content: "案例2内容",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
					CodeRepo:  "www.github.com",
					Keywords:  "mysql_keywords",
					Shorthand: "mysql_shorthand",
					Highlight: "mysql_highlight",
					Guidance:  "mysql_guidance",
				}
				s.assertCase(t, wantCase, ca)
				publishCase, err := s.dao.GetPublishCase(ctx, 3)
				require.NoError(t, err)
				publishCase.Ctime = 123
				s.assertCase(t, wantCase, dao.Case(publishCase))
			},
			req: web.SaveReq{
				Case: web.Case{
					Id:        3,
					Title:     "案例2",
					Content:   "案例2内容",
					Labels:    []string{"MySQL"},
					CodeRepo:  "www.github.com",
					Keywords:  "mysql_keywords",
					Shorthand: "mysql_shorthand",
					Highlight: "mysql_highlight",
					Guidance:  "mysql_guidance",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 3,
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/case/publish", iox.NewJSONReader(tc.req))
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

func (s *HandlerTestSuite) TestPublist() {
	data := make([]dao.PublishCase, 0, 100)
	for idx := 0; idx < 100; idx++ {
		data = append(data, dao.PublishCase{
			Uid:     uid,
			Title:   fmt.Sprintf("这是发布的案例标题 %d", idx),
			Content: fmt.Sprintf("这是发布的案例内容 %d", idx),
			Utime:   0,
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
							Id:    100,
							Title: "这是发布的案例标题 99",
							Utime: time.UnixMilli(0).Format(time.DateTime),
						},
						{
							Id:    99,
							Title: "这是发布的案例标题 98",
							Utime: time.UnixMilli(0).Format(time.DateTime),
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
							Id:    1,
							Title: "这是发布的案例标题 0",
							Utime: time.UnixMilli(0).Format(time.DateTime),
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = s.rdb.Delete(ctx, "webook:question:total")
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) TestPubDetail() {
	err := s.db.Create(&dao.PublishCase{
		Id:  3,
		Uid: uid,
		Labels: sqlx.JsonColumn[[]string]{
			Valid: true,
			Val:   []string{"Redis"},
		},
		Title:     "redis案例标题",
		Content:   "redis案例内容",
		CodeRepo:  "redis仓库",
		Keywords:  "redis_keywords",
		Shorthand: "redis_shorthand",
		Highlight: "redis_highlight",
		Guidance:  "redis_guidance",
		Utime:     13,
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
					Id:        3,
					Labels:    []string{"Redis"},
					Title:     "redis案例标题",
					Content:   "redis案例内容",
					CodeRepo:  "redis仓库",
					Keywords:  "redis_keywords",
					Shorthand: "redis_shorthand",
					Highlight: "redis_highlight",
					Guidance:  "redis_guidance",
					Utime:     time.UnixMilli(13).Format(time.DateTime),
				},
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/case/pub/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Case]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

// assertCase 不比较 id
func (s *HandlerTestSuite) assertCase(t *testing.T, expect dao.Case, ca dao.Case) {
	assert.True(t, ca.Id > 0)
	assert.True(t, ca.Ctime > 0)
	assert.True(t, ca.Utime > 0)
	ca.Id = 0
	ca.Ctime = 0
	ca.Utime = 0
	assert.Equal(t, expect, ca)
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
