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

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
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

type CaseSetTestSuite struct {
	suite.Suite
	server   *egin.Component
	db       *egorm.Component
	dao      dao.CaseSetDAO
	caseDao  dao.CaseDAO
	ctrl     *gomock.Controller
	producer *eveMocks.MockSyncEventProducer
}

func (s *CaseSetTestSuite) SetupSuite() {
	ctrl := gomock.NewController(s.T())
	s.producer = eveMocks.NewMockSyncEventProducer(ctrl)

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

	module, err := startup.InitExamModule(s.producer, nil, intrModule, &member.Module{},
		session.DefaultProvider(),
		&ai.Module{})
	require.NoError(s.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()

	module.CsHdl.PublicRoutes(server.Engine)
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: uid,
			Data: map[string]string{
				"creator":   "true",
				"memberDDL": strconv.FormatInt(time.Now().Add(time.Hour).UnixMilli(), 10),
			},
		}))
	})
	module.CsHdl.PrivateRoutes(server.Engine)
	server.Use(middleware.NewCheckMembershipMiddlewareBuilder(nil).Build())

	s.server = server
	s.db = testioc.InitDB()
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewCaseSetDAO(s.db)
	s.caseDao = dao.NewCaseDao(s.db)
}

func (s *CaseSetTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `case_sets`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `case_set_cases`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `cases`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `case_examine_records`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `case_results`").Error
	require.NoError(s.T(), err)
}

