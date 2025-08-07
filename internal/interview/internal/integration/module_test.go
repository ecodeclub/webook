// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build e2e

package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/interview/internal/domain"
	"github.com/ecodeclub/webook/internal/interview/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/interview/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/interview/internal/service"
	"github.com/ecodeclub/webook/internal/interview/internal/web"
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

const testID = int64(223999)

func TestInterviewModule(t *testing.T) {
	suite.Run(t, new(InterviewModuleTestSuite))
}

type InterviewModuleTestSuite struct {
	suite.Suite
	db  *egorm.Component
	svc service.InterviewService
}

func (s *InterviewModuleTestSuite) SetupSuite() {
	s.db = testioc.InitDB()
	s.NoError(dao.InitTables(s.db))
	m := startup.InitModule(s.db)
	s.svc = m.JourneySvc
}

func (s *InterviewModuleTestSuite) newGinServer(handler ginx.Handler) *egin.Component {
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: testID,
		}))
	})

	handler.PrivateRoutes(server.Engine)
	return server
}

func (s *InterviewModuleTestSuite) TearDownSuite() {
	// s.NoError(s.db.Exec("TRUNCATE TABLE `interview_journeys`").Error)
	// s.NoError(s.db.Exec("TRUNCATE TABLE `interview_rounds`").Error)
}

