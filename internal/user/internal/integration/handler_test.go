//go:build e2e

package integration

import (
	"net/http"
	"testing"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	test2 "github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ecodeclub/webook/internal/user/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/user/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/user/internal/web"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert/v2"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HandleTestSuite struct {
	suite.Suite
	db     *egorm.Component
	server *egin.Component
}

func (s *HandleTestSuite) SetupSuite() {
	econf.Set("http_users", map[string]any{})
	s.db = testioc.InitDB()
	err := dao.InitTables(s.db)
	require.NoError(s.T(), err)
	econf.Set("server", map[string]string{})
	server := egin.Load("server").Build()
	hdl := startup.InitHandler(nil, nil, nil)
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: 123,
		}))
	})
	hdl.PrivateRoutes(server.Engine)
	s.server = server
}

func (s *HandleTestSuite) TearDownSuite() {
	err := s.db.Exec("TRUNCATE table `users`").Error
	require.NoError(s.T(), err)
}

func (s *HandleTestSuite) TestEditProfile() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      EditReq
		wantResp test2.Result[any]
		wantCode int
	}{
		{
			name: "编辑成功",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.User{
					Id:       123,
					Nickname: "old name",
					Avatar:   "old avatar",
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				var u dao.User
				err := s.db.Where("id = ?", 123).First(&u).Error
				require.NoError(t, err)
				u.Ctime = 0
				u.Utime = 0
				assert.Equal(t, dao.User{
					Id:       123,
					Avatar:   "new avatar",
					Nickname: "new name",
				}, u)
			},
			req: EditReq{
				Avatar:   "new avatar",
				Nickname: "new name",
			},
			wantResp: test2.Result[any]{
				Msg: "OK",
			},
			wantCode: 200,
		},
		{
			name: "编辑成功-部分数据",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.User{
					Id:       123,
					Nickname: "old name",
					Avatar:   "old avatar",
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				var u dao.User
				err := s.db.Where("id = ?", 123).First(&u).Error
				require.NoError(t, err)
				u.Ctime = 0
				u.Utime = 0
				assert.Equal(t, dao.User{
					Id:       123,
					Avatar:   "old avatar",
					Nickname: "new name",
				}, u)
			},
			req: EditReq{
				Nickname: "new name",
			},
			wantResp: test2.Result[any]{
				Msg: "OK",
			},
			wantCode: 200,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/users/profile", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test2.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			assert.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
			// 清理掉 123 的数据
			s.db.Exec("TRUNCATE table `users`")
		})
	}
}

func (s *HandleTestSuite) TestProfile() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		wantResp test2.Result[web.Profile]
		wantCode int
	}{
		{
			name: "获得数据",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.User{
					Id:       123,
					Nickname: "old name",
					Avatar:   "old avatar",
				}).Error
				require.NoError(t, err)
			},
			wantResp: test2.Result[web.Profile]{
				Data: web.Profile{
					Nickname: "old name",
					Avatar:   "old avatar",
				},
			},
			wantCode: 200,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodGet,
				"/users/profile", nil)
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test2.NewJSONResponseRecorder[web.Profile]()
			s.server.ServeHTTP(recorder, req)
			assert.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func TestUserHandler(t *testing.T) {
	suite.Run(t, new(HandleTestSuite))
}

type EditReq struct {
	Avatar   string `json:"avatar"`
	Nickname string `json:"nickname"`
}
