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

	evtmocks "github.com/ecodeclub/webook/internal/feedback/internal/event/mocks"
	"github.com/ecodeclub/webook/internal/feedback/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/feedback/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/feedback/internal/web"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"

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

const uid = 2051

type HandlerTestSuite struct {
	suite.Suite
	server   *egin.Component
	db       *egorm.Component
	dao      dao.FeedbackDAO
	ctrl     *gomock.Controller
	producer *evtmocks.MockIncreaseCreditsEventProducer
}

func (s *HandlerTestSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `feedbacks`").Error
	require.NoError(s.T(), err)

	s.ctrl.Finish()
}

func (s *HandlerTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `feedbacks`").Error
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())
	s.producer = evtmocks.NewMockIncreaseCreditsEventProducer(s.ctrl)
	handler, err := startup.InitHandler(s.producer)
	require.NoError(s.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid:  uid,
			Data: map[string]string{"creator": "true"},
		}))
	})
	handler.MemberRoutes(server.Engine)
	s.server = server
	s.db = testioc.InitDB()
	s.dao = dao.NewFeedbackDAO(s.db)
}

func (s *HandlerTestSuite) TestCreate() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.CreateReq
		wantCode int
	}{
		{
			name: "新建",
			before: func(t *testing.T) {
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				feedBack, err := s.dao.Info(ctx, 1)
				require.NoError(t, err)
				s.assertFeedBack(t, dao.Feedback{
					UID:     uid,
					Biz:     "case",
					BizID:   1,
					Content: "case写的不行",
					Status:  0,
				}, feedBack)
			},
			req: web.CreateReq{
				Feedback: web.Feedback{
					BizID:   1,
					Biz:     "case",
					Content: "case写的不行",
				},
			},
			wantCode: 200,
		},
	}
	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/feedback/create", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t)
			// 清理 的数据
			err = s.db.Exec("TRUNCATE table `feedbacks`").Error
			require.NoError(t, err)
		})
	}
}

func (s *HandlerTestSuite) TestUpdateStatus() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.UpdateStatusReq
		wantCode int
	}{
		{
			name: "拒绝",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.Feedback{
					ID:      2,
					BizID:   1,
					Biz:     "que",
					UID:     uid,
					Content: "que不行",
					Status:  0,
					Ctime:   123,
					Utime:   321,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				feedBack, err := s.dao.Info(ctx, 2)
				require.NoError(t, err)
				s.assertFeedBack(t, dao.Feedback{
					UID:     uid,
					Biz:     "que",
					BizID:   1,
					Content: "que不行",
					Status:  2,
				}, feedBack)
			},
			req: web.UpdateStatusReq{
				FID:    2,
				Status: 2,
			},
			wantCode: 200,
		},
		{
			name: "采纳",
			before: func(t *testing.T) {
				t.Helper()
				err := s.db.Create(&dao.Feedback{
					ID:      3,
					BizID:   1,
					Biz:     "skill",
					UID:     uid,
					Content: "skill不行",
					Status:  0,
					Ctime:   123,
					Utime:   321,
				}).Error
				require.NoError(t, err)

				s.producer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				feedBack, err := s.dao.Info(ctx, 3)
				require.NoError(t, err)
				s.assertFeedBack(t, dao.Feedback{
					UID:     uid,
					Biz:     "skill",
					BizID:   1,
					Content: "skill不行",
					Status:  1,
				}, feedBack)
			},
			req: web.UpdateStatusReq{
				FID:    3,
				Status: 1,
			},
			wantCode: 200,
		},
	}
	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/feedback/update-status", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t)
			// 清理 的数据
			err = s.db.Exec("TRUNCATE table `feedbacks`").Error
			require.NoError(t, err)
		})
	}
}

