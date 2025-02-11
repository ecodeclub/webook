//go:build e2e

package integration

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/ecodeclub/webook/internal/member"

	"github.com/ecodeclub/webook/internal/ai"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/ginx/session"
	eveMocks "github.com/ecodeclub/webook/internal/cases/internal/event/mocks"
	"github.com/ecodeclub/webook/internal/cases/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/cases/internal/web"
	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"
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
	"go.uber.org/mock/gomock"
)

type AdminCaseSetTestSuite struct {
	suite.Suite
	server   *egin.Component
	db       *egorm.Component
	dao      dao.CaseSetDAO
	caseDao  dao.CaseDAO
	ctrl     *gomock.Controller
	producer *eveMocks.MockSyncEventProducer
}

func (s *AdminCaseSetTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())
	s.producer = eveMocks.NewMockSyncEventProducer(s.ctrl)
	intrSvc := intrmocks.NewMockService(s.ctrl)
	intrModule := &interactive.Module{
		Svc: intrSvc,
	}
	module, err := startup.InitModule(s.producer,
		nil, &ai.Module{}, &member.Module{},
		session.DefaultProvider(),
		intrModule)
	require.NoError(s.T(), err)
	adminHandler := module.AdminSetHandler
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
	adminHandler.PrivateRoutes(server.Engine)
	s.server = server
	server.Use(middleware.NewCheckMembershipMiddlewareBuilder(nil).Build())
	s.db = testioc.InitDB()
	s.dao = dao.NewCaseSetDAO(s.db)
	s.caseDao = dao.NewCaseDao(s.db)
}

func (s *AdminCaseSetTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `case_sets`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `case_set_cases`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `cases`").Error
	require.NoError(s.T(), err)
}

func (s *AdminCaseSetTestSuite) TestSave() {
	testcases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T, id int64)
		req      web.CaseSet
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "保存",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T, id int64) {
				set, err := s.dao.GetByID(context.Background(), id)
				require.NoError(s.T(), err)
				assert.True(t, set.Ctime != 0)
				assert.True(t, set.Utime != 0)
				set.Ctime = 0
				set.Utime = 0
				assert.Equal(t, dao.CaseSet{
					Id:          1,
					Uid:         uid,
					Title:       "test title",
					Description: "test description",
					Biz:         "baguwen",
					BizId:       22,
				}, set)

			},
			req: web.CaseSet{

				Title:       "test title",
				Description: "test description",
				Biz:         "baguwen",
				BizId:       22,
			},
			wantCode: 200,
		},
		{
			name: "编辑",
			before: func(t *testing.T) {
				_, err := s.dao.Create(context.Background(), dao.CaseSet{
					Id:          1,
					Uid:         uid,
					Title:       "test title",
					Description: "test description",
					Biz:         "baguwen",
					BizId:       23,
					Ctime:       123,
					Utime:       234,
				})
				require.NoError(s.T(), err)

			},
			after: func(t *testing.T, id int64) {
				set, err := s.dao.GetByID(context.Background(), id)
				require.NoError(s.T(), err)
				assert.True(t, set.Ctime != 0)
				assert.True(t, set.Utime != 0)
				set.Ctime = 0
				set.Utime = 0
				assert.Equal(t, dao.CaseSet{
					Id:          1,
					Uid:         uid,
					Title:       "new title",
					Description: "new description",
					Biz:         "jijibo",
					BizId:       66,
				}, set)

			},
			req: web.CaseSet{
				Id:          1,
				Title:       "new title",
				Description: "new description",
				Biz:         "jijibo",
				BizId:       66,
			},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/case-sets/save", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t, recorder.MustScan().Data)
			// 清理掉 123 的数据
			err = s.db.Exec("TRUNCATE table `case_sets`").Error
			require.NoError(t, err)

		})
	}
}

