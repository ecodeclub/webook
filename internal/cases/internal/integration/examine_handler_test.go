//go:build e2e

package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ecodeclub/webook/internal/member"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/ai"
	aimocks "github.com/ecodeclub/webook/internal/ai/mocks"
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	eveMocks "github.com/ecodeclub/webook/internal/cases/internal/event/mocks"
	"github.com/ecodeclub/webook/internal/cases/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/cases/internal/web"
	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"
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

type ExamineHandlerTest struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
	dao    dao.ExamineDAO
	ctrl   *gomock.Controller
}

func (s *ExamineHandlerTest) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())
	producer := eveMocks.NewMockSyncEventProducer(s.ctrl)
	ctrl := gomock.NewController(s.T())
	aiSvc := aimocks.NewMockService(ctrl)
	aiSvc.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, req ai.LLMRequest) (ai.LLMResponse, error) {
		return ai.LLMResponse{
			Tokens: req.Uid,
			Amount: req.Uid,
			Answer: "通过",
		}, nil
	}).AnyTimes()
	intrSvc := intrmocks.NewMockService(s.ctrl)
	intrModule := &interactive.Module{
		Svc: intrSvc,
	}
	module, err := startup.InitExamModule(producer, nil, intrModule,
		&member.Module{}, session.DefaultProvider(), &ai.Module{Svc: aiSvc})
	require.NoError(s.T(), err)
	hdl := module.ExamineHdl
	s.db = testioc.InitDB()
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set(session.CtxSessionKey,
			session.NewMemorySession(session.Claims{
				Uid: uid,
			}))
	})
	hdl.MemberRoutes(server.Engine)
	s.server = server
	s.dao = dao.NewGORMExamineDAO(s.db)
	// 提前准备 Question，这是所有测试都可以使用的
	err = s.db.Create(&dao.PublishCase{
		Id:    1,
		Title: "测试案例1",
	}).Error
	assert.NoError(s.T(), err)
	err = s.db.Create(&dao.PublishCase{
		Id:    2,
		Title: "测试案例2",
	}).Error
	assert.NoError(s.T(), err)
}

func (s *ExamineHandlerTest) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `case_examine_records`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `case_results`").Error
	require.NoError(s.T(), err)
}

func (s *ExamineHandlerTest) TearDownSuite() {
	err := s.db.Exec("TRUNCATE TABLE `publish_cases`").Error
	require.NoError(s.T(), err)
}

func (s *ExamineHandlerTest) TestExamine() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		req web.ExamineReq

		wantCode int
		wantResp test.Result[web.ExamineResult]
	}{
		{
			name: "新用户",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				var record dao.CaseExamineRecord
				err := s.db.Where("uid = ? ", uid).Order("id DESC").First(&record).Error
				require.NoError(t, err)
				assert.True(t, record.Utime > 0)
				record.Utime = 0
				assert.True(t, record.Ctime > 0)
				record.Ctime = 0
				assert.True(t, record.Id > 0)
				record.Id = 0
				assert.True(t, len(record.Tid) > 0)
				record.Tid = ""
				assert.Equal(t, dao.CaseExamineRecord{
					Uid:       uid,
					Cid:       1,
					Result:    domain.ResultPassed.ToUint8(),
					RawResult: "通过",
					Tokens:    uid,
					Amount:    uid,
				}, record)

				var caseRes dao.CaseResult
				err = s.db.WithContext(ctx).
					Where("cid = ? AND uid = ?", 1, uid).
					First(&caseRes).Error
				require.NoError(t, err)
				assert.True(t, caseRes.Ctime > 0)
				caseRes.Ctime = 0
				assert.True(t, caseRes.Utime > 0)
				caseRes.Utime = 0
				assert.True(t, caseRes.Id > 0)
				caseRes.Id = 0
				assert.Equal(t, dao.CaseResult{
					Result: domain.ResultPassed.ToUint8(),
					Cid:    1,
					Uid:    uid,
				}, caseRes)
			},
			req: web.ExamineReq{
				Cid:   1,
				Input: "测试一下",
			},
			wantCode: 200,
			wantResp: test.Result[web.ExamineResult]{
				Data: web.ExamineResult{
					Result:    domain.ResultPassed.ToUint8(),
					RawResult: "通过",
					Amount:    uid,
				},
			},
		},
		{
			// 这个测试依赖于前面的测试产生的 eid = 1
			name: "老用户重复测试",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.CaseResult{
					Id:     2,
					Uid:    uid,
					Cid:    2,
					Result: domain.ResultPassed.ToUint8(),
					Ctime:  123,
					Utime:  123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				const csid = 2
				var record dao.CaseExamineRecord
				err := s.db.Where("uid = ? ", uid).Order("id DESC").First(&record).Error
				require.NoError(t, err)
				assert.True(t, record.Utime > 0)
				record.Utime = 0
				assert.True(t, record.Ctime > 0)
				record.Ctime = 0
				assert.True(t, record.Id > 0)
				record.Id = 0
				assert.True(t, len(record.Tid) > 0)
				record.Tid = ""
				assert.Equal(t, dao.CaseExamineRecord{
					Uid:       uid,
					Cid:       2,
					Result:    domain.ResultPassed.ToUint8(),
					RawResult: "通过",
					Tokens:    uid,
					Amount:    uid,
				}, record)

				var caseRes dao.CaseResult
				err = s.db.WithContext(ctx).
					Where("cid = ? AND uid = ?", 2, uid).
					First(&caseRes).Error
				require.NoError(t, err)
				assert.True(t, caseRes.Ctime > 0)
				caseRes.Ctime = 0
				assert.True(t, caseRes.Utime > 0)
				caseRes.Utime = 0
				assert.True(t, caseRes.Id > 0)
				caseRes.Id = 0
				assert.Equal(t, dao.CaseResult{
					Result: domain.ResultPassed.ToUint8(),
					Cid:    csid,
					Uid:    uid,
				}, caseRes)
			},
			wantCode: 200,
			req: web.ExamineReq{
				Cid:   2,
				Input: "测试一下",
			},
			wantResp: test.Result[web.ExamineResult]{
				Data: web.ExamineResult{
					Result:    domain.ResultPassed.ToUint8(),
					RawResult: "通过",
					Amount:    uid,
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/cases/examine", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.ExamineResult]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
		})
	}
}

func TestExamineHandler(t *testing.T) {
	suite.Run(t, new(ExamineHandlerTest))
}
