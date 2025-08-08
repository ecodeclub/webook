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
	"fmt"
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

const (
	testID  = int64(223999)
	testID2 = int64(224000)
	testID3 = int64(224001)
	testID4 = int64(224002)
	testID5 = int64(224003)
	testID6 = int64(224004)
	testID7 = int64(224005)
)

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

func (s *InterviewModuleTestSuite) newGinServer(uid int64, handler ginx.Handler) *egin.Component {
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: uid,
		}))
	})

	handler.PrivateRoutes(server.Engine)
	return server
}

func (s *InterviewModuleTestSuite) TearDownSuite() {
	s.NoError(s.db.Exec("TRUNCATE TABLE `interview_journeys`").Error)
	s.NoError(s.db.Exec("TRUNCATE TABLE `interview_rounds`").Error)
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
			server := s.newGinServer(testID, tc.newHandlerFunc(t, ctrl))
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
			server := s.newGinServer(testID, tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			result := recorder.MustScan()
			tc.respAssertFunc(t, result)
		})
	}
}

func (s *InterviewModuleTestSuite) TestHandler_List() {
	t := s.T()

	testCases := []struct {
		name           string
		before         func(t *testing.T) (journeyIDs []int64, uid int64)
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) ginx.Handler
		req            web.ListReq

		wantCode       int
		respAssertFunc assert.ValueAssertionFunc
		after          func(t *testing.T, resp ginx.DataList[web.Journey])
	}{
		{
			name: "空列表查询成功",
			before: func(t *testing.T) (journeyIDs []int64, uid int64) {
				t.Helper()
				return []int64{}, testID2 // 不创建任何数据
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.ListReq{
				Offset: 0,
				Limit:  10,
			},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[ginx.DataList[web.Journey]])
				return assert.Empty(t, r.Data.List) && assert.Equal(t, 0, r.Data.Total)
			},
			after: func(t *testing.T, resp ginx.DataList[web.Journey]) {
				t.Helper()
				assert.Empty(t, resp.List)
				assert.Equal(t, 0, resp.Total)
			},
		},
		{
			name: "单条记录查询成功",
			before: func(t *testing.T) (journeyIDs []int64, uid int64) {
				t.Helper()
				id, _, err := s.svc.Save(t.Context(), domain.InterviewJourney{
					Uid:         testID4,
					CompanyName: "单条记录公司",
					JobInfo:     "/jobinfo/single",
					ResumeURL:   "/resume/single",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusActive,
				})
				require.NoError(t, err)
				return []int64{id}, testID4
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.ListReq{
				Offset: 0,
				Limit:  10,
			},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[ginx.DataList[web.Journey]])
				return assert.Len(t, r.Data.List, 1) && assert.Equal(t, 1, r.Data.Total)
			},
			after: func(t *testing.T, resp ginx.DataList[web.Journey]) {
				t.Helper()
				assert.Len(t, resp.List, 1)
				assert.Equal(t, 1, resp.Total)
				assert.Equal(t, "单条记录公司", resp.List[0].CompanyName)
				assert.Empty(t, resp.List[0].Rounds) // List接口不包含轮次信息
			},
		},
		{
			name: "多条记录查询成功",
			before: func(t *testing.T) (journeyIDs []int64, uid int64) {
				t.Helper()
				var ids []int64
				companies := []string{"公司A", "公司B", "公司C"}

				uid = testID5
				for i, company := range companies {
					id, _, err := s.svc.Save(t.Context(), domain.InterviewJourney{
						Uid:         uid,
						CompanyName: company,
						JobInfo:     fmt.Sprintf("/jobinfo/%d", i),
						ResumeURL:   fmt.Sprintf("/resume/%d", i),
						Stime:       time.Now().UnixMilli() + int64(i*1000), // 确保不同的时间
						Status:      domain.StatusActive,
					})
					require.NoError(t, err)
					ids = append(ids, id)
					time.Sleep(time.Millisecond) // 确保时间差异
				}
				return ids, uid
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.ListReq{
				Offset: 0,
				Limit:  10,
			},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[ginx.DataList[web.Journey]])
				return assert.Len(t, r.Data.List, 3) && assert.Equal(t, 3, r.Data.Total)
			},
			after: func(t *testing.T, resp ginx.DataList[web.Journey]) {
				t.Helper()
				assert.Len(t, resp.List, 3)
				assert.Equal(t, 3, resp.Total)
				// 验证按更新时间倒序排列（最新的在前面）
				assert.Equal(t, "公司C", resp.List[0].CompanyName)
				assert.Equal(t, "公司B", resp.List[1].CompanyName)
				assert.Equal(t, "公司A", resp.List[2].CompanyName)
			},
		},
		{
			name: "分页查询第一页",
			before: func(t *testing.T) (journeyIDs []int64, uid int64) {
				t.Helper()
				var ids []int64
				uid = testID6
				for i := 0; i < 5; i++ {
					id, _, err := s.svc.Save(t.Context(), domain.InterviewJourney{
						Uid:         uid,
						CompanyName: fmt.Sprintf("分页公司%d", i),
						JobInfo:     fmt.Sprintf("/jobinfo/page/%d", i),
						ResumeURL:   fmt.Sprintf("/resume/page/%d", i),
						Stime:       time.Now().UnixMilli() + int64(i*1000),
						Status:      domain.StatusActive,
					})
					require.NoError(t, err)
					ids = append(ids, id)
					time.Sleep(time.Millisecond)
				}
				return ids, uid
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.ListReq{
				Offset: 0,
				Limit:  2,
			},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[ginx.DataList[web.Journey]])
				return assert.Len(t, r.Data.List, 2) && assert.Equal(t, 5, r.Data.Total)
			},
			after: func(t *testing.T, resp ginx.DataList[web.Journey]) {
				t.Helper()
				assert.Len(t, resp.List, 2)
				assert.Equal(t, 5, resp.Total)
			},
		},
		{
			name: "分页查询第二页",
			before: func(t *testing.T) (journeyIDs []int64, uid int64) {
				t.Helper()
				var ids []int64
				uid = testID7
				for i := 0; i < 5; i++ {
					id, _, err := s.svc.Save(t.Context(), domain.InterviewJourney{
						Uid:         uid,
						CompanyName: fmt.Sprintf("分页公司2_%d", i),
						JobInfo:     fmt.Sprintf("/jobinfo/page2/%d", i),
						ResumeURL:   fmt.Sprintf("/resume/page2/%d", i),
						Stime:       time.Now().UnixMilli() + int64(i*1000),
						Status:      domain.StatusActive,
					})
					require.NoError(t, err)
					ids = append(ids, id)
					time.Sleep(time.Millisecond)
				}
				return ids, uid
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req: web.ListReq{
				Offset: 2,
				Limit:  2,
			},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[ginx.DataList[web.Journey]])
				return assert.Len(t, r.Data.List, 2) && assert.Equal(t, 5, r.Data.Total)
			},
			after: func(t *testing.T, resp ginx.DataList[web.Journey]) {
				t.Helper()
				assert.Len(t, resp.List, 2)
				assert.Equal(t, 5, resp.Total)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			_, uid := tc.before(t)

			req, err := http.NewRequest(http.MethodPost,
				"/interview-journeys/list", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[ginx.DataList[web.Journey]]()
			server := s.newGinServer(uid, tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			result := recorder.MustScan()
			tc.respAssertFunc(t, result)
			tc.after(t, result.Data)
		})
	}
}