func (s *AdminCaseSetTestSuite) Test_UpdateCases() {
	testcases := []struct {
		name     string
		before   func(t *testing.T) int64
		after    func(t *testing.T, id int64)
		req      web.UpdateCases
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "空案例集_添加多个案例",
			before: func(t *testing.T) int64 {
				id, err := s.dao.Create(context.Background(), dao.CaseSet{
					Uid:         uid,
					Title:       "test title",
					Description: "test description",
					Biz:         "baguwen",
					BizId:       23,
					Ctime:       123,
					Utime:       234,
				})
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(1))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(2))
				require.NoError(t, err)
				return id
			},
			req: web.UpdateCases{
				CIDs: []int64{1, 2},
			},
			after: func(t *testing.T, id int64) {
				cases, err := s.dao.GetCasesByID(context.Background(), id)
				require.NoError(t, err)
				for idx, ca := range cases {
					require.True(t, ca.Ctime != 0)
					require.True(t, ca.Utime != 0)
					ca.Ctime = 0
					ca.Utime = 0
					cases[idx] = ca
				}
				assert.Equal(t, []dao.Case{
					getTestCase(1),
					getTestCase(2),
				}, cases)
			},
			wantCode: 200,
		},
		{
			name: "非空案例集_添加多个案例",
			before: func(t *testing.T) int64 {
				id, err := s.dao.Create(context.Background(), dao.CaseSet{
					Uid:         uid,
					Title:       "test title",
					Description: "test description",
					Biz:         "baguwen",
					BizId:       23,
					Ctime:       123,
					Utime:       234,
				})
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(1))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(2))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(3))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(4))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(5))
				require.NoError(t, err)
				err = s.dao.UpdateCasesByID(context.Background(), id, []int64{1, 2})
				require.NoError(t, err)
				return id
			},
			req: web.UpdateCases{
				CIDs: []int64{1, 2, 3, 4, 5},
			},
			wantCode: 200,
			after: func(t *testing.T, id int64) {
				cases, err := s.dao.GetCasesByID(context.Background(), id)
				require.NoError(t, err)
				for idx, ca := range cases {
					require.True(t, ca.Ctime != 0)
					require.True(t, ca.Utime != 0)
					ca.Ctime = 0
					ca.Utime = 0
					cases[idx] = ca
				}
				assert.Equal(t, []dao.Case{
					getTestCase(1),
					getTestCase(2),
					getTestCase(3),
					getTestCase(4),
					getTestCase(5),
				}, cases)
			},
		},
		{
			name: "删除部分案例",
			before: func(t *testing.T) int64 {
				id, err := s.dao.Create(context.Background(), dao.CaseSet{
					Uid:         uid,
					Title:       "test title",
					Description: "test description",
					Biz:         "baguwen",
					BizId:       23,
					Ctime:       123,
					Utime:       234,
				})
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(1))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(2))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(3))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(4))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(5))
				require.NoError(t, err)
				err = s.dao.UpdateCasesByID(context.Background(), id, []int64{1, 2, 3, 4, 5})
				require.NoError(t, err)
				return id
			},
			req: web.UpdateCases{
				CIDs: []int64{1, 2, 3},
			},
			wantCode: 200,
			after: func(t *testing.T, id int64) {
				cases, err := s.dao.GetCasesByID(context.Background(), id)
				require.NoError(t, err)
				for idx, ca := range cases {
					require.True(t, ca.Ctime != 0)
					require.True(t, ca.Utime != 0)
					ca.Ctime = 0
					ca.Utime = 0
					cases[idx] = ca
				}
				assert.Equal(t, []dao.Case{
					getTestCase(1),
					getTestCase(2),
					getTestCase(3),
				}, cases)
			},
		},
		{
			name: "删除全部案例",
			before: func(t *testing.T) int64 {
				id, err := s.dao.Create(context.Background(), dao.CaseSet{
					Uid:         uid,
					Title:       "test title",
					Description: "test description",
					Biz:         "baguwen",
					BizId:       23,
					Ctime:       123,
					Utime:       234,
				})
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(1))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(2))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(3))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(4))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(5))
				require.NoError(t, err)
				err = s.dao.UpdateCasesByID(context.Background(), id, []int64{1, 2, 3, 4, 5})
				require.NoError(t, err)
				return id
			},
			req: web.UpdateCases{
				CIDs: []int64{},
			},
			wantCode: 200,
			after: func(t *testing.T, id int64) {
				cases, err := s.dao.GetCasesByID(context.Background(), id)
				require.NoError(t, err)
				for idx, ca := range cases {
					require.True(t, ca.Ctime != 0)
					require.True(t, ca.Utime != 0)
					ca.Ctime = 0
					ca.Utime = 0
					cases[idx] = ca
				}
				assert.Equal(t, 0, len(cases))
			},
		},
		{
			name: "同时添加/删除部分案例",
			before: func(t *testing.T) int64 {
				id, err := s.dao.Create(context.Background(), dao.CaseSet{
					Uid:         uid,
					Title:       "test title",
					Description: "test description",
					Biz:         "baguwen",
					BizId:       23,
					Ctime:       123,
					Utime:       234,
				})
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(1))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(2))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(3))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(4))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(5))
				require.NoError(t, err)
				err = s.dao.UpdateCasesByID(context.Background(), id, []int64{1, 2, 3})
				require.NoError(t, err)
				return id
			},
			req: web.UpdateCases{
				CIDs: []int64{1, 2, 4},
			},
			wantCode: 200,
			after: func(t *testing.T, id int64) {
				cases, err := s.dao.GetCasesByID(context.Background(), id)
				require.NoError(t, err)
				for idx, ca := range cases {
					require.True(t, ca.Ctime != 0)
					require.True(t, ca.Utime != 0)
					ca.Ctime = 0
					ca.Utime = 0
					cases[idx] = ca
				}
				assert.Equal(t, []dao.Case{
					getTestCase(1),
					getTestCase(2),
					getTestCase(4),
				}, cases)
			},
		},
		{
			name: "案例集不存在",
			before: func(t *testing.T) int64 {
				return 3
			},
			after: func(t *testing.T, id int64) {
			},
			req: web.UpdateCases{
				CIDs: []int64{1, 2, 3},
			},
			wantCode: 500,
			wantResp: test.Result[int64]{Code: 502001, Msg: "系统错误"},
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			id := tc.before(t)
			tc.req.CSID = id
			req, err := http.NewRequest(http.MethodPost,
				"/case-sets/cases/save", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t, id)
			// 清理掉 123 的数据
			err = s.db.Exec("TRUNCATE table `case_sets`").Error
			err = s.db.Exec("TRUNCATE table `case_set_cases`").Error
			require.NoError(t, err)

		})
	}
}