func (s *InterviewModuleTestSuite) TestHandler_Save() {
	t := s.T()

	testCases := []struct {
		name           string
		before         func(t *testing.T) (jid int64, roundIDs []int64)
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) ginx.Handler
		req            web.SaveReq

		wantCode       int
		respAssertFunc assert.ValueAssertionFunc
		after          func(t *testing.T, req web.SaveReq, resp web.SaveResp)
	}{

		{
			name: "仅创建面试历程无面试轮成功",
			before: func(t *testing.T) (jid int64, roundIDs []int64) {
				t.Helper()
				return 0, nil
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.SaveReq{
				Journey: web.Journey{
					CompanyName: "company-name-1",
					JobInfo:     "/jobinfo/1",
					ResumeURL:   "/resume/1",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusActive.String(),
				},
			},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[web.SaveResp])
				return assert.Positive(t, r.Data.Jid) && assert.Empty(t, r.Data.RoundIDs)
			},
			after: func(t *testing.T, req web.SaveReq, resp web.SaveResp) {
				t.Helper()
				actual, err := s.svc.Detail(t.Context(), resp.Jid, testID)
				require.NoError(t, err)
				s.assertJourney(t, req.Journey, actual)
			},
		},
		{
			name: "仅更新面试历程无面试轮成功",
			before: func(t *testing.T) (jid int64, roundIDs []int64) {
				t.Helper()
				id, roundIDs, err := s.svc.Save(t.Context(), domain.InterviewJourney{
					Uid:         testID,
					CompanyName: "company-name-2",
					JobInfo:     "/jobinfo/2",
					ResumeURL:   "/resume/2",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusActive,
				})
				require.NoError(t, err)
				return id, roundIDs
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.SaveReq{
				Journey: web.Journey{
					CompanyName: "company-name-3",
					JobInfo:     "/jobinfo/3",
					ResumeURL:   "/resume/3",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusAbandoned.String(),
				},
			},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[web.SaveResp])
				return assert.Positive(t, r.Data.Jid) && assert.Empty(t, r.Data.RoundIDs)
			},
			after: func(t *testing.T, req web.SaveReq, resp web.SaveResp) {
				t.Helper()
				actual, err := s.svc.Detail(t.Context(), resp.Jid, testID)
				require.NoError(t, err)
				s.assertJourney(t, req.Journey, actual)
			},
		},
		{
			name: "创建面试历程及一轮面试成功",
			before: func(t *testing.T) (jid int64, roundIDs []int64) {
				t.Helper()
				return 0, nil
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.SaveReq{
				Journey: web.Journey{
					CompanyName: "company-name-4",
					JobInfo:     "/jobinfo/4",
					ResumeURL:   "/resume/4",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusActive.String(),
					Rounds: []web.Round{
						{
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/1",
							ResumeURL:     "/resume/1",
							AudioURL:      "/audio/1",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultPending.String(),
							AllowSharing:  false,
						},
					},
				},
			},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[web.SaveResp])
				res := assert.Positive(t, r.Data.Jid)
				res = res && assert.NotEmpty(t, r.Data.RoundIDs)
				for i := range r.Data.RoundIDs {
					res = res && assert.Positive(t, r.Data.RoundIDs[i])
				}
				return res
			},
			after: func(t *testing.T, req web.SaveReq, resp web.SaveResp) {
				t.Helper()
				actual, err := s.svc.Detail(t.Context(), resp.Jid, testID)
				require.NoError(t, err)
				s.assertJourney(t, req.Journey, actual)
			},
		},
		{
			name: "更新面试历程及一轮面试成功",
			before: func(t *testing.T) (jid int64, roundIDs []int64) {
				t.Helper()
				id, roundIDs, err := s.svc.Save(t.Context(), domain.InterviewJourney{
					Uid:         testID,
					CompanyName: "company-name-5-更新面试历程及一轮面试成功",
					JobInfo:     "/jobinfo/5",
					ResumeURL:   "/resume/5",
					Stime:       time.Now().UnixMilli(),
					Rounds: []domain.InterviewRound{
						{
							Uid:           testID,
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/1",
							ResumeURL:     "/resume/1",
							AudioURL:      "/audio/1",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultPending,
							AllowSharing:  false,
						},
					},
				})
				require.NoError(t, err)
				return id, roundIDs
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.SaveReq{
				Journey: web.Journey{
					CompanyName: "company-name-6-更新面试历程及一轮面试成功",
					JobInfo:     "/jobinfo/6",
					ResumeURL:   "/resume/6",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusSucceeded.String(),
					Rounds: []web.Round{
						{
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/1",
							ResumeURL:     "/resume/1",
							AudioURL:      "/audio/1",
							SelfResult:    false,
							SelfSummary:   "一般",
							Result:        domain.ResultApproved.String(),
							AllowSharing:  false,
						},
					},
				},
			},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[web.SaveResp])
				return assert.Positive(t, r.Data.Jid) && assert.Len(t, r.Data.RoundIDs, 1)
			},
			after: func(t *testing.T, req web.SaveReq, resp web.SaveResp) {
				t.Helper()
				actual, err := s.svc.Detail(t.Context(), resp.Jid, testID)
				require.NoError(t, err)
				s.assertJourney(t, req.Journey, actual)
			},
		},
		{
			name: "创建面试历程及多轮面试成功",
			before: func(t *testing.T) (jid int64, roundIDs []int64) {
				t.Helper()
				return 0, nil
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.SaveReq{
				Journey: web.Journey{
					CompanyName: "company-name-7",
					JobInfo:     "/jobinfo/7",
					ResumeURL:   "/resume/7",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusFailed.String(),
					Rounds: []web.Round{
						{
							RoundNumber:   2,
							RoundType:     "技术2面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/2",
							ResumeURL:     "/resume/2",
							AudioURL:      "/audio/2",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultPending.String(),
							AllowSharing:  false,
						},
						{
							RoundNumber:   4,
							RoundType:     "技术4面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/2",
							ResumeURL:     "/resume/2",
							AudioURL:      "/audio/2",
							SelfResult:    false,
							SelfSummary:   "bad",
							Result:        domain.ResultRejected.String(),
							AllowSharing:  false,
						},
					},
				},
			},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[web.SaveResp])
				res := assert.Positive(t, r.Data.Jid)
				res = res && assert.Len(t, r.Data.RoundIDs, 2)
				for i := range r.Data.RoundIDs {
					res = res && assert.Positive(t, r.Data.RoundIDs[i])
				}
				return res
			},
			after: func(t *testing.T, req web.SaveReq, resp web.SaveResp) {
				t.Helper()
				actual, err := s.svc.Detail(t.Context(), resp.Jid, testID)
				require.NoError(t, err)
				s.assertJourney(t, req.Journey, actual)
			},
		},
		{
			name: "更新面试历程及多轮面试成功",
			before: func(t *testing.T) (jid int64, roundIDs []int64) {
				t.Helper()
				id, roundIDs, err := s.svc.Save(t.Context(), domain.InterviewJourney{
					Uid:         testID,
					CompanyName: "company-name-8",
					JobInfo:     "/jobinfo/8",
					ResumeURL:   "/resume/8",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusActive,
					Rounds: []domain.InterviewRound{
						{
							Uid:           testID,
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/1",
							ResumeURL:     "/resume/1",
							AudioURL:      "/audio/1",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultApproved,
							AllowSharing:  false,
						},
						{
							Uid:           testID,
							RoundNumber:   3,
							RoundType:     "技术3面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/1",
							ResumeURL:     "/resume/1",
							AudioURL:      "/audio/1",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultPending,
							AllowSharing:  false,
						},
					},
				})
				require.NoError(t, err)
				return id, roundIDs
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.SaveReq{
				Journey: web.Journey{
					CompanyName: "company-name-9",
					JobInfo:     "/jobinfo/9",
					ResumeURL:   "/resume/9",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusFailed.String(),
					Rounds: []web.Round{
						{
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/2",
							ResumeURL:     "/resume/2",
							AudioURL:      "/audio/2",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultPending.String(),
							AllowSharing:  true,
						},
						{
							RoundNumber:   3,
							RoundType:     "技术3面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/1",
							ResumeURL:     "/resume/1",
							AudioURL:      "/audio/1",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultPending.String(),
							AllowSharing:  true,
						},
					},
				},
			},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[web.SaveResp])
				return assert.Positive(t, r.Data.Jid) && assert.Len(t, r.Data.RoundIDs, 2)
			},
			after: func(t *testing.T, req web.SaveReq, resp web.SaveResp) {
				t.Helper()
				actual, err := s.svc.Detail(t.Context(), resp.Jid, testID)
				require.NoError(t, err)
				s.assertJourney(t, req.Journey, actual)
			},
		},
		{
			name: "更新面试历程及多个面试并创建多个面试成功",
			before: func(t *testing.T) (jid int64, roundIDs []int64) {
				t.Helper()
				id, roundIDs, err := s.svc.Save(t.Context(), domain.InterviewJourney{
					Uid:         testID,
					CompanyName: "company-name-10",
					JobInfo:     "/jobinfo/10",
					ResumeURL:   "/resume/10",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusActive,
					Rounds: []domain.InterviewRound{
						{
							Uid:           testID,
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/1",
							ResumeURL:     "/resume/1",
							AudioURL:      "/audio/1",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultApproved,
							AllowSharing:  false,
						},
						{
							Uid:           testID,
							RoundNumber:   2,
							RoundType:     "技术2面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/2",
							ResumeURL:     "/resume/2",
							AudioURL:      "/audio/2",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultPending,
							AllowSharing:  false,
						},
					},
				})
				require.NoError(t, err)
				return id, roundIDs
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.SaveReq{
				Journey: web.Journey{
					CompanyName: "company-name-11",
					JobInfo:     "/jobinfo/11",
					ResumeURL:   "/resume/11",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusFailed.String(),
					Rounds: []web.Round{
						{
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/2",
							ResumeURL:     "/resume/2",
							AudioURL:      "/audio/2",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultPending.String(),
							AllowSharing:  true,
						},
						{
							RoundNumber:   2,
							RoundType:     "技术2面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/2",
							ResumeURL:     "/resume/2",
							AudioURL:      "/audio/2",
							SelfResult:    false,
							SelfSummary:   "bad",
							Result:        domain.ResultPending.String(),
							AllowSharing:  true,
						},
						{
							RoundNumber:   3,
							RoundType:     "技术3面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/3",
							ResumeURL:     "/resume/3",
							AudioURL:      "/audio/3",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultPending.String(),
							AllowSharing:  true,
						},
						{
							RoundNumber:   4,
							RoundType:     "技术4面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/4",
							ResumeURL:     "/resume/4",
							AudioURL:      "/audio/4",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultPending.String(),
							AllowSharing:  true,
						},
					},
				},
			},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[web.SaveResp])
				res := assert.Positive(t, r.Data.Jid)
				res = res && assert.Len(t, r.Data.RoundIDs, 4)
				for i := range r.Data.RoundIDs {
					res = res && assert.Positive(t, r.Data.RoundIDs[i])
				}
				return res
			},
			after: func(t *testing.T, req web.SaveReq, resp web.SaveResp) {
				t.Helper()
				actual, err := s.svc.Detail(t.Context(), resp.Jid, testID)
				require.NoError(t, err)
				s.assertJourney(t, req.Journey, actual)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			jid, roundIDs := tc.before(t)

			tc.req.Journey.ID = jid
			require.GreaterOrEqual(t, len(tc.req.Journey.Rounds), len(roundIDs))
			for i := range roundIDs {
				tc.req.Journey.Rounds[i].ID = roundIDs[i]
			}

			req, err := http.NewRequest(http.MethodPost,
				"/interview-journeys/save", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[web.SaveResp]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			result := recorder.MustScan()
			tc.respAssertFunc(t, result)
			tc.after(t, tc.req, result.Data)
		})
	}
}

