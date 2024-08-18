//go:build e2e

package integration

import (
	"context"
	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/cases"
	casemocks "github.com/ecodeclub/webook/internal/cases/mocks"
	"github.com/ecodeclub/webook/internal/resume/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/resume/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/resume/internal/web"
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
	"net/http"
	"testing"
)

const uid = 123

type TestSuite struct {
	suite.Suite
	db     *egorm.Component
	server *egin.Component
	hdl    *web.Handler
	pdao   dao.ResumeProjectDAO
}

func (t *TestSuite) TearDownTest() {
	err := t.db.Exec("TRUNCATE  TABLE `resume_projects`").Error
	require.NoError(t.T(), err)
	err = t.db.Exec("TRUNCATE TABLE `contributions`").Error
	require.NoError(t.T(), err)
	err = t.db.Exec("TRUNCATE  TABLE `difficulties`").Error
	require.NoError(t.T(), err)
	err = t.db.Exec("TRUNCATE  TABLE `ref_cases`").Error
	require.NoError(t.T(), err)
}

func (s *TestSuite) SetupSuite() {
	ctrl := gomock.NewController(s.T())
	examSvc := casemocks.NewMockExamineService(ctrl)
	examSvc.EXPECT().GetResults(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, uid int64, ids []int64) (map[int64]cases.ExamineCaseResult, error) {
		res := slice.Map(ids, func(idx int, src int64) cases.ExamineCaseResult {
			return cases.ExamineCaseResult{
				Cid:    src,
				Result: cases.CaseResult(src % 4),
			}
		})
		resMap := make(map[int64]cases.ExamineCaseResult, len(res))
		for _, examRes := range res {
			resMap[examRes.Cid] = examRes
		}
		return resMap, nil
	}).AnyTimes()

	module := startup.InitModule(&cases.Module{
		ExamService: examSvc,
	})
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid:  uid,
			Data: map[string]string{"creator": "true"},
		}))
	})
	module.Hdl.PrivateRoutes(server.Engine)
	s.server = server
	s.db = testioc.InitDB()
	err := dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.pdao = dao.NewResumeProjectDAO(s.db)
}

func TestResumeModule(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestSaveResumeProject() {
	testcases := []struct {
		name     string
		req      web.SaveProjectReq
		before   func(t *testing.T)
		after    func(t *testing.T)
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "新建",
			req: web.SaveProjectReq{
				Project: web.Project{
					StartTime:    1,
					EndTime:      321,
					Name:         "project",
					Introduction: "introduction",
					Core:         true,
				},
			},
			before: func(t *testing.T) {
			},
			after: func(t *testing.T) {
				project, err := s.pdao.First(context.Background(), 1)
				require.NoError(t, err)
				require.True(t, project.Ctime != 0)
				require.True(t, project.Utime != 0)
				project.Ctime = 0
				project.Utime = 0
				assert.Equal(t, dao.ResumeProject{
					ID:           1,
					StartTime:    1,
					EndTime:      321,
					Uid:          uid,
					Name:         "project",
					Introduction: "introduction",
					Core:         true,
				}, project)
			},
			wantResp: test.Result[int64]{
				Data: 1,
			},
			wantCode: 200,
		},
		{
			name: "更新",
			req: web.SaveProjectReq{
				Project: web.Project{
					Id:           1,
					StartTime:    2,
					EndTime:      666,
					Uid:          uid,
					Name:         "projectnew",
					Introduction: "introductionnew",
					Core:         false,
				},
			},
			before: func(t *testing.T) {
				_, err := s.pdao.Upsert(context.Background(), dao.ResumeProject{
					StartTime:    1,
					Uid:          uid,
					EndTime:      321,
					Name:         "project",
					Introduction: "introduction",
					Core:         true,
					Utime:        1,
					Ctime:        2,
				})
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				project, err := s.pdao.First(context.Background(), 1)
				require.NoError(t, err)
				require.True(t, project.Ctime != 0)
				require.True(t, project.Utime != 0)
				project.Ctime = 0
				project.Utime = 0
				assert.Equal(t, dao.ResumeProject{
					ID:           1,
					StartTime:    2,
					EndTime:      666,
					Uid:          uid,
					Name:         "projectnew",
					Introduction: "introductionnew",
					Core:         false,
				}, project)
			},
			wantResp: test.Result[int64]{
				Data: 1,
			},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/resume/project/save", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
			// 清理数据
			err = s.db.Exec("TRUNCATE  TABLE `resume_projects`").Error
			require.NoError(t, err)
		})
	}
}