func (s *AdminCaseSetTestSuite) Test_List() {
	for i := 1; i < 20; i++ {
		_, err := s.dao.Create(context.Background(), getTestCaseSet(int64(i)))
		require.NoError(s.T(), err)
	}
	testcases := []struct {
		name     string
		after    func(t *testing.T, id int64)
		req      web.Page
		wantCode int
		wantResp web.CaseSetList
	}{
		{
			name: "列表",
			req: web.Page{
				Offset: 0,
				Limit:  2,
			},
			wantCode: 200,
			wantResp: web.CaseSetList{
				Total: 19,
				CaseSets: []web.CaseSet{
					{
						Id:          19,
						Title:       "title19",
						Description: "description19",
						Biz:         "baguwen",
						BizId:       49,
					},
					{
						Id:          18,
						Title:       "title18",
						Description: "description18",
						Biz:         "baguwen",
						BizId:       48,
					},
				},
			},
		},
		{
			name: "列表--分页",
			req: web.Page{
				Offset: 2,
				Limit:  2,
			},
			wantCode: 200,
			wantResp: web.CaseSetList{
				Total: 19,
				CaseSets: []web.CaseSet{
					{
						Id:          17,
						Title:       "title17",
						Description: "description17",
						Biz:         "baguwen",
						BizId:       47,
					},
					{
						Id:          16,
						Title:       "title16",
						Description: "description16",
						Biz:         "baguwen",
						BizId:       46,
					},
				},
			},
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/case-sets/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.CaseSetList]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			res := recorder.MustScan().Data
			for idx, ca := range res.CaseSets {
				require.True(t, ca.Utime != 0)
				ca.Utime = 0
				res.CaseSets[idx] = ca
			}
			assert.Equal(t, tc.wantResp, res)
		})
	}
}

func (s *AdminCaseSetTestSuite) Test_Detail() {
	testcases := []struct {
		name     string
		before   func(t *testing.T) int64
		wantCode int
		wantResp web.CaseSet
	}{
		{
			name: "题集详情",
			before: func(t *testing.T) int64 {
				id, err := s.dao.Create(context.Background(), dao.CaseSet{
					Uid:         uid,
					Title:       "test title",
					Description: "test description",
					Biz:         "baguwen",
					BizId:       23,
					Ctime:       123,
					Utime:       234,
				})
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(1))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(2))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(3))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(4))
				require.NoError(t, err)
				_, err = s.caseDao.Save(context.Background(), getTestCase(5))
				require.NoError(t, err)
				err = s.dao.UpdateCasesByID(context.Background(), id, []int64{1, 2, 3, 4, 5})
				require.NoError(t, err)
				return id
			},
			wantCode: 200,
			wantResp: web.CaseSet{
				Title:       "test title",
				Description: "test description",
				Biz:         "baguwen",
				Cases: []web.Case{
					getCase(1),
					getCase(2),
					getCase(3),
					getCase(4),
					getCase(5),
				},
				BizId: 23,
			},
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			id := tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/case-sets/detail", iox.NewJSONReader(web.CaseSetID{
					ID: id,
				}))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.CaseSet]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			res := recorder.MustScan().Data
			for idx, ca := range res.Cases {
				require.True(t, ca.Utime != 0)
				ca.Utime = 0
				res.Cases[idx] = ca
			}
			tc.wantResp.Id = id
			require.True(t, res.Utime != 0)
			res.Utime = 0
			assert.Equal(t, tc.wantResp, res)
		})
	}
}