func (s *InterviewModuleTestSuite) assertJourney(t *testing.T, expected web.Journey, actual domain.InterviewJourney) {
	t.Helper()

	assert.Equal(t, expected.CompanyID, actual.CompanyID)
	assert.Equal(t, expected.CompanyName, actual.CompanyName)
	assert.Equal(t, expected.JobInfo, actual.JobInfo)
	assert.Equal(t, expected.ResumeURL, actual.ResumeURL)
	assert.Equal(t, expected.Status, actual.Status.String())
	assert.Equal(t, expected.Stime, actual.Stime)
	assert.Equal(t, expected.Etime, actual.Etime)

	require.Equal(t, len(expected.Rounds), len(actual.Rounds))
	for i := range expected.Rounds {
		s.assertRound(t, expected.Rounds[i], actual.Rounds[i])
	}
}

func (s *InterviewModuleTestSuite) assertRound(t *testing.T, expected web.Round, actual domain.InterviewRound) {
	t.Helper()
	assert.Greater(t, actual.ID, int64(0))
	assert.Equal(t, testID, actual.Uid)
	assert.Equal(t, expected.RoundNumber, actual.RoundNumber)
	assert.Equal(t, expected.RoundType, actual.RoundType)
	assert.Equal(t, expected.InterviewDate, actual.InterviewDate)
	assert.Equal(t, expected.JobInfo, actual.JobInfo)
	assert.Equal(t, expected.ResumeURL, actual.ResumeURL)
	assert.Equal(t, expected.AudioURL, actual.AudioURL)
	assert.Equal(t, expected.SelfResult, actual.SelfResult)
	assert.Equal(t, expected.SelfSummary, actual.SelfSummary)
	assert.Equal(t, expected.Result, actual.Result.String())
	assert.Equal(t, expected.AllowSharing, actual.AllowSharing)
}

