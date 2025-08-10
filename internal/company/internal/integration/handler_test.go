//go:build e2e

package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/company/internal/integration/startup"
	comdao "github.com/ecodeclub/webook/internal/company/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/company/internal/web"
	"github.com/ecodeclub/webook/internal/pkg/middleware"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CompanyTestSuite struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
}

func (c *CompanyTestSuite) SetupSuite() {
	module, err := startup.InitModule()
	require.NoError(c.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	handler := module.Hdl
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{Uid: 123}))
	})
	handler.PrivateRoutes(server.Engine)
	server.Use(middleware.NewCheckMembershipMiddlewareBuilder(nil).Build())
	c.server = server
	c.db = testioc.InitDB()
}

func (c *CompanyTestSuite) TearDownTest() {
	require.NoError(c.T(), c.db.Exec("TRUNCATE TABLE `companies`").Error)
}

func (c *CompanyTestSuite) Test_Save() {
	dao := comdao.NewGORMCompanyDAO(c.db)
	testcases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T, id int64)
		req      web.SaveCompanyReq
		wantCode int
	}{
		{
			name:   "创建公司",
			before: func(t *testing.T) {},
			after: func(t *testing.T, id int64) {
				got, err := dao.FindById(context.Background(), id)
				require.NoError(t, err)
				require.Equal(t, "ABC Corp", got.Name)
				require.NotZero(t, got.Ctime)
				require.NotZero(t, got.Utime)
			},
			req:      web.SaveCompanyReq{Name: "ABC Corp"},
			wantCode: 200,
		},
		{
			name: "更新公司",
			before: func(t *testing.T) {
				err := c.db.WithContext(context.Background()).Create(&comdao.Company{
					Id:    2000,
					Name:  "Old Name",
					Ctime: 123,
					Utime: 123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T, id int64) {
				got, err := dao.FindById(context.Background(), id)
				require.NoError(t, err)
				require.Equal(t, int64(2000), got.Id)
				require.Equal(t, "New Name", got.Name)
				require.NotZero(t, got.Utime)
			},
			req:      web.SaveCompanyReq{ID: 2000, Name: "New Name"},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		c.T().Run(tc.name, func(t *testing.T) {
			// prepare
			if tc.before != nil {
				tc.before(t)
			}
			req, err := http.NewRequest(http.MethodPost, "/companies/save", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[int64]()
			c.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			if tc.after != nil {
				tc.after(t, recorder.MustScan().Data)
			}
		})
	}
}

func (c *CompanyTestSuite) Test_Detail() {
	dao := comdao.NewGORMCompanyDAO(c.db)
	testcases := []struct {
		name     string
		before   func(t *testing.T) int64
		after    func(t *testing.T, id int64)
		req      web.IdReq
		wantCode int
	}{
		{
			name: "查询详情_存在",
			before: func(t *testing.T) int64 {
				err := c.db.WithContext(context.Background()).Create(&comdao.Company{
					Id:    10,
					Name:  "Detail Co",
					Ctime: 1,
					Utime: 1,
				}).Error
				require.NoError(t, err)
				return 10
			},
			after: func(t *testing.T, id int64) {
				got, err := dao.FindById(context.Background(), id)
				require.NoError(t, err)
				require.Equal(t, "Detail Co", got.Name)
			},
			req:      web.IdReq{},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		c.T().Run(tc.name, func(t *testing.T) {
			id := int64(0)
			if tc.before != nil {
				id = tc.before(t)
			}
			req, err := http.NewRequest(http.MethodPost, "/companies/detail", iox.NewJSONReader(web.IdReq{Id: id}))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[web.CompanyVO]()
			c.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			if tc.after != nil {
				tc.after(t, id)
			}
		})
	}
}

func (c *CompanyTestSuite) Test_List() {
	testcases := []struct {
		name     string
		before   func(t *testing.T)
		req      web.Page
		wantCode int
	}{
		{
			name: "列出公司",
			before: func(t *testing.T) {
				for _, name := range []string{"A", "B", "C"} {
					req, err := http.NewRequest(http.MethodPost, "/companies/save", iox.NewJSONReader(web.SaveCompanyReq{Name: name}))
					require.NoError(t, err)
					req.Header.Set("content-type", "application/json")
					recorder := test.NewJSONResponseRecorder[int64]()
					c.server.ServeHTTP(recorder, req)
					require.Equal(t, 200, recorder.Code)
				}
			},
			req:      web.Page{Offset: 0, Limit: 10},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		c.T().Run(tc.name, func(t *testing.T) {
			if tc.before != nil {
				tc.before(t)
			}
			req, err := http.NewRequest(http.MethodPost, "/companies/list", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[web.ListCompanyResp]()
			c.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
		})
	}
}

func (c *CompanyTestSuite) Test_Delete() {
	dao := comdao.NewGORMCompanyDAO(c.db)
	testcases := []struct {
		name     string
		before   func(t *testing.T) int64
		after    func(t *testing.T, id int64)
		req      web.IdReq
		wantCode int
	}{
		{
			name: "删除存在的公司",
			before: func(t *testing.T) int64 {
				// create
				req, err := http.NewRequest(http.MethodPost, "/companies/save", iox.NewJSONReader(web.SaveCompanyReq{Name: "ToDelete"}))
				require.NoError(t, err)
				req.Header.Set("content-type", "application/json")
				recorder := test.NewJSONResponseRecorder[int64]()
				c.server.ServeHTTP(recorder, req)
				require.Equal(t, 200, recorder.Code)
				return recorder.MustScan().Data
			},
			after: func(t *testing.T, id int64) {
				_, err := dao.FindById(context.Background(), id)
				require.Error(t, err)
			},
			req:      web.IdReq{},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		c.T().Run(tc.name, func(t *testing.T) {
			id := int64(0)
			if tc.before != nil {
				id = tc.before(t)
			}
			req, err := http.NewRequest(http.MethodPost, "/companies/delete", iox.NewJSONReader(web.IdReq{Id: id}))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[string]()
			c.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			if tc.after != nil {
				tc.after(t, id)
			}
		})
	}
}

func TestCompany(t *testing.T) {
	suite.Run(t, new(CompanyTestSuite))
}
