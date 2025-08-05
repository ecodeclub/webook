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
	db         *egorm.Component
	journeySvc service.InterviewJourneyService
	roundSvc   service.InterviewRoundService
}

func (s *InterviewModuleTestSuite) SetupSuite() {
	s.db = testioc.InitDB()
	s.NoError(dao.InitTables(s.db))
	m := startup.InitModule(s.db)
	s.journeySvc = m.JourneySvc
	s.roundSvc = m.RoundSvc
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
	s.NoError(s.db.Exec("TRUNCATE TABLE `interview_journeys`").Error)
	s.NoError(s.db.Exec("TRUNCATE TABLE `interview_rounds`").Error)
}

func (s *InterviewModuleTestSuite) TestRoundHandler_Create() {
	t := s.T()

	testCases := []struct {
		name           string
		before         func(t *testing.T) (jid int64)
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) ginx.Handler
		req            web.CreateRoundReq

		wantCode       int
		respAssertFunc assert.ValueAssertionFunc
		after          func(t *testing.T, rid int64, req web.CreateRoundReq)
	}{
		{
			name: "创建轮数成功",
			before: func(t *testing.T) (jid int64) {
				t.Helper()
				id, err := s.journeySvc.Create(t.Context(), domain.InterviewJourney{
					Uid:         testID,
					CompanyName: "company-name-1",
					JobInfo:     "/jobinfo/1",
					ResumeURL:   "/resume/1",
					Stime:       time.Now().UnixMilli(),
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewRoundHandler(s.roundSvc)
			},
			req: web.CreateRoundReq{
				Round: web.Round{
					RoundNumber:   1,
					RoundType:     "技术1面",
					InterviewDate: time.Now().UnixMilli(),
					JobInfo:       "/jobinfo/1",
					ResumeURL:     "/resume/1",
					AudioURL:      "/audio/1",
					SelfResult:    true,
					SelfSummary:   "good",
					Result:        "PENDING",
					AllowSharing:  false,
				},
			},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[int64])
				return assert.Positive(t, r.Data)
			},
			after: func(t *testing.T, rid int64, req web.CreateRoundReq) {
				t.Helper()
				actual, err := s.roundSvc.FindByID(t.Context(), rid, req.Round.Jid, testID)
				require.NoError(t, err)
				s.assertRound(t, rid, testID, req.Round, actual)
			},
		},
		{
			name: "创建轮数失败_轮数编号不唯一",
			before: func(t *testing.T) (jid int64) {
				t.Helper()
				id, err := s.journeySvc.Create(t.Context(), domain.InterviewJourney{
					Uid:         testID,
					CompanyName: "company-name-2",
					JobInfo:     "/jobinfo/2",
					ResumeURL:   "/resume/2",
					Stime:       time.Now().UnixMilli(),
				})
				require.NoError(t, err)

				_, err = s.roundSvc.Create(t.Context(), domain.InterviewRound{
					Jid:           id,
					Uid:           testID,
					RoundNumber:   2,
					RoundType:     "技术二面",
					InterviewDate: time.Now().UnixMilli(),
					JobInfo:       "/jobinfo/2",
					ResumeURL:     "/resume/2",
					AudioURL:      "/audio/2",
					SelfResult:    true,
					SelfSummary:   "good",
					Result:        "REJECTED",
					AllowSharing:  false,
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewRoundHandler(s.roundSvc)
			},
			req: web.CreateRoundReq{
				Round: web.Round{
					RoundNumber:   2,
					RoundType:     "技术二面",
					InterviewDate: time.Now().UnixMilli(),
					JobInfo:       "/jobinfo/2",
					ResumeURL:     "/resume/2",
					AudioURL:      "/audio/2",
					SelfResult:    true,
					SelfSummary:   "good",
					Result:        "REJECTED",
					AllowSharing:  false,
				},
			},
			wantCode: 500,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				return assert.Equal(t, test.Result[int64]{
					Code: 519001, Msg: "系统错误",
				}, i)
			},
			after: func(t *testing.T, rid int64, req web.CreateRoundReq) {
				t.Helper()
			},
		},
		{
			name: "创建轮数失败_官方结果非法",
			before: func(t *testing.T) (jid int64) {
				t.Helper()
				id, err := s.journeySvc.Create(t.Context(), domain.InterviewJourney{
					Uid:         testID,
					CompanyName: "company-name-3",
					JobInfo:     "/jobinfo/3",
					ResumeURL:   "/resume/3",
					Stime:       time.Now().UnixMilli(),
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewRoundHandler(s.roundSvc)
			},
			req: web.CreateRoundReq{
				Round: web.Round{
					RoundNumber:   3,
					RoundType:     "技术3面",
					InterviewDate: time.Now().UnixMilli(),
					JobInfo:       "/jobinfo/3",
					ResumeURL:     "/resume/3",
					AudioURL:      "/audio/3",
					SelfResult:    true,
					SelfSummary:   "good",
					Result:        "",
					AllowSharing:  false,
				},
			},
			wantCode: 500,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				return assert.Equal(t, test.Result[int64]{
					Code: 519001, Msg: "系统错误",
				}, i)
			},
			after: func(t *testing.T, rid int64, req web.CreateRoundReq) {
				t.Helper()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			jid := tc.before(t)

			tc.req.Round.Jid = jid

			req, err := http.NewRequest(http.MethodPost,
				"/interview-rounds/create", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[int64]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			result := recorder.MustScan()
			tc.respAssertFunc(t, result)
			tc.after(t, result.Data, tc.req)
		})
	}
}

func (s *InterviewModuleTestSuite) assertRound(t *testing.T, rid, uid int64, expected web.Round, actual domain.InterviewRound) {
	t.Helper()
	assert.Equal(t, rid, actual.ID)
	assert.Equal(t, expected.Jid, actual.Jid)
	assert.Equal(t, uid, actual.Uid)
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

func (s *InterviewModuleTestSuite) TestRoundHandler_Update() {
	t := s.T()

	testCases := []struct {
		name           string
		before         func(t *testing.T) (jid, rid int64)
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) ginx.Handler
		req            web.UpdateRoundReq

		wantCode       int
		respAssertFunc assert.ValueAssertionFunc
		after          func(t *testing.T, req web.UpdateRoundReq)
	}{
		{
			name: "更新轮数成功",
			before: func(t *testing.T) (jid, rid int64) {
				t.Helper()
				jid, err := s.journeySvc.Create(t.Context(), domain.InterviewJourney{
					Uid:         testID,
					CompanyName: "company-name-4",
					JobInfo:     "/jobinfo/4",
					ResumeURL:   "/resume/4",
					Stime:       time.Now().UnixMilli(),
				})
				require.NoError(t, err)

				rid, err = s.roundSvc.Create(t.Context(), domain.InterviewRound{
					Jid:           jid,
					Uid:           testID,
					RoundNumber:   4,
					RoundType:     "技术4面",
					InterviewDate: time.Now().UnixMilli(),
					JobInfo:       "/jobinfo/4",
					ResumeURL:     "/resume/4",
					AudioURL:      "/audio/4",
					SelfResult:    true,
					SelfSummary:   "good",
					Result:        "REJECTED",
					AllowSharing:  false,
				})
				require.NoError(t, err)
				return
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewRoundHandler(s.roundSvc)
			},
			req: web.UpdateRoundReq{
				Round: web.Round{
					RoundNumber:   5,
					RoundType:     "技术5面",
					InterviewDate: time.Now().UnixMilli(),
					JobInfo:       "/jobinfo/5",
					ResumeURL:     "/resume/5",
					AudioURL:      "/audio/5",
					SelfResult:    true,
					SelfSummary:   "good",
					Result:        "REJECTED",
					AllowSharing:  false,
				},
			},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				return assert.Equal(t, test.Result[any]{Msg: "OK"}, i)
			},
			after: func(t *testing.T, req web.UpdateRoundReq) {
				t.Helper()
				actual, err := s.roundSvc.FindByID(t.Context(), req.Round.ID, req.Round.Jid, testID)
				require.NoError(t, err)
				s.assertRound(t, req.Round.ID, testID, req.Round, actual)
			},
		},
		{
			name: "更新轮数失败_不可撤销授权",
			before: func(t *testing.T) (jid, rid int64) {
				t.Helper()
				jid, err := s.journeySvc.Create(t.Context(), domain.InterviewJourney{
					Uid:         testID,
					CompanyName: "company-name-5",
					JobInfo:     "/jobinfo/5",
					ResumeURL:   "/resume/5",
					Stime:       time.Now().UnixMilli(),
				})
				require.NoError(t, err)

				rid, err = s.roundSvc.Create(t.Context(), domain.InterviewRound{
					Jid:           jid,
					Uid:           testID,
					RoundNumber:   5,
					RoundType:     "技术5面",
					InterviewDate: time.Now().UnixMilli(),
					JobInfo:       "/jobinfo/5",
					ResumeURL:     "/resume/5",
					AudioURL:      "/audio/5",
					SelfResult:    true,
					SelfSummary:   "good",
					Result:        "REJECTED",
					AllowSharing:  true,
				})
				require.NoError(t, err)
				return
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewRoundHandler(s.roundSvc)
			},
			req: web.UpdateRoundReq{
				Round: web.Round{
					RoundNumber:   5,
					RoundType:     "技术5面",
					InterviewDate: time.Now().UnixMilli(),
					JobInfo:       "/jobinfo/5",
					ResumeURL:     "/resume/5",
					AudioURL:      "/audio/5",
					SelfResult:    true,
					SelfSummary:   "good",
					Result:        "REJECTED",
					AllowSharing:  false,
				},
			},
			wantCode: 500,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				return assert.Equal(t, test.Result[any]{
					Code: 519001, Msg: "系统错误",
				}, i)
			},
			after: func(t *testing.T, req web.UpdateRoundReq) {
				t.Helper()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			jid, rid := tc.before(t)

			tc.req.Round.Jid = jid
			tc.req.Round.ID = rid

			req, err := http.NewRequest(http.MethodPost,
				"/interview-rounds/update", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[any]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.respAssertFunc(t, recorder.MustScan())
			tc.after(t, tc.req)
		})
	}
}

/*
func (s *InterviewModuleTestSuite) TestJourneyHandler_Create() {
	t := s.T()

	testCases := []struct {
		name           string
		before         func(t *testing.T) (jid int64)
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) ginx.Handler
		req            web.CreateJourneyReq

		wantCode       int
		respAssertFunc assert.ValueAssertionFunc
		after          func(t *testing.T, rid int64, req web.CreateJourneyReq)
	}{
		{
			name: "创建旅程成功",
			before: func(t *testing.T) (jid int64) {
				t.Helper()
				id, err := s.journeySvc.Create(t.Context(), domain.InterviewJourney{
					Uid:         testID,
					CompanyName: "company-name-6",
					JobInfo:     "/jobinfo/6",
					ResumeURL:   "/resume/6",
					Stime:       time.Now().UnixMilli(),
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewJourneyHandler(s.journeySvc)
			},
			req: web.CreateJourneyReq{
				Journey: web.Journey{
					ID:          0,
					CompanyID:   0,
					CompanyName: "",
					JobInfo:     "",
					ResumeURL:   "",
					Status:      "",
					Stime:       0,
					Etime:       0,
					Rounds:      nil,
				},
			},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				r := i.(test.Result[int64])
				return assert.Positive(t, r.Data)
			},
			after: func(t *testing.T, rid int64, req web.CreateJourneyReq) {
				t.Helper()
				actual, err := s.roundSvc.FindByID(t.Context(), rid, req.Round.Jid, testID)
				require.NoError(t, err)
				s.assertRound(t, rid, testID, req.Round, actual)
			},
		},
		{
			name: "创建轮数失败_轮数编号不唯一",
			before: func(t *testing.T) (jid int64) {
				t.Helper()
				id, err := s.journeySvc.Create(t.Context(), domain.InterviewJourney{
					Uid:         testID,
					CompanyName: "company-name-2",
					JobInfo:     "/jobinfo/2",
					ResumeURL:   "/resume/2",
					Stime:       time.Now().UnixMilli(),
				})
				require.NoError(t, err)

				_, err = s.roundSvc.Create(t.Context(), domain.InterviewRound{
					Jid:           id,
					Uid:           testID,
					RoundNumber:   2,
					RoundType:     "技术二面",
					InterviewDate: time.Now().UnixMilli(),
					JobInfo:       "/jobinfo/2",
					ResumeURL:     "/resume/2",
					AudioURL:      "/audio/2",
					SelfResult:    true,
					SelfSummary:   "good",
					Result:        "REJECTED",
					AllowSharing:  false,
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewRoundHandler(s.roundSvc)
			},
			req: web.CreateRoundReq{
				Round: web.Round{
					RoundNumber:   2,
					RoundType:     "技术二面",
					InterviewDate: time.Now().UnixMilli(),
					JobInfo:       "/jobinfo/2",
					ResumeURL:     "/resume/2",
					AudioURL:      "/audio/2",
					SelfResult:    true,
					SelfSummary:   "good",
					Result:        "REJECTED",
					AllowSharing:  false,
				},
			},
			wantCode: 500,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				return assert.Equal(t, test.Result[int64]{
					Code: 519001, Msg: "系统错误",
				}, i)
			},
			after: func(t *testing.T, rid int64, req web.CreateRoundReq) {
				t.Helper()
			},
		},
		{
			name: "创建轮数失败_官方结果非法",
			before: func(t *testing.T) (jid int64) {
				t.Helper()
				id, err := s.journeySvc.Create(t.Context(), domain.InterviewJourney{
					Uid:         testID,
					CompanyName: "company-name-3",
					JobInfo:     "/jobinfo/3",
					ResumeURL:   "/resume/3",
					Stime:       time.Now().UnixMilli(),
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewRoundHandler(s.roundSvc)
			},
			req: web.CreateRoundReq{
				Round: web.Round{
					RoundNumber:   3,
					RoundType:     "技术3面",
					InterviewDate: time.Now().UnixMilli(),
					JobInfo:       "/jobinfo/3",
					ResumeURL:     "/resume/3",
					AudioURL:      "/audio/3",
					SelfResult:    true,
					SelfSummary:   "good",
					Result:        "",
					AllowSharing:  false,
				},
			},
			wantCode: 500,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				return assert.Equal(t, test.Result[int64]{
					Code: 519001, Msg: "系统错误",
				}, i)
			},
			after: func(t *testing.T, rid int64, req web.CreateRoundReq) {
				t.Helper()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			jid := tc.before(t)

			tc.req.Round.Jid = jid

			req, err := http.NewRequest(http.MethodPost,
				"/interview-rounds/create", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[int64]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			result := recorder.MustScan()
			tc.respAssertFunc(t, result)
			tc.after(t, result.Data, tc.req)
		})
	}
}

func (s *InterviewModuleTestSuite) assertJourney(t *testing.T, rid, uid int64, expected web.Round, actual domain.InterviewRound) {
	t.Helper()
	assert.Equal(t, rid, actual.ID)
	assert.Equal(t, expected.Jid, actual.Jid)
	assert.Equal(t, uid, actual.Uid)
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

func (s *InterviewModuleTestSuite) TestJourneyHandler_Update() {
	t := s.T()

	testCases := []struct {
		name           string
		before         func(t *testing.T) (jid, rid int64)
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) ginx.Handler
		req            web.UpdateRoundReq

		wantCode       int
		respAssertFunc assert.ValueAssertionFunc
		after          func(t *testing.T, req web.UpdateRoundReq)
	}{
		{
			name: "更新轮数成功",
			before: func(t *testing.T) (jid, rid int64) {
				t.Helper()
				jid, err := s.journeySvc.Create(t.Context(), domain.InterviewJourney{
					Uid:         testID,
					CompanyName: "company-name-4",
					JobInfo:     "/jobinfo/4",
					ResumeURL:   "/resume/4",
					Stime:       time.Now().UnixMilli(),
				})
				require.NoError(t, err)

				rid, err = s.roundSvc.Create(t.Context(), domain.InterviewRound{
					Jid:           jid,
					Uid:           testID,
					RoundNumber:   4,
					RoundType:     "技术4面",
					InterviewDate: time.Now().UnixMilli(),
					JobInfo:       "/jobinfo/4",
					ResumeURL:     "/resume/4",
					AudioURL:      "/audio/4",
					SelfResult:    true,
					SelfSummary:   "good",
					Result:        "REJECTED",
					AllowSharing:  false,
				})
				t.Logf("jid = %d, rid = %d\n", jid, rid)
				require.NoError(t, err)
				return
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewRoundHandler(s.roundSvc)
			},
			req: web.UpdateRoundReq{
				Round: web.Round{
					RoundNumber:   5,
					RoundType:     "技术5面",
					InterviewDate: time.Now().UnixMilli(),
					JobInfo:       "/jobinfo/5",
					ResumeURL:     "/resume/5",
					AudioURL:      "/audio/5",
					SelfResult:    true,
					SelfSummary:   "good",
					Result:        "REJECTED",
					AllowSharing:  false,
				},
			},
			wantCode: 200,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				return assert.Equal(t, test.Result[any]{Msg: "OK"}, i)
			},
			after: func(t *testing.T, req web.UpdateRoundReq) {
				t.Helper()
				actual, err := s.roundSvc.FindByID(t.Context(), req.Round.ID, req.Round.Jid, testID)
				require.NoError(t, err)
				s.assertRound(t, req.Round.ID, testID, req.Round, actual)
			},
		},
		{
			name: "更新轮数失败_不可撤销授权",
			before: func(t *testing.T) (jid, rid int64) {
				t.Helper()
				jid, err := s.journeySvc.Create(t.Context(), domain.InterviewJourney{
					Uid:         testID,
					CompanyName: "company-name-4",
					JobInfo:     "/jobinfo/4",
					ResumeURL:   "/resume/4",
					Stime:       time.Now().UnixMilli(),
				})
				require.NoError(t, err)

				rid, err = s.roundSvc.Create(t.Context(), domain.InterviewRound{
					Jid:           jid,
					Uid:           testID,
					RoundNumber:   4,
					RoundType:     "技术4面",
					InterviewDate: time.Now().UnixMilli(),
					JobInfo:       "/jobinfo/4",
					ResumeURL:     "/resume/4",
					AudioURL:      "/audio/4",
					SelfResult:    true,
					SelfSummary:   "good",
					Result:        "REJECTED",
					AllowSharing:  true,
				})
				t.Logf("jid = %d, rid = %d\n", jid, rid)
				require.NoError(t, err)
				return
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) ginx.Handler {
				t.Helper()
				return web.NewInterviewRoundHandler(s.roundSvc)
			},
			req: web.UpdateRoundReq{
				Round: web.Round{
					RoundNumber:   5,
					RoundType:     "技术5面",
					InterviewDate: time.Now().UnixMilli(),
					JobInfo:       "/jobinfo/5",
					ResumeURL:     "/resume/5",
					AudioURL:      "/audio/5",
					SelfResult:    true,
					SelfSummary:   "good",
					Result:        "REJECTED",
					AllowSharing:  false,
				},
			},
			wantCode: 500,
			respAssertFunc: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				return assert.Equal(t, test.Result[any]{
					Code: 519001, Msg: "系统错误",
				}, i)
			},
			after: func(t *testing.T, req web.UpdateRoundReq) {
				t.Helper()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			jid, rid := tc.before(t)

			tc.req.Round.Jid = jid
			tc.req.Round.ID = rid

			req, err := http.NewRequest(http.MethodPost,
				"/interview-rounds/update", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[any]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.respAssertFunc(t, recorder.MustScan())
			tc.after(t, tc.req)
		})
	}
}
*/
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
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).First(&interview).Error)
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
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).First(&interview).Error)
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
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).First(&interview).Error)
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
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).First(&interview).Error)
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
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).First(&interview).Error)
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
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).First(&interview).Error)
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
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).First(&interview).Error)
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
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).First(&interview).Error)
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
