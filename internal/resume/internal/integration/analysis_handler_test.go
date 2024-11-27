package integration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/ai"
	aimocks "github.com/ecodeclub/webook/internal/ai/mocks"
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/resume/internal/domain"
	"github.com/ecodeclub/webook/internal/resume/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/resume/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type AnalysisTestSuite struct {
	suite.Suite
	server *egin.Component
}

func (a *AnalysisTestSuite) SetupSuite() {
	ctrl := gomock.NewController(a.T())
	aiSvc := aimocks.NewMockService(ctrl)
	aiSvc.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, req ai.LLMRequest) (ai.LLMResponse, error) {
		switch req.Biz {
		case domain.BizResumeSkillKeyPoints:
			return ai.LLMResponse{
				Tokens: 100,
				Amount: 100,
				Answer: fmt.Sprintf("skill的keypoints %s", req.Input[0]),
			}, nil
		case domain.BizSkillsRewrite:
			return ai.LLMResponse{
				Tokens: 200,
				Amount: 200,
				Answer: fmt.Sprintf("%s:%s", req.Input[0], req.Input[1]),
			}, nil
		case domain.BizResumeProjectKeyPoints:
			return ai.LLMResponse{
				Tokens: 150,
				Amount: 150,
				Answer: fmt.Sprintf("project的keypoints %s", req.Input[0]),
			}, nil
		case domain.BizResumeProjectRewrite:
			return ai.LLMResponse{
				Tokens: 220,
				Amount: 220,
				Answer: fmt.Sprintf("%s:%s", req.Input[0], req.Input[1]),
			}, nil
		case domain.BizResumeJobsKeyPoints:
			return ai.LLMResponse{
				Tokens: 300,
				Amount: 300,
				Answer: fmt.Sprintf("jobs的keypoints %s", req.Input[0]),
			}, nil
		case domain.BizResumeJobsRewrite:
			return ai.LLMResponse{
				Tokens: 400,
				Amount: 400,
				Answer: fmt.Sprintf("%s:%s", req.Input[0], req.Input[1]),
			}, nil
		default:
			return ai.LLMResponse{}, errors.New("mock Err")
		}
	}).AnyTimes()
	module := startup.InitModule(&cases.Module{}, &ai.Module{Svc: aiSvc})

	hdl := module.AnalysisHandler
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set(session.CtxSessionKey,
			session.NewMemorySession(session.Claims{
				Uid: uid,
			}))
	})
	hdl.PublicRoutes(server.Engine)
	a.server = server
}

func (a *AnalysisTestSuite) TestAnalysis() {
	testCases := []struct {
		name string

		req web.AnalysisReq

		wantCode int
		wantResp test.Result[web.AnalysisResp]
	}{
		{
			name: "分析简历",
			req: web.AnalysisReq{
				Resume: "resume",
			},
			wantCode: 200,
			wantResp: test.Result[web.AnalysisResp]{
				Data: web.AnalysisResp{
					Amount:         1370,
					RewriteSkills:  "resume:skill的keypoints resume",
					RewriteJobs:    "resume:jobs的keypoints resume",
					RewriteProject: "resume:project的keypoints resume",
				},
			},
		},
	}
	for _, tc := range testCases {
		a.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/resume/analysis", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.AnalysisResp]()
			a.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			data := recorder.MustScan()
			assert.Equal(t, tc.wantResp, data)
		})
	}
}

func TestAnalysisModule(t *testing.T) {
	suite.Run(t, new(AnalysisTestSuite))
}