func (s *InterviewModuleTestSuite) TestHandler_Save_Failed() {
	t := s.T()

	testCases := []struct {
		name           string
		before         func(t *testing.T) (jid int64, roundIDs []int64)
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) ginx.Handler
		req            web.SaveReq

		wantCode       int
		respAssertFunc assert.ValueAssertionFunc
	}{

		{
			name: "有面试历程撤销一个授权失败",
			before: func(t *testing.T) (jid int64, roundIDs []int64) {
				t.Helper()
				id, roundIDs, err := s.svc.Save(t.Context(), domain.InterviewJourney{
					Uid:         testID,
					CompanyName: "company-name-12",
					JobInfo:     "/jobinfo/12",
					ResumeURL:   "/resume/12",
					Stime:       time.Now().UnixMilli(),
					Rounds: []domain.InterviewRound{
						{
							Uid:           testID,
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/1",
							ResumeURL:     "/resume/1",
							AudioURL:      "/audio/1",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultApproved,
							AllowSharing:  true,
						},
					},
				})
				require.NoError(t, err)
				return id, roundIDs
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.SaveReq{
				Journey: web.Journey{
					CompanyName: "company-name-13",
					JobInfo:     "/jobinfo/13",
					ResumeURL:   "/resume/13",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusSucceeded.String(),
					Rounds: []web.Round{
						{
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/1",
							ResumeURL:     "/resume/1",
							AudioURL:      "/audio/1",
							SelfResult:    false,
							SelfSummary:   "一般",
							Result:        domain.ResultApproved.String(),
							AllowSharing:  false,
						},
					},
				},
			},
			wantCode: 500,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[any])
				return assert.Equal(t, test.Result[any]{Code: 519001, Msg: "系统错误"}, r)
			},
		},
		{
			name: "有面试历程撤销多个授权失败",
			before: func(t *testing.T) (jid int64, roundIDs []int64) {
				t.Helper()
				id, roundIDs, err := s.svc.Save(t.Context(), domain.InterviewJourney{
					Uid:         testID,
					CompanyName: "company-name-12",
					JobInfo:     "/jobinfo/12",
					ResumeURL:   "/resume/12",
					Stime:       time.Now().UnixMilli(),
					Rounds: []domain.InterviewRound{
						{
							Uid:           testID,
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/1",
							ResumeURL:     "/resume/1",
							AudioURL:      "/audio/1",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultApproved,
							AllowSharing:  true,
						},
						{
							Uid:           testID,
							RoundNumber:   2,
							RoundType:     "技术2面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/2",
							ResumeURL:     "/resume/2",
							AudioURL:      "/audio/2",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultApproved,
							AllowSharing:  true,
						},
						{
							Uid:           testID,
							RoundNumber:   3,
							RoundType:     "技术3面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/3",
							ResumeURL:     "/resume/3",
							AudioURL:      "/audio/3",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultApproved,
							AllowSharing:  true,
						},
					},
				})
				require.NoError(t, err)
				return id, roundIDs
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.SaveReq{
				Journey: web.Journey{
					CompanyName: "company-name-13",
					JobInfo:     "/jobinfo/13",
					ResumeURL:   "/resume/13",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusSucceeded.String(),
					Rounds: []web.Round{
						{
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/1",
							ResumeURL:     "/resume/1",
							AudioURL:      "/audio/1",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultApproved.String(),
							AllowSharing:  false,
						},
						{
							RoundNumber:   2,
							RoundType:     "技术2面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/2",
							ResumeURL:     "/resume/2",
							AudioURL:      "/audio/2",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultApproved.String(),
							AllowSharing:  true,
						},
						{
							RoundNumber:   3,
							RoundType:     "技术3面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/3",
							ResumeURL:     "/resume/3",
							AudioURL:      "/audio/3",
							SelfResult:    true,
							SelfSummary:   "good",
							Result:        domain.ResultApproved.String(),
							AllowSharing:  false,
						},
					},
				},
			},
			wantCode: 500,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[any])
				return assert.Equal(t, test.Result[any]{Code: 519001, Msg: "系统错误"}, r)
			},
		},
		{
			name: "无面试历程仅创建一轮面试失败",
			before: func(t *testing.T) (jid int64, roundIDs []int64) {
				t.Helper()
				return 0, nil
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.SaveReq{
				Journey: web.Journey{
					Rounds: []web.Round{
						{
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/1",
							ResumeURL:     "/resume/1",
							AudioURL:      "/audio/1",
							SelfResult:    false,
							SelfSummary:   "一般",
							Result:        domain.ResultApproved.String(),
							AllowSharing:  false,
						},
					},
				},
			},
			wantCode: 500,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[any])
				return assert.Equal(t, test.Result[any]{Code: 419001, Msg: "面试历程有必填字段未填写"}, r)
			},
		},
		{
			name: "无面试历程仅创建多轮面试失败",
			before: func(t *testing.T) (jid int64, roundIDs []int64) {
				t.Helper()
				return 0, nil
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.SaveReq{
				Journey: web.Journey{
					Rounds: []web.Round{
						{
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/1",
							ResumeURL:     "/resume/1",
							AudioURL:      "/audio/1",
							SelfResult:    false,
							SelfSummary:   "一般",
							Result:        domain.ResultApproved.String(),
							AllowSharing:  false,
						},
						{
							RoundNumber:   2,
							RoundType:     "技术2面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/2",
							ResumeURL:     "/resume/2",
							AudioURL:      "/audio/2",
							SelfResult:    false,
							SelfSummary:   "一般",
							Result:        domain.ResultApproved.String(),
							AllowSharing:  false,
						},
					},
				},
			},
			wantCode: 500,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[any])
				return assert.Equal(t, test.Result[any]{Code: 419001, Msg: "面试历程有必填字段未填写"}, r)
			},
		},
		{
			name: "有面试历程仅创建一轮面试失败",
			before: func(t *testing.T) (jid int64, roundIDs []int64) {
				t.Helper()
				return 0, nil
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.SaveReq{
				Journey: web.Journey{
					CompanyName: "company-name-14",
					JobInfo:     "/jobinfo/14",
					ResumeURL:   "/resume/14",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusSucceeded.String(),
					Rounds: []web.Round{
						{
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/1",
							AudioURL:      "/audio/1",
							SelfResult:    false,
							SelfSummary:   "一般",
							Result:        domain.ResultApproved.String(),
							AllowSharing:  false,
						},
					},
				},
			},
			wantCode: 500,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[any])
				return assert.Equal(t, test.Result[any]{Code: 419002, Msg: "面试轮次有必填字段未填写"}, r)
			},
		},
		{
			name: "有面试历程仅创建多轮面试失败",
			before: func(t *testing.T) (jid int64, roundIDs []int64) {
				t.Helper()
				return 0, nil
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.SaveReq{
				Journey: web.Journey{
					CompanyName: "company-name-15",
					JobInfo:     "/jobinfo/15",
					ResumeURL:   "/resume/15",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusFailed.String(),
					Rounds: []web.Round{
						{
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/1",
							ResumeURL:     "/resume/1",
							AudioURL:      "/audio/1",
							SelfResult:    false,
							SelfSummary:   "一般",
							Result:        domain.ResultApproved.String(),
							AllowSharing:  false,
						},
						{
							RoundNumber:   2,
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/2",
							ResumeURL:     "/resume/2",
							AudioURL:      "/audio/2",
							SelfResult:    false,
							SelfSummary:   "一般",
							Result:        domain.ResultApproved.String(),
							AllowSharing:  false,
						},
					},
				},
			},
			wantCode: 500,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[any])
				return assert.Equal(t, test.Result[any]{Code: 419002, Msg: "面试轮次有必填字段未填写"}, r)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			jid, roundIDs := tc.before(t)

			tc.req.Journey.ID = jid
			require.GreaterOrEqual(t, len(tc.req.Journey.Rounds), len(roundIDs))
			for i := range roundIDs {
				tc.req.Journey.Rounds[i].ID = roundIDs[i]
			}

			req, err := http.NewRequest(http.MethodPost,
				"/interview-journeys/save", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[any]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			result := recorder.MustScan()
			tc.respAssertFunc(t, result)
		})
	}
}

