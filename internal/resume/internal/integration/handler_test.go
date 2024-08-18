//go:build e2e

package integration

import (
	"context"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/cases"
	casemocks "github.com/ecodeclub/webook/internal/cases/mocks"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/resume/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/resume/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/resume/internal/web"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type TestSuite struct {
	suite.Suite
	db     *egorm.Component
	server *egin.Component
	hdl    *web.Handler
}

func (t *TestSuite) TearDownTest() {
	err := t.db.Exec("TRUNCATE  TABLE `resume_project`").Error
	require.NoError(t.T(), err)
	err = t.db.Exec("TRUNCATE TABLE `contribution`").Error
	require.NoError(t.T(), err)
	err = t.db.Exec("TRUNCATE  TABLE `difficulty`").Error
	require.NoError(t.T(), err)
	err = t.db.Exec("TRUNCATE  TABLE `ref_case`").Error
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

	require.NoError(s.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid:  uid,
			Data: map[string]string{"creator": "true"},
		}))
	})
	handler.PrivateRoutes(server.Engine)
	s.server = server
	s.db = testioc.InitDB()
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewSkillDAO(s.db)
}
