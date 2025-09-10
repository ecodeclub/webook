//go:build e2e

package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/ecodeclub/webook/internal/company"

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
	// c端company
	cServer *egin.Component
	db      *egorm.Component
	svc     company.Service
}

func (c *CompanyTestSuite) SetupSuite() {
	module, err := startup.InitModule()
	require.NoError(c.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	econf.Set("cServer", map[string]any{"contextTimeout": "1s"})
	cServer := egin.Load("cServer").Build()
	adminHdl := module.AdminHdl
	hdl := module.Hdl
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{Uid: 123}))
	})
	adminHdl.PrivateRoutes(server.Engine)
	cServer.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{Uid: 123}))
	})
	hdl.PrivateRoutes(cServer.Engine)
	server.Use(middleware.NewCheckMembershipMiddlewareBuilder(nil).Build())
	c.server = server
	c.cServer = cServer
	c.svc = module.Svc
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

func (c *CompanyTestSuite) Test_CList() {
	testcases := []struct {
		name     string
		before   func(t *testing.T)
		req      web.Page
		wantCode int
		verify   func(t *testing.T, resp web.ListCompanyResp)
	}{
		{
			name: "C端列出公司",
			before: func(t *testing.T) {
				companies := []comdao.Company{
					{Id: 101, Name: "公司A", Ctime: 123, Utime: 123},
					{Id: 102, Name: "公司B", Ctime: 123, Utime: 123},
					{Id: 103, Name: "公司C", Ctime: 123, Utime: 123},
				}
				for _, company := range companies {
					err := c.db.WithContext(context.Background()).Create(&company).Error
					require.NoError(t, err)
				}
			},
			req:      web.Page{Offset: 0, Limit: 10},
			wantCode: 200,
			verify: func(t *testing.T, resp web.ListCompanyResp) {
				require.GreaterOrEqual(t, len(resp.List), 3)

				// 验证返回的数据中包含我们创建的公司
				companyNames := make(map[string]bool)
				for _, company := range resp.List {
					companyNames[company.Name] = true
				}

				require.True(t, companyNames["公司A"], "返回结果应包含'公司A'")
				require.True(t, companyNames["公司B"], "返回结果应包含'公司B'")
				require.True(t, companyNames["公司C"], "返回结果应包含'公司C'")
			},
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
			c.cServer.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)

			result := recorder.MustScan()
			if tc.verify != nil {
				tc.verify(t, result.Data)
			}
		})
	}
}

func (c *CompanyTestSuite) Test_CDetail() {
	testcases := []struct {
		name     string
		before   func(t *testing.T) int64
		req      web.IdReq
		wantCode int
		verify   func(t *testing.T, resp web.CompanyVO, id int64)
	}{
		{
			name: "C端查询详情_存在",
			before: func(t *testing.T) int64 {
				company := comdao.Company{
					Id:    20,
					Name:  "C端详情公司",
					Ctime: 1000,
					Utime: 2000,
				}
				err := c.db.WithContext(context.Background()).Create(&company).Error
				require.NoError(t, err)
				return 20
			},
			req:      web.IdReq{},
			wantCode: 200,
			verify: func(t *testing.T, resp web.CompanyVO, id int64) {
				// 验证返回的数据是否正确
				require.Equal(t, id, resp.ID)
				require.Equal(t, "C端详情公司", resp.Name)

				// 可以验证更多字段，如创建时间等
				require.NotZero(t, resp.Ctime)

				// 验证数据库中的数据
				var company comdao.Company
				err := c.db.WithContext(context.Background()).Where("id = ?", id).First(&company).Error
				require.NoError(t, err)
				require.Equal(t, "C端详情公司", company.Name)
			},
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
			c.cServer.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)

			result := recorder.MustScan()
			if tc.verify != nil {
				tc.verify(t, result.Data, id)
			}
		})
	}
}

func (c *CompanyTestSuite) Test_GetCompaniesByIds() {
	testcases := []struct {
		name    string
		before  func(t *testing.T) []int64
		ids     []int64
		wantErr error
		verify  func(t *testing.T, companies map[int64]company.Company)
	}{
		{
			name: "通过多个ID获取公司",
			before: func(t *testing.T) []int64 {
				companies := []comdao.Company{
					{Id: 201, Name: "公司X", Ctime: 123, Utime: 123},
					{Id: 202, Name: "公司Y", Ctime: 123, Utime: 123},
					{Id: 203, Name: "公司Z", Ctime: 123, Utime: 123},
				}
				for _, company := range companies {
					err := c.db.WithContext(context.Background()).Create(&company).Error
					require.NoError(t, err)
				}
				return []int64{201, 202, 203}
			},
			ids:     []int64{201, 202, 203},
			wantErr: nil,
			verify: func(t *testing.T, companies map[int64]company.Company) {
				require.Equal(t, 3, len(companies), "应返回3个公司")

				// 验证每个公司的数据
				c1, ok := companies[201]
				require.True(t, ok, "应包含ID为201的公司")
				require.Equal(t, "公司X", c1.Name)

				c2, ok := companies[202]
				require.True(t, ok, "应包含ID为202的公司")
				require.Equal(t, "公司Y", c2.Name)

				c3, ok := companies[203]
				require.True(t, ok, "应包含ID为203的公司")
				require.Equal(t, "公司Z", c3.Name)
			},
		},
		{
			name: "部分ID不存在",
			before: func(t *testing.T) []int64 {
				companies := []comdao.Company{
					{Id: 301, Name: "公司甲", Ctime: 123, Utime: 123},
					{Id: 302, Name: "公司乙", Ctime: 123, Utime: 123},
				}
				for _, company := range companies {
					err := c.db.WithContext(context.Background()).Create(&company).Error
					require.NoError(t, err)
				}
				return []int64{301, 302}
			},
			ids:     []int64{301, 302, 999},
			wantErr: nil,
			verify: func(t *testing.T, companies map[int64]company.Company) {
				require.Equal(t, 2, len(companies), "应只返回存在的公司")

				// 验证返回的公司ID
				_, ok301 := companies[301]
				require.True(t, ok301, "应包含ID为301的公司")

				_, ok302 := companies[302]
				require.True(t, ok302, "应包含ID为302的公司")

				_, ok999 := companies[999]
				require.False(t, ok999, "不应包含不存在的ID为999的公司")

				// 验证公司名称
				require.Equal(t, "公司甲", companies[301].Name)
				require.Equal(t, "公司乙", companies[302].Name)
			},
		},
		{
			name:    "空ID列表",
			before:  func(t *testing.T) []int64 { return []int64{} },
			ids:     []int64{},
			wantErr: nil,
			verify: func(t *testing.T, companies map[int64]company.Company) {
				require.Equal(t, 0, len(companies), "空ID列表应返回空结果")
			},
		},
	}

	for _, tc := range testcases {
		c.T().Run(tc.name, func(t *testing.T) {
			// 准备测试数据
			if tc.before != nil {
				tc.before(t)
			}

			// 调用服务方法
			ctx := context.Background()
			companies, err := c.svc.GetByIds(ctx, tc.ids)

			// 验证错误
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			} else {
				require.NoError(t, err)
			}

			// 验证结果
			if tc.verify != nil {
				tc.verify(t, companies)
			}
		})
	}
}

func TestCompany(t *testing.T) {
	suite.Run(t, new(CompanyTestSuite))
}