func (s *AdminCaseSetTestSuite) TestQuestionSet_Candidates() {
	testCases := []struct {
		name string

		before func(t *testing.T)
		req    web.CandidateReq

		wantCode int
		wantResp test.Result[web.CasesList]
	}{
		{
			name: "获取成功",
			before: func(t *testing.T) {
				// 准备数据
				// 创建一个空案例集
				id, err := s.dao.Create(context.Background(), dao.CaseSet{
					Id:          1,
					Uid:         uid,
					Title:       "Go",
					Description: "Go题集",
					Biz:         "roadmap",
					BizId:       2,
					Utime:       123,
				})
				require.NoError(t, err)
				// 添加案例
				cases := []dao.Case{
					getTestCase(1),
					getTestCase(2),
					getTestCase(3),
					getTestCase(4),
					getTestCase(5),
					getTestCase(6),
				}
				err = s.db.WithContext(context.Background()).Create(&cases).Error
				require.NoError(t, err)
				cids := []int64{1, 2, 3}
				require.NoError(t, s.dao.UpdateCasesByID(context.Background(), id, cids))
			},
			req: web.CandidateReq{
				CSID:   1,
				Offset: 1,
				Limit:  2,
			},
			wantCode: 200,
			wantResp: test.Result[web.CasesList]{
				Data: web.CasesList{
					Total: 3,
					Cases: []web.Case{
						getCase(5),
						getCase(4),
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/case-sets/candidate", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.CasesList]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func TestCaseSetAdminHandler(t *testing.T) {
	suite.Run(t, new(AdminCaseSetTestSuite))
}

func getTestCase(id int64) dao.Case {
	return dao.Case{
		Id:           id,
		Uid:          uid,
		Introduction: fmt.Sprintf("intr%d", id),
		Title:        fmt.Sprintf("title%d", id),
		Content:      fmt.Sprintf("content%d", id),
		Labels: sqlx.JsonColumn[[]string]{
			Valid: true,
			Val:   []string{"case", "mysql"},
		},
		GithubRepo: fmt.Sprintf("githubrepo%d", id),
		GiteeRepo:  fmt.Sprintf("giteerepo%d", id),
		Keywords:   fmt.Sprintf("keywords%d", id),
		Shorthand:  fmt.Sprintf("shorthand%d", id),
		Highlight:  fmt.Sprintf("highlight%d", id),
		Guidance:   fmt.Sprintf("guidance%d", id),
		Biz:        "question",
		BizId:      11,
		Status:     2,
	}
}

func getTestCaseSet(id int64) dao.CaseSet {
	return dao.CaseSet{
		Id:          id,
		Uid:         uid,
		Title:       fmt.Sprintf("title%d", id),
		Description: fmt.Sprintf("description%d", id),
		Biz:         "baguwen",
		BizId:       id + 30,
	}
}

func getCase(id int64) web.Case {
	ca := web.Case{
		Id:           id,
		Introduction: fmt.Sprintf("intr%d", id),
		Title:        fmt.Sprintf("title%d", id),
		Content:      fmt.Sprintf("content%d", id),
		Labels:       []string{"case", "mysql"},
		GithubRepo:   fmt.Sprintf("githubrepo%d", id),
		GiteeRepo:    fmt.Sprintf("giteerepo%d", id),
		Keywords:     fmt.Sprintf("keywords%d", id),
		Shorthand:    fmt.Sprintf("shorthand%d", id),
		Highlight:    fmt.Sprintf("highlight%d", id),
		Guidance:     fmt.Sprintf("guidance%d", id),
		Biz:          "question",
		BizId:        11,
		Status:       2,
	}
	return ca
}