func (s *InterviewModuleTestSuite) TestHandler_Detail() {
	t := s.T()

	testCases := []struct {
		name           string
		before         func(t *testing.T) int64 // 返回创建的journey ID
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) ginx.Handler
		req            web.DetailReq

		wantCode       int
		respAssertFunc assert.ValueAssertionFunc
		after          func(t *testing.T, resp web.Journey)
	}{
		{
			name: "成功获取详情-无面试轮次",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, _, err := s.svc.Save(t.Context(), domain.InterviewJourney{
					Uid:         testID3,
					CompanyName: "无轮次公司",
					JobInfo:     "/jobinfo/no-rounds",
					ResumeURL:   "/resume/no-rounds",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusActive,
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req:      web.DetailReq{},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[web.Journey])
				return assert.Equal(t, "无轮次公司", r.Data.CompanyName) &&
					assert.Empty(t, r.Data.Rounds)
			},
			after: func(t *testing.T, resp web.Journey) {
				t.Helper()
				assert.Equal(t, "无轮次公司", resp.CompanyName)
				assert.Equal(t, "/jobinfo/no-rounds", resp.JobInfo)
				assert.Equal(t, "/resume/no-rounds", resp.ResumeURL)
				assert.Equal(t, domain.StatusActive.String(), resp.Status)
				assert.Empty(t, resp.Rounds)
			},
		},
		{
			name: "成功获取详情-包含单个面试轮次",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, _, err := s.svc.Save(t.Context(), domain.InterviewJourney{
					Uid:         testID3,
					CompanyName: "单轮面试公司",
					JobInfo:     "/jobinfo/single-round",
					ResumeURL:   "/resume/single-round",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusActive,
					Rounds: []domain.InterviewRound{
						{
							Uid:           testID3,
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/round1",
							ResumeURL:     "/resume/round1",
							AudioURL:      "/audio/round1",
							SelfResult:    true,
							SelfSummary:   "表现良好",
							Result:        domain.ResultPending,
							AllowSharing:  false,
						},
					},
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req:      web.DetailReq{},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[web.Journey])
				return assert.Equal(t, "单轮面试公司", r.Data.CompanyName) &&
					assert.Len(t, r.Data.Rounds, 1)
			},
			after: func(t *testing.T, resp web.Journey) {
				t.Helper()
				assert.Equal(t, "单轮面试公司", resp.CompanyName)
				assert.Len(t, resp.Rounds, 1)
				round := resp.Rounds[0]
				assert.Equal(t, 1, round.RoundNumber)
				assert.Equal(t, "技术1面", round.RoundType)
				assert.Equal(t, "/jobinfo/round1", round.JobInfo)
				assert.Equal(t, "/resume/round1", round.ResumeURL)
				assert.Equal(t, "/audio/round1", round.AudioURL)
				assert.Equal(t, true, round.SelfResult)
				assert.Equal(t, "表现良好", round.SelfSummary)
				assert.Equal(t, domain.ResultPending.String(), round.Result)
				assert.Equal(t, false, round.AllowSharing)
			},
		},
		{
			name: "成功获取详情-包含多个面试轮次",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, _, err := s.svc.Save(t.Context(), domain.InterviewJourney{
					Uid:         testID3,
					CompanyName: "多轮面试公司",
					JobInfo:     "/jobinfo/multi-rounds",
					ResumeURL:   "/resume/multi-rounds",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusActive,
					Rounds: []domain.InterviewRound{
						{
							Uid:           testID3,
							RoundNumber:   1,
							RoundType:     "技术1面",
							InterviewDate: time.Now().UnixMilli(),
							JobInfo:       "/jobinfo/round1",
							ResumeURL:     "/resume/round1",
							AudioURL:      "/audio/round1",
							SelfResult:    true,
							SelfSummary:   "第一轮表现良好",
							Result:        domain.ResultApproved,
							AllowSharing:  false,
						},
						{
							Uid:           testID3,
							RoundNumber:   2,
							RoundType:     "技术2面",
							InterviewDate: time.Now().UnixMilli() + 1000,
							JobInfo:       "/jobinfo/round2",
							ResumeURL:     "/resume/round2",
							AudioURL:      "/audio/round2",
							SelfResult:    false,
							SelfSummary:   "第二轮有待提高",
							Result:        domain.ResultPending,
							AllowSharing:  true,
						},
					},
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req:      web.DetailReq{},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[web.Journey])
				return assert.Equal(t, "多轮面试公司", r.Data.CompanyName) &&
					assert.Len(t, r.Data.Rounds, 2)
			},
			after: func(t *testing.T, resp web.Journey) {
				t.Helper()
				assert.Equal(t, "多轮面试公司", resp.CompanyName)
				assert.Len(t, resp.Rounds, 2)

				// 验证第一轮
				round1 := resp.Rounds[0]
				assert.Equal(t, 1, round1.RoundNumber)
				assert.Equal(t, "技术1面", round1.RoundType)
				assert.Equal(t, domain.ResultApproved.String(), round1.Result)
				assert.Equal(t, false, round1.AllowSharing)

				// 验证第二轮
				round2 := resp.Rounds[1]
				assert.Equal(t, 2, round2.RoundNumber)
				assert.Equal(t, "技术2面", round2.RoundType)
				assert.Equal(t, domain.ResultPending.String(), round2.Result)
				assert.Equal(t, true, round2.AllowSharing)
			},
		},
		{
			name: "记录不存在测试",
			before: func(t *testing.T) int64 {
				t.Helper()
				// 不创建任何记录，直接返回一个不存在的ID
				return 999999
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req:      web.DetailReq{},
			wantCode: 500,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[any])
				return assert.Equal(t, test.Result[any]{Code: 519001, Msg: "系统错误"}, r)
			},
			after: func(t *testing.T, resp web.Journey) {
				t.Helper()
				// 错误情况下，不会有有效的响应数据
			},
		},
		{
			name: "不同状态的面试历程-成功状态",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, _, err := s.svc.Save(t.Context(), domain.InterviewJourney{
					Uid:         testID3,
					CompanyName: "成功状态公司",
					JobInfo:     "/jobinfo/succeeded",
					ResumeURL:   "/resume/succeeded",
					Stime:       time.Now().UnixMilli(),
					Etime:       time.Now().UnixMilli() + 86400000, // 结束时间
					Status:      domain.StatusSucceeded,
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req:      web.DetailReq{},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[web.Journey])
				return assert.Equal(t, "成功状态公司", r.Data.CompanyName) &&
					assert.Equal(t, domain.StatusSucceeded.String(), r.Data.Status)
			},
			after: func(t *testing.T, resp web.Journey) {
				t.Helper()
				assert.Equal(t, "成功状态公司", resp.CompanyName)
				assert.Equal(t, domain.StatusSucceeded.String(), resp.Status)
				assert.Greater(t, resp.Etime, int64(0)) // 验证有结束时间
			},
		},
		{
			name: "不同状态的面试历程-失败状态",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, _, err := s.svc.Save(t.Context(), domain.InterviewJourney{
					Uid:         testID3,
					CompanyName: "失败状态公司",
					JobInfo:     "/jobinfo/failed",
					ResumeURL:   "/resume/failed",
					Stime:       time.Now().UnixMilli(),
					Status:      domain.StatusFailed,
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.svc)
			},
			req:      web.DetailReq{},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[web.Journey])
				return assert.Equal(t, "失败状态公司", r.Data.CompanyName) &&
					assert.Equal(t, domain.StatusFailed.String(), r.Data.Status)
			},
			after: func(t *testing.T, resp web.Journey) {
				t.Helper()
				assert.Equal(t, "失败状态公司", resp.CompanyName)
				assert.Equal(t, domain.StatusFailed.String(), resp.Status)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			jid := tc.before(t)
			tc.req.ID = jid

			req, err := http.NewRequest(http.MethodPost,
				"/interview-journeys/detail", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")

			// 根据预期状态码选择不同的录制器类型
			if tc.wantCode == 200 {
				recorder := test.NewJSONResponseRecorder[web.Journey]()
				server := s.newGinServer(testID3, tc.newHandlerFunc(t, ctrl))
				server.ServeHTTP(recorder, req)
				require.Equal(t, tc.wantCode, recorder.Code)
				result := recorder.MustScan()
				tc.respAssertFunc(t, result)
				tc.after(t, result.Data)
			} else {
				recorder := test.NewJSONResponseRecorder[any]()
				server := s.newGinServer(testID3, tc.newHandlerFunc(t, ctrl))
				server.ServeHTTP(recorder, req)
				require.Equal(t, tc.wantCode, recorder.Code)
				result := recorder.MustScan()
				tc.respAssertFunc(t, result)
				tc.after(t, web.Journey{}) // 错误情况传空对象
			}
		})
	}
}
