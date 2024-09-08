package integration

import (
	"context"
	"testing"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/cases"
	casemocks "github.com/ecodeclub/webook/internal/cases/mocks"
	"github.com/ecodeclub/webook/internal/resume/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/resume/internal/repository/dao"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const uid = 1235678

type ExperienceTestSuite struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
	dao    dao.ExperienceDAO
	ctrl   *gomock.Controller
}

func (s *ExperienceTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `experiences`").Error
	require.NoError(s.T(), err)
}

func (s *ExperienceTestSuite) SetupSuite() {
	ctrl := gomock.NewController(s.T())
	examSvc := casemocks.NewMockExamineService(ctrl)
	examSvc.EXPECT().GetResults(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, uid int64, ids []int64) (map[int64]cases.ExamineResult, error) {
		res := slice.Map(ids, func(idx int, src int64) cases.ExamineResult {
			return cases.ExamineResult{
				Cid:    src,
				Result: cases.ExamineResultEnum(src % 4),
			}
		})
		resMap := make(map[int64]cases.ExamineResult, len(res))
		for _, examRes := range res {
			resMap[examRes.Cid] = examRes
		}
		return resMap, nil
	}).AnyTimes()

	module := startup.InitModule(&cases.Module{
		ExamineSvc: examSvc,
	})
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()

	server.Use(func(ctx *gin.Context) {
		ctx.Set(session.CtxSessionKey,
			session.NewMemorySession(session.Claims{
				Uid: uid,
			}))
	})

	module.ExperienceHdl.PrivateRoutes(server.Engine)

	s.db = testioc.InitDB()
	err := dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewExperienceDAO(s.db)
}

func TestExperienceModule(t *testing.T) {
	suite.Run(t, new(ExperienceTestSuite))
}