func (s *CaseSetTestSuite) TestCaseSetDetailByBiz() {
	var now int64 = 123
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)
		req    web.BizReq

		wantCode int
		wantResp test.Result[web.CaseSet]
	}{
		{
			name: "空案例集",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				// 创建一个空题集
				id, err := s.dao.Create(ctx, dao.CaseSet{
					Id:          321,
					Uid:         uid,
					Title:       "Go",
					Biz:         "roadmap",
					BizId:       2,
					Description: "Go的desc",
				})
				require.NoError(t, err)
				require.Equal(t, int64(321), id)
			},
			after: func(t *testing.T) {
			},
			req: web.BizReq{
				Biz:   "roadmap",
				BizId: 2,
			},
			wantCode: 200,
			wantResp: test.Result[web.CaseSet]{
				Data: web.CaseSet{
					Id:          321,
					Title:       "Go",
					Description: "Go的desc",
					Biz:         "roadmap",
					BizId:       2,
					Interactive: web.Interactive{
						ViewCnt:    322,
						LikeCnt:    323,
						CollectCnt: 324,
						Liked:      true,
						Collected:  false,
					},
				},
			},
		},
		{
			name: "非空案例集",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				id, err := s.dao.Create(ctx, dao.CaseSet{
					Id:          322,
					Uid:         uid,
					Title:       "Go",
					Description: "Go案例集",
					Biz:         "roadmap",
					BizId:       3,
				})
				require.NoError(t, err)
				require.Equal(t, int64(322), id)

				// 添加案例
				cases := []dao.Case{
					{
						Id:      614,
						Uid:     uid + 1,
						Biz:     "project",
						BizId:   1,
						Title:   "Go案例1",
						Content: "Go案例1",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:      615,
						Uid:     uid + 2,
						Biz:     "project",
						BizId:   1,
						Title:   "Go案例2",
						Content: "Go案例2",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:      616,
						Uid:     uid + 3,
						Biz:     "project",
						BizId:   1,
						Title:   "Go案例3",
						Content: "Go案例3",
						Ctime:   now,
						Utime:   now,
					},
				}
				err = s.db.WithContext(ctx).Create(&cases).Error
				require.NoError(t, err)
				cids := []int64{614, 615, 616}
				require.NoError(t, s.dao.UpdateCasesByID(ctx, id, cids))

				// 添加用户答题记录，只需要添加一个就可以
				err = s.db.WithContext(ctx).Create(&dao.CaseResult{
					Uid:    uid,
					Cid:    614,
					Result: domain.ResultPassed.ToUint8(),
					Ctime:  now,
					Utime:  now,
				}).Error
				require.NoError(t, err)

				// 题集中题目为1
				cs, err := s.dao.GetCasesByID(ctx, id)
				require.NoError(t, err)
				require.Equal(t, len(cids), len(cs))
			},
			after: func(t *testing.T) {
			},
			req: web.BizReq{
				Biz:   "roadmap",
				BizId: 3,
			},
			wantCode: 200,
			wantResp: test.Result[web.CaseSet]{
				Data: web.CaseSet{
					Id:          322,
					Biz:         "roadmap",
					BizId:       3,
					Title:       "Go",
					Description: "Go案例集",
					Interactive: web.Interactive{
						ViewCnt:    323,
						LikeCnt:    324,
						CollectCnt: 325,
						Liked:      false,
						Collected:  true,
					},
					Cases: []web.Case{
						{
							Id:            614,
							Biz:           "project",
							BizId:         1,
							Title:         "Go案例1",
							Content:       "Go案例1",
							ExamineResult: domain.ResultPassed.ToUint8(),
							Utime:         now,
							Interactive: web.Interactive{
								ViewCnt:    615,
								LikeCnt:    616,
								CollectCnt: 617,
								Liked:      false,
								Collected:  true,
							},
						},
						{
							Id:      615,
							Biz:     "project",
							BizId:   1,
							Title:   "Go案例2",
							Content: "Go案例2",
							Utime:   now,
							Interactive: web.Interactive{
								ViewCnt:    616,
								LikeCnt:    617,
								CollectCnt: 618,
								Liked:      true,
								Collected:  false,
							},
						},
						{
							Id:      616,
							Biz:     "project",
							BizId:   1,
							Title:   "Go案例3",
							Content: "Go案例3",
							Utime:   now,
							Interactive: web.Interactive{
								ViewCnt:    617,
								LikeCnt:    618,
								CollectCnt: 619,
								Liked:      false,
								Collected:  true,
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
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/case-sets/detail/biz", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.CaseSet]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			data := recorder.MustScan()
			assert.True(t, data.Data.Utime != 0)
			data.Data.Utime = 0
			assert.Equal(t, tc.wantResp, data)
			tc.after(t)
		})
	}
}

func (s *CaseSetTestSuite) TestCaseSet_Detail() {
	var now int64 = 123
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)
		req    web.CaseSetID

		wantCode int
		wantResp test.Result[web.CaseSet]
	}{
		{
			name: "空案例集",
			before: func(t *testing.T) {
				t.Helper()

				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				// 创建一个空案例集
				id, err := s.dao.Create(ctx, dao.CaseSet{
					Id:          321,
					Uid:         uid,
					Title:       "Go",
					Biz:         "roadmap",
					BizId:       2,
					Description: "Go案例集",
					Utime:       now,
				})
				require.NoError(t, err)
				require.Equal(t, int64(321), id)
			},
			after: func(t *testing.T) {
			},
			req: web.CaseSetID{
				ID: 321,
			},
			wantCode: 200,
			wantResp: test.Result[web.CaseSet]{
				Data: web.CaseSet{
					Id:          321,
					Title:       "Go",
					Description: "Go案例集",
					Biz:         "roadmap",
					BizId:       2,
					Interactive: web.Interactive{
						ViewCnt:    322,
						LikeCnt:    323,
						CollectCnt: 324,
						Liked:      true,
						Collected:  false,
					},
				},
			},
		},
		{
			name: "非空案例集",
			before: func(t *testing.T) {
				t.Helper()

				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				// 创建一个空题集
				id, err := s.dao.Create(ctx, dao.CaseSet{
					Id:          322,
					Uid:         uid,
					Title:       "Go",
					Description: "Go案例集",
					Biz:         "roadmap",
					BizId:       2,
				})
				require.NoError(t, err)
				require.Equal(t, int64(322), id)

				// 添加问题
				questions := []dao.Case{
					{
						Id:      614,
						Uid:     uid + 1,
						Biz:     "project",
						BizId:   1,
						Title:   "Go案例1",
						Content: "Go案例1",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:      615,
						Uid:     uid + 2,
						Biz:     "project",
						BizId:   1,
						Title:   "Go案例2",
						Content: "Go案例2",
						Ctime:   now,
						Utime:   now,
					},
					{
						Id:      616,
						Uid:     uid + 3,
						Biz:     "project",
						BizId:   1,
						Title:   "Go案例3",
						Content: "Go案例3",
						Ctime:   now,
						Utime:   now,
					},
				}
				err = s.db.WithContext(ctx).Create(&questions).Error
				require.NoError(t, err)
				cids := []int64{614, 615, 616}
				require.NoError(t, s.dao.UpdateCasesByID(ctx, id, cids))

				// 添加用户答题记录，只需要添加一个就可以
				err = s.db.WithContext(ctx).Create(&dao.CaseResult{
					Uid:    uid,
					Cid:    614,
					Result: domain.ResultPassed.ToUint8(),
					Ctime:  now,
					Utime:  now,
				}).Error
				require.NoError(t, err)

				// 题集中题目为1
				qs, err := s.dao.GetCasesByID(ctx, id)
				require.NoError(t, err)
				require.Equal(t, len(cids), len(qs))
			},
			after: func(t *testing.T) {
			},
			req: web.CaseSetID{
				ID: 322,
			},
			wantCode: 200,
			wantResp: test.Result[web.CaseSet]{
				Data: web.CaseSet{
					Id:          322,
					Biz:         "roadmap",
					BizId:       2,
					Title:       "Go",
					Description: "Go案例集",
					Interactive: web.Interactive{
						ViewCnt:    323,
						LikeCnt:    324,
						CollectCnt: 325,
						Liked:      false,
						Collected:  true,
					},
					Cases: []web.Case{
						{
							Id:            614,
							Biz:           "project",
							BizId:         1,
							Title:         "Go案例1",
							Content:       "Go案例1",
							ExamineResult: domain.ResultPassed.ToUint8(),
							Utime:         now,
							Interactive: web.Interactive{
								ViewCnt:    615,
								LikeCnt:    616,
								CollectCnt: 617,
								Liked:      false,
								Collected:  true,
							},
						},
						{
							Id:      615,
							Biz:     "project",
							BizId:   1,
							Title:   "Go案例2",
							Content: "Go案例2",
							Utime:   now,
							Interactive: web.Interactive{
								ViewCnt:    616,
								LikeCnt:    617,
								CollectCnt: 618,
								Liked:      true,
								Collected:  false,
							},
						},
						{
							Id:      616,
							Biz:     "project",
							BizId:   1,
							Title:   "Go案例3",
							Content: "Go案例3",
							Utime:   now,
							Interactive: web.Interactive{
								ViewCnt:    617,
								LikeCnt:    618,
								CollectCnt: 619,
								Liked:      false,
								Collected:  true,
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
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/case-sets/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.CaseSet]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			data := recorder.MustScan()
			assert.True(t, data.Data.Utime != 0)
			data.Data.Utime = 0
			assert.Equal(t, tc.wantResp, data)
			tc.after(t)
		})
	}
}

func (s *CaseSetTestSuite) TestCaseSet_ListAllCaseSets() {
	// 插入一百条
	total := 100
	data := make([]dao.CaseSet, 0, total)

	for idx := 0; idx < total; idx++ {
		// 空题集
		data = append(data, dao.CaseSet{
			Uid:         int64(uid + idx),
			Title:       fmt.Sprintf("案例集标题 %d", idx),
			Description: fmt.Sprintf("案例集简介 %d", idx),
			Utime:       123,
		})
	}
	// 这个接口是不会查询到这些数据的
	data = append(data, dao.CaseSet{
		Uid:         200,
		Title:       fmt.Sprintf("案例集标题 %d", 200),
		Description: fmt.Sprintf("案例集简介 %d", 200),
		Biz:         "project",
		BizId:       200,
		Utime:       123,
	})
	err := s.db.Create(&data).Error
	require.NoError(s.T(), err)

	testCases := []struct {
		name string
		req  web.Page

		wantCode int
		wantResp test.Result[web.CaseSetList]
	}{
		{
			name: "获取成功",
			req: web.Page{
				Limit:  2,
				Offset: 0,
			},
			wantCode: 200,
			wantResp: test.Result[web.CaseSetList]{
				Data: web.CaseSetList{
					Total: 100,
					CaseSets: []web.CaseSet{
						{
							Id:          100,
							Title:       "案例集标题 99",
							Description: "案例集简介 99",
							Biz:         "baguwen",
							Utime:       123,
							Interactive: web.Interactive{
								ViewCnt:    101,
								LikeCnt:    102,
								CollectCnt: 103,
								Liked:      false,
								Collected:  true,
							},
						},
						{
							Id:          99,
							Title:       "案例集标题 98",
							Description: "案例集简介 98",
							Biz:         "baguwen",
							Utime:       123,
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
		},
		{
			name: "获取部分",
			req: web.Page{
				Limit:  2,
				Offset: 99,
			},
			wantCode: 200,
			wantResp: test.Result[web.CaseSetList]{
				Data: web.CaseSetList{
					Total: 100,
					CaseSets: []web.CaseSet{
						{
							Id:          1,
							Title:       "案例集标题 0",
							Description: "案例集简介 0",
							Biz:         "baguwen",
							Utime:       123,
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
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/case-sets/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.CaseSetList]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *CaseSetTestSuite) TestQuestionSet_RetrieveQuestionSetDetail_Failed() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)
		req    web.CaseSetID

		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "题集ID非法_题集ID不存在",
			before: func(t *testing.T) {
				t.Helper()
			},
			after: func(t *testing.T) {
				t.Helper()
			},
			req: web.CaseSetID{
				ID: 10000,
			},
			wantCode: 500,
			wantResp: test.Result[int64]{Code: 505001, Msg: "系统错误"},
		},
	}
	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/case-sets/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
		})
	}
}

func (s *CaseSetTestSuite) mockInteractive(biz string, id int64) interactive.Interactive {
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

func TestCaseSetHandler(t *testing.T) {
	suite.Run(t, new(CaseSetTestSuite))
}
