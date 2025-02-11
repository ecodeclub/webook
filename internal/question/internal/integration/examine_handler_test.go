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
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ecodeclub/webook/internal/member"

	"github.com/ecodeclub/webook/internal/ai"
	aimocks "github.com/ecodeclub/webook/internal/ai/mocks"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/webook/internal/permission"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/question/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/question/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ExamineHandlerTest struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
	dao    dao.ExamineDAO
}

func (s *ExamineHandlerTest) SetupSuite() {
	ctrl := gomock.NewController(s.T())
	aiSvc := aimocks.NewMockService(ctrl)
	aiSvc.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, req ai.LLMRequest) (ai.LLMResponse, error) {
		return ai.LLMResponse{
			Tokens: req.Uid,
			Amount: req.Uid,
			Answer: "最终评分 \n 1",
		}, nil
	}).AnyTimes()
	module, err := startup.InitModule(nil, nil, &interactive.Module{},
		&permission.Module{}, &ai.Module{Svc: aiSvc},
		session.DefaultProvider(),
		&member.Module{})
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
	err = s.db.Create(&dao.PublishQuestion{
		Id:    1,
		Title: "测试题目1",
	}).Error
	assert.NoError(s.T(), err)
	err = s.db.Create(&dao.PublishQuestion{
		Id:    2,
		Title: "测试题目2",
	}).Error
	assert.NoError(s.T(), err)
}

func (s *ExamineHandlerTest) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `question_results`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `examine_records`").Error
	require.NoError(s.T(), err)
}

func (s *ExamineHandlerTest) TearDownSuite() {
	err := s.db.Exec("TRUNCATE TABLE `publish_questions`").Error
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
				var record dao.ExamineRecord
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
				assert.Equal(t, dao.ExamineRecord{
					Uid:       uid,
					Qid:       1,
					Result:    domain.ResultBasic.ToUint8(),
					RawResult: "最终评分 \n 1\n\n最终评分 \n 1",
					Tokens:    369,
					Amount:    369,
				}, record)

				var queRes dao.QuestionResult
				err = s.db.WithContext(ctx).
					Where("qid = ? AND uid = ?", 1, uid).
					First(&queRes).Error
				require.NoError(t, err)
				assert.True(t, queRes.Ctime > 0)
				queRes.Ctime = 0
				assert.True(t, queRes.Utime > 0)
				queRes.Utime = 0
				assert.True(t, queRes.Id > 0)
				queRes.Id = 0
				assert.Equal(t, dao.QuestionResult{
					Result: domain.ResultBasic.ToUint8(),
					Qid:    1,
					Uid:    uid,
				}, queRes)
			},
			req: web.ExamineReq{
				Qid:   1,
				Input: "测试一下",
			},
			wantCode: 200,
			wantResp: test.Result[web.ExamineResult]{
				Data: web.ExamineResult{
					Result:    domain.ResultBasic.ToUint8(),
					RawResult: "最终评分 \n 1\n\n最终评分 \n 1",
					Amount:    369,
				},
			},
		},
		{
			// 这个测试依赖于前面的测试产生的 eid = 1
			name: "老用户重复测试",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.QuestionResult{
					Id:     2,
					Uid:    uid,
					Qid:    2,
					Result: domain.ResultIntermediate.ToUint8(),
					Ctime:  123,
					Utime:  123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				const qid = 2
				var record dao.ExamineRecord
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
				assert.Equal(t, dao.ExamineRecord{
					Uid:       uid,
					Qid:       2,
					Result:    domain.ResultBasic.ToUint8(),
					RawResult: "最终评分 \n 1\n\n最终评分 \n 1",
					Tokens:    369,
					Amount:    369,
				}, record)

				var queRes dao.QuestionResult
				err = s.db.WithContext(ctx).
					Where("qid = ? AND uid = ?", 2, uid).
					First(&queRes).Error
				require.NoError(t, err)
				assert.True(t, queRes.Ctime > 0)
				queRes.Ctime = 0
				assert.True(t, queRes.Utime > 0)
				queRes.Utime = 0
				assert.True(t, queRes.Id > 0)
				queRes.Id = 0
				assert.Equal(t, dao.QuestionResult{
					Result: domain.ResultBasic.ToUint8(),
					Qid:    qid,
					Uid:    uid,
				}, queRes)
			},
			wantCode: 200,
			req: web.ExamineReq{
				Qid:   2,
				Input: "测试一下",
			},
			wantResp: test.Result[web.ExamineResult]{
				Data: web.ExamineResult{
					Result:    domain.ResultBasic.ToUint8(),
					RawResult: "最终评分 \n 1\n\n最终评分 \n 1",
					Amount:    369,
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/question/examine", iox.NewJSONReader(tc.req))
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

func (s *ExamineHandlerTest) TestCorrect() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		req web.CorrectReq

		wantCode int
		wantResp test.Result[web.ExamineResult]
	}{
		{
			name: "修改成通过",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.QuestionResult{
					Id:     2,
					Uid:    uid,
					Qid:    1,
					Result: domain.ResultFailed.ToUint8(),
					Ctime:  123,
					Utime:  123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
				defer cancel()
				var record dao.QuestionResult
				err := s.db.WithContext(ctx).Where("uid = ? AND qid = ?", uid, 1).First(&record).Error

				require.NoError(t, err)
				assert.True(t, record.Utime > 0)
				record.Utime = 0
				assert.True(t, record.Ctime > 0)
				record.Ctime = 0
				assert.True(t, record.Id > 0)
				record.Id = 0

				assert.Equal(t, dao.QuestionResult{
					Uid:    uid,
					Qid:    1,
					Result: domain.ResultBasic.ToUint8(),
				}, record)

			},
			req: web.CorrectReq{
				Qid:    1,
				Result: domain.ResultBasic.ToUint8(),
			},
			wantCode: 200,
			wantResp: test.Result[web.ExamineResult]{
				Data: web.ExamineResult{
					Qid:    1,
					Result: domain.ResultBasic.ToUint8(),
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/question/examine/correct", iox.NewJSONReader(tc.req))
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