func (s *HandlerTestSuite) TestInfo() {
	t := s.T()
	err := s.db.Create(&dao.Feedback{
		ID:      4,
		BizID:   3,
		Biz:     "cases",
		UID:     uid,
		Content: "cases",
		Status:  2,
		Ctime:   1712160000000,
		Utime:   1712246400000,
	}).Error
	actualReq := web.FeedbackID{
		FID: 4,
	}
	req, err := http.NewRequest(http.MethodPost,
		"/feedback/detail", iox.NewJSONReader(actualReq))
	req.Header.Set("content-type", "application/json")
	require.NoError(t, err)
	recorder := test.NewJSONResponseRecorder[web.Feedback]()
	s.server.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
	wantResp := test.Result[web.Feedback]{
		Data: web.Feedback{
			ID:      4,
			BizID:   3,
			Biz:     "cases",
			Content: "cases",
			Status:  2,
		},
	}
	actualResp := recorder.MustScan()
	require.True(t, actualResp.Data.Ctime != "")
	require.True(t, actualResp.Data.Utime != "")
	actualResp.Data.Utime = ""
	actualResp.Data.Ctime = ""
	assert.Equal(t, wantResp, actualResp)
	err = s.db.Exec("TRUNCATE table `feedbacks`").Error
	require.NoError(t, err)
}

func (s *HandlerTestSuite) TestList() {
	data := make([]dao.Feedback, 0, 100)
	for idx := 1; idx < 10; idx++ {
		// 创建采纳的case
		data = append(data, dao.Feedback{
			ID:     int64(idx),
			UID:    uid,
			Biz:    "case",
			BizID:  int64(idx),
			Status: 1,
			Utime:  0,
		})
	}
	for idx := 10; idx < 20; idx++ {
		// 创建未处理的case
		data = append(data, dao.Feedback{
			ID:     int64(idx),
			UID:    uid,
			Biz:    "case",
			BizID:  int64(idx),
			Status: 0,
			Utime:  0,
		})
	}
	err := s.db.Model(&dao.Feedback{}).Create(&data).Error
	require.NoError(s.T(), err)
	testCases := []struct {
		name     string
		req      web.ListReq
		wantResp test.Result[web.FeedbackList]
		wantCode int
	}{
		{
			name: "查看反馈",
			req: web.ListReq{
				Offset: 0,
				Limit:  2,
			},
			wantCode: 200,
			wantResp: test.Result[web.FeedbackList]{
				Data: web.FeedbackList{
					Feedbacks: []web.Feedback{
						{
							ID:     19,
							Biz:    "case",
							BizID:  19,
							Status: 0,
							Utime:  time.UnixMilli(0).Format(time.DateTime),
							Ctime:  time.UnixMilli(0).Format(time.DateTime),
						},
						{
							ID:     18,
							Biz:    "case",
							BizID:  18,
							Status: 0,
							Utime:  time.UnixMilli(0).Format(time.DateTime),
							Ctime:  time.UnixMilli(0).Format(time.DateTime),
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
				"/feedback/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.FeedbackList]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
	// 清理 的数据
	err = s.db.Exec("TRUNCATE table `feedbacks`").Error
	require.NoError(s.T(), err)

}

func (s *HandlerTestSuite) TestPendingCount() {
	t := s.T()
	data := make([]dao.Feedback, 0, 100)
	for idx := 1; idx < 10; idx++ {
		// 创建采纳的case
		data = append(data, dao.Feedback{
			ID:     int64(idx),
			UID:    uid,
			Biz:    "case",
			BizID:  int64(idx),
			Status: 1,
			Utime:  0,
		})
	}
	for idx := 10; idx < 20; idx++ {
		// 创建未处理的case
		data = append(data, dao.Feedback{
			ID:     int64(idx),
			UID:    uid,
			Biz:    "case",
			BizID:  int64(idx),
			Status: 0,
			Utime:  0,
		})
	}
	err := s.db.Model(&dao.Feedback{}).Create(&data).Error
	require.NoError(s.T(), err)
	req, err := http.NewRequest(http.MethodGet,
		"/feedback/pending-count", iox.NewJSONReader(nil))
	req.Header.Set("content-type", "application/json")
	require.NoError(t, err)
	recorder := test.NewJSONResponseRecorder[int64]()
	s.server.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
	require.Equal(t, int64(10), recorder.MustScan().Data)
}

// assertFeedBack 不比较 id
func (s *HandlerTestSuite) assertFeedBack(t *testing.T, expect dao.Feedback, feedBack dao.Feedback) {
	assert.True(t, feedBack.ID > 0)
	assert.True(t, feedBack.Ctime > 0)
	assert.True(t, feedBack.Utime > 0)
	feedBack.ID = 0
	feedBack.Ctime = 0
	feedBack.Utime = 0
	assert.Equal(t, expect, feedBack)
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