/*
func (s *InterviewModuleTestSuite) TestAdminHandler_List() {
	t := s.T()

	err := s.db.Exec("TRUNCATE TABLE `materials`").Error
	require.NoError(t, err)

	total := 10
	for idx := 0; idx < total; idx++ {
		id := int64(3000 + idx)
		_, err := s.svc.Submit(context.Background(), domain.interview{
			Uid:       id,
			AudioURL:  fmt.Sprintf("/%d/admin/audio", id),
			ResumeURL: fmt.Sprintf("/%d/admin/resume", id),
			Remark:    fmt.Sprintf("admin/remark-%d", id),
		})
		require.NoError(t, err)
	}

	listReq := web.ListMaterialsReq{
		Limit:  2,
		Offset: 0,
	}

	req, err := http.NewRequest(http.MethodPost,
		"/interview/list", iox.NewJSONReader(listReq))
	require.NoError(t, err)
	req.Header.Set("content-type", "application/json")
	recorder := test.NewJSONResponseRecorder[web.ListMaterialsResp]()
	server := s.newAdminGinServer(web.NewAdminHandler(s.svc, nil, nil, nil))
	server.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
	result := recorder.MustScan()
	require.Equal(t, int64(total), result.Data.Total)
	require.Len(t, result.Data.Materials, listReq.Limit)
}

func (s *InterviewModuleTestSuite) TestAdminHandler_Accept() {
	t := s.T()
	testCases := []struct {
		name           string
		before         func(t *testing.T) int64
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller, id int64) *web.AdminHandler
		req            web.AcceptMaterialReq

		wantCode int
		wantResp test.Result[any]
		after    func(t *testing.T, id int64)
	}{
		{
			name: "接受素材成功",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, err := s.svc.Submit(t.Context(), domain.interview{
					Uid:       testID,
					AudioURL:  fmt.Sprintf("/%d/admin/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/admin/resume", testID),
					Remark:    fmt.Sprintf("admin/remark-%d", testID),
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller, id int64) *web.AdminHandler {
				t.Helper()
				producer := evtmocks.NewMockMemberEventProducer(ctrl)
				producer.EXPECT().Produce(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, event event.MemberEvent) error {
					assert.NotZero(t, event.Key)
					assert.Equal(t, testID, event.Uid)
					assert.Equal(t, uint64(30), event.Days)
					assert.Equal(t, "interview", event.Biz)
					assert.Equal(t, id, event.BizId)
					assert.Equal(t, "素材被采纳", event.Action)
					return nil
				}).Times(1)
				return web.NewAdminHandler(s.svc, nil, producer, nil)
			},
			req: web.AcceptMaterialReq{
				ID: 0,
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
				var interview domain.interview
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).FindByJourneyID(&interview).Error)
				assert.Equal(t, testID, interview.Uid)
				assert.Equal(t, fmt.Sprintf("/%d/admin/audio", testID), interview.AudioURL)
				assert.Equal(t, fmt.Sprintf("/%d/admin/resume", testID), interview.ResumeURL)
				assert.Equal(t, fmt.Sprintf("admin/remark-%d", testID), interview.Remark)
				assert.Equal(t, domain.MaterialStatusAccepted, interview.Status)
				assert.NotZero(t, interview.Ctime)
				assert.NotZero(t, interview.Utime)
			},
		},
		{
			name: "接受素材失败_素材ID不存在",
			before: func(t *testing.T) int64 {
				t.Helper()
				return -1
			},
			newHandlerFunc: func(t *testing.T, _ *gomock.Controller, _ int64) *web.AdminHandler {
				t.Helper()
				return web.NewAdminHandler(s.svc, nil, nil, nil)
			},
			req: web.AcceptMaterialReq{
				ID: 0,
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 518001, Msg: "系统错误",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
			},
		},
		{
			name: "接受素材失败_福利发放失败",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, err := s.svc.Submit(t.Context(), domain.interview{
					Uid:       testID,
					AudioURL:  fmt.Sprintf("/%d/admin/2/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/admin/2/resume", testID),
					Remark:    fmt.Sprintf("admin/2/remark-%d", testID),
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller, _ int64) *web.AdminHandler {
				t.Helper()
				producer := evtmocks.NewMockMemberEventProducer(ctrl)
				producer.EXPECT().Produce(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ event.MemberEvent) error {
					return errors.New("fake error")
				}).Times(1)
				return web.NewAdminHandler(s.svc, nil, producer, nil)
			},
			req: web.AcceptMaterialReq{
				ID: 0,
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
				var interview domain.interview
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).FindByJourneyID(&interview).Error)
				assert.Equal(t, testID, interview.Uid)
				assert.Equal(t, fmt.Sprintf("/%d/admin/2/audio", testID), interview.AudioURL)
				assert.Equal(t, fmt.Sprintf("/%d/admin/2/resume", testID), interview.ResumeURL)
				assert.Equal(t, fmt.Sprintf("admin/2/remark-%d", testID), interview.Remark)
				assert.Equal(t, domain.MaterialStatusAccepted, interview.Status)
				assert.NotZero(t, interview.Ctime)
				assert.NotZero(t, interview.Utime)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			id := tc.before(t)
			tc.req.ID = id

			req, err := http.NewRequest(http.MethodPost,
				"/interview/accept", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[any]()
			server := s.newAdminGinServer(tc.newHandlerFunc(t, ctrl, id))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp.Data, recorder.MustScan().Data)

			tc.after(t, id)
		})
	}
}

func (s *InterviewModuleTestSuite) TestAdminHandler_Notify() {
	t := s.T()
	testCases := []struct {
		name           string
		before         func(t *testing.T) int64
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.AdminHandler
		req            web.NotifyUserReq

		wantCode int
		wantResp test.Result[any]
		after    func(t *testing.T, id int64)
	}{
		{
			name: "通知用户成功",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, err := s.svc.Submit(t.Context(), domain.interview{
					Uid:       testID,
					AudioURL:  fmt.Sprintf("/%d/admin/3/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/admin/3/resume", testID),
					Remark:    fmt.Sprintf("admin/3/remark-%d", testID),
				})
				require.NoError(t, err)
				err = s.svc.Accept(t.Context(), id)
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.AdminHandler {
				t.Helper()
				userSvc := usermocks.NewMockUserService(ctrl)
				userSvc.EXPECT().Profile(gomock.Any(), testID).Return(user.User{Id: testID, Phone: "13845016319"}, nil).Times(1)

				cli := smsmocks.NewMockClient(ctrl)
				cli.EXPECT().Send(gomock.Any()).DoAndReturn(func(req client.SendReq) (client.SendResp, error) {
					assert.Contains(t, req.PhoneNumbers, "13845016319")
					assert.NotZero(t, req.TemplateID)
					assert.Equal(t, "2025-7-01 20:00", req.TemplateParam["date"])
					return client.SendResp{
						RequestID: fmt.Sprintf("%d", time.Now().UnixMilli()),
						PhoneNumbers: map[string]client.SendRespStatus{
							"13845016319": {
								Code:    client.OK,
								Message: "发送成功",
							},
						},
					}, nil
				})
				return web.NewAdminHandler(s.svc, userSvc, nil, cli)
			},
			req: web.NotifyUserReq{
				Date: "2025-7-01 20:00",
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
				var interview domain.interview
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).FindByJourneyID(&interview).Error)
				assert.Equal(t, testID, interview.Uid)
				assert.Equal(t, fmt.Sprintf("/%d/admin/3/audio", testID), interview.AudioURL)
				assert.Equal(t, fmt.Sprintf("/%d/admin/3/resume", testID), interview.ResumeURL)
				assert.Equal(t, fmt.Sprintf("admin/3/remark-%d", testID), interview.Remark)
				assert.Equal(t, domain.MaterialStatusAccepted, interview.Status)
				assert.NotZero(t, interview.Ctime)
				assert.NotZero(t, interview.Utime)
			},
		},
		{
			name: "通知用户失败_素材未被接受",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, err := s.svc.Submit(t.Context(), domain.interview{
					Uid:       testID,
					AudioURL:  fmt.Sprintf("/%d/admin/4/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/admin/4/resume", testID),
					Remark:    fmt.Sprintf("admin/4/remark-%d", testID),
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, _ *gomock.Controller) *web.AdminHandler {
				t.Helper()
				return web.NewAdminHandler(s.svc, nil, nil, nil)
			},
			req: web.NotifyUserReq{
				Date: "2025-7-02 20:00",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 518001, Msg: "系统错误",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
				var interview domain.interview
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).FindByJourneyID(&interview).Error)
				assert.Equal(t, testID, interview.Uid)
				assert.Equal(t, fmt.Sprintf("/%d/admin/4/audio", testID), interview.AudioURL)
				assert.Equal(t, fmt.Sprintf("/%d/admin/4/resume", testID), interview.ResumeURL)
				assert.Equal(t, fmt.Sprintf("admin/4/remark-%d", testID), interview.Remark)
				assert.Equal(t, domain.MaterialStatusInit, interview.Status)
				assert.NotZero(t, interview.Ctime)
				assert.NotZero(t, interview.Utime)
			},
		},
		{
			name: "通知用户失败_用户ID不存在",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, err := s.svc.Submit(t.Context(), domain.interview{
					Uid:       testID,
					AudioURL:  fmt.Sprintf("/%d/admin/5/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/admin/5/resume", testID),
					Remark:    fmt.Sprintf("admin/5/remark-%d", testID),
				})
				require.NoError(t, err)
				err = s.svc.Accept(t.Context(), id)
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.AdminHandler {
				t.Helper()
				userSvc := usermocks.NewMockUserService(ctrl)
				userSvc.EXPECT().Profile(gomock.Any(), testID).Return(user.User{}, errors.New("fake error")).Times(1)
				return web.NewAdminHandler(s.svc, userSvc, nil, nil)
			},
			req: web.NotifyUserReq{
				Date: "2025-7-03 20:00",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 518001, Msg: "系统错误",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
				var interview domain.interview
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).FindByJourneyID(&interview).Error)
				assert.Equal(t, testID, interview.Uid)
				assert.Equal(t, fmt.Sprintf("/%d/admin/5/audio", testID), interview.AudioURL)
				assert.Equal(t, fmt.Sprintf("/%d/admin/5/resume", testID), interview.ResumeURL)
				assert.Equal(t, fmt.Sprintf("admin/5/remark-%d", testID), interview.Remark)
				assert.Equal(t, domain.MaterialStatusAccepted, interview.Status)
				assert.NotZero(t, interview.Ctime)
				assert.NotZero(t, interview.Utime)
			},
		},
		{
			name: "通知用户失败_用户未绑定手机号",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, err := s.svc.Submit(t.Context(), domain.interview{
					Uid:       testID,
					AudioURL:  fmt.Sprintf("/%d/admin/6/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/admin/6/resume", testID),
					Remark:    fmt.Sprintf("admin/6/remark-%d", testID),
				})
				require.NoError(t, err)
				err = s.svc.Accept(t.Context(), id)
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.AdminHandler {
				t.Helper()
				userSvc := usermocks.NewMockUserService(ctrl)
				userSvc.EXPECT().Profile(gomock.Any(), testID).Return(user.User{Id: testID, Phone: ""}, nil).Times(1)
				return web.NewAdminHandler(s.svc, userSvc, nil, nil)
			},
			req: web.NotifyUserReq{
				Date: "2025-7-04 20:00",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 418001, Msg: "用户未绑定手机号",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
				var interview domain.interview
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).FindByJourneyID(&interview).Error)
				assert.Equal(t, testID, interview.Uid)
				assert.Equal(t, fmt.Sprintf("/%d/admin/6/audio", testID), interview.AudioURL)
				assert.Equal(t, fmt.Sprintf("/%d/admin/6/resume", testID), interview.ResumeURL)
				assert.Equal(t, fmt.Sprintf("admin/6/remark-%d", testID), interview.Remark)
				assert.Equal(t, domain.MaterialStatusAccepted, interview.Status)
				assert.NotZero(t, interview.Ctime)
				assert.NotZero(t, interview.Utime)
			},
		},
		{
			name: "通知用户失败_发送短信失败",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, err := s.svc.Submit(t.Context(), domain.interview{
					Uid:       testID,
					AudioURL:  fmt.Sprintf("/%d/admin/7/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/admin/7/resume", testID),
					Remark:    fmt.Sprintf("admin/7/remark-%d", testID),
				})
				require.NoError(t, err)
				err = s.svc.Accept(t.Context(), id)
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.AdminHandler {
				t.Helper()
				t.Helper()
				userSvc := usermocks.NewMockUserService(ctrl)
				userSvc.EXPECT().Profile(gomock.Any(), testID).Return(user.User{Id: testID, Phone: "13845016319"}, nil).Times(1)

				cli := smsmocks.NewMockClient(ctrl)
				cli.EXPECT().Send(gomock.Any()).DoAndReturn(func(req client.SendReq) (client.SendResp, error) {
					assert.Contains(t, req.PhoneNumbers, "13845016319")
					assert.NotZero(t, req.TemplateID)
					assert.Equal(t, "2025-7-05 20:00", req.TemplateParam["date"])
					return client.SendResp{}, errors.New("fake error")
				})
				return web.NewAdminHandler(s.svc, userSvc, nil, cli)
			},
			req: web.NotifyUserReq{
				Date: "2025-7-05 20:00",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 518001, Msg: "系统错误",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
				var interview domain.interview
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).FindByJourneyID(&interview).Error)
				assert.Equal(t, testID, interview.Uid)
				assert.Equal(t, fmt.Sprintf("/%d/admin/7/audio", testID), interview.AudioURL)
				assert.Equal(t, fmt.Sprintf("/%d/admin/7/resume", testID), interview.ResumeURL)
				assert.Equal(t, fmt.Sprintf("admin/7/remark-%d", testID), interview.Remark)
				assert.Equal(t, domain.MaterialStatusAccepted, interview.Status)
				assert.NotZero(t, interview.Ctime)
				assert.NotZero(t, interview.Utime)
			},
		},
		{
			name: "通知用户失败_用户接收失败",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, err := s.svc.Submit(t.Context(), domain.interview{
					Uid:       testID,
					AudioURL:  fmt.Sprintf("/%d/admin/8/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/admin/8/resume", testID),
					Remark:    fmt.Sprintf("admin/8/remark-%d", testID),
				})
				require.NoError(t, err)
				err = s.svc.Accept(t.Context(), id)
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.AdminHandler {
				t.Helper()
				userSvc := usermocks.NewMockUserService(ctrl)
				userSvc.EXPECT().Profile(gomock.Any(), testID).Return(user.User{Id: testID, Phone: "13845016329"}, nil).Times(1)

				cli := smsmocks.NewMockClient(ctrl)
				cli.EXPECT().Send(gomock.Any()).DoAndReturn(func(req client.SendReq) (client.SendResp, error) {
					assert.Contains(t, req.PhoneNumbers, "13845016329")
					assert.NotZero(t, req.TemplateID)
					assert.Equal(t, "2025-7-06 20:00", req.TemplateParam["date"])
					return client.SendResp{
						RequestID: fmt.Sprintf("%d", time.Now().UnixMilli()),
						PhoneNumbers: map[string]client.SendRespStatus{
							"13845016329": {
								Code:    "Failed",
								Message: "用户已停机",
							},
						},
					}, nil
				})
				return web.NewAdminHandler(s.svc, userSvc, nil, cli)
			},
			req: web.NotifyUserReq{
				Date: "2025-7-06 20:00",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 518002, Msg: "用户接收通知失败",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
				var interview domain.interview
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).FindByJourneyID(&interview).Error)
				assert.Equal(t, testID, interview.Uid)
				assert.Equal(t, fmt.Sprintf("/%d/admin/8/audio", testID), interview.AudioURL)
				assert.Equal(t, fmt.Sprintf("/%d/admin/8/resume", testID), interview.ResumeURL)
				assert.Equal(t, fmt.Sprintf("admin/8/remark-%d", testID), interview.Remark)
				assert.Equal(t, domain.MaterialStatusAccepted, interview.Status)
				assert.NotZero(t, interview.Ctime)
				assert.NotZero(t, interview.Utime)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			id := tc.before(t)
			tc.req.ID = id

			req, err := http.NewRequest(http.MethodPost,
				"/interview/notify", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[any]()
			server := s.newAdminGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp.Data, recorder.MustScan().Data)

			tc.after(t, id)
		})
	}
}
*/
