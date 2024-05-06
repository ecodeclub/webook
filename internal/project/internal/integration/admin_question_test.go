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

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/webook/internal/project/internal/domain"
	"github.com/ecodeclub/webook/internal/project/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/project/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *AdminProjectTestSuite) TestQuestionSave() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		req      web.QuestionSaveReq
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "保存成功-新建",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				rsm, err := s.adminPrjDAO.QuestionById(ctx, 1)
				require.NoError(t, err)
				assert.True(t, rsm.Ctime > 0)
				rsm.Ctime = 0
				assert.True(t, rsm.Utime > 0)
				rsm.Utime = 0
				assert.Equal(t, dao.ProjectQuestion{
					Id:       1,
					Pid:      1,
					Title:    "标题1",
					Answer:   "回答1",
					Analysis: "分析1",
					Status:   domain.QuestionStatusUnpublished.ToUint8(),
				}, rsm)
			},
			req: web.QuestionSaveReq{
				Pid: 1,
				Question: web.Question{
					Title:    "标题1",
					Answer:   "回答1",
					Analysis: "分析1",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{Data: 1},
		},
		{
			name: "保存成功-更新",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.ProjectQuestion{
					Id:       2,
					Pid:      1,
					Title:    "老的标题2",
					Answer:   "老的回答2",
					Analysis: "老的分析2",
					Status:   domain.QuestionStatusPublished.ToUint8(),
					Ctime:    123,
					Utime:    123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				rsm, err := s.adminPrjDAO.QuestionById(ctx, 2)
				require.NoError(t, err)
				assert.True(t, rsm.Utime > 123)
				rsm.Utime = 0
				assert.Equal(t, dao.ProjectQuestion{
					Id:       2,
					Pid:      1,
					Title:    "标题2",
					Answer:   "回答2",
					Analysis: "分析2",
					Status:   domain.QuestionStatusUnpublished.ToUint8(),
					Ctime:    123,
				}, rsm)
			},
			req: web.QuestionSaveReq{
				Pid: 1,
				Question: web.Question{
					Id:       2,
					Title:    "标题2",
					Answer:   "回答2",
					Analysis: "分析2",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{Data: 2},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/project/question/save", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
		})
	}
}

func (s *AdminProjectTestSuite) TestQuestionPublish() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		req      web.QuestionSaveReq
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "发表成功-新建",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				rsm, err := s.adminPrjDAO.QuestionById(ctx, 1)
				require.NoError(t, err)
				assert.True(t, rsm.Ctime > 0)
				rsm.Ctime = 0
				assert.True(t, rsm.Utime > 0)
				rsm.Utime = 0
				assert.Equal(t, dao.ProjectQuestion{
					Id:       1,
					Pid:      1,
					Title:    "标题1",
					Answer:   "回答1",
					Analysis: "分析1",
					Status:   domain.QuestionStatusPublished.ToUint8(),
				}, rsm)

				var pubRsm dao.PubProjectQuestion
				err = s.db.WithContext(ctx).Where("id = ?", 1).
					First(&pubRsm).Error
				require.NoError(t, err)
				assert.True(t, pubRsm.Ctime > 0)
				pubRsm.Ctime = 0
				assert.True(t, pubRsm.Utime > 0)
				pubRsm.Utime = 0
				assert.Equal(t, dao.PubProjectQuestion{
					Id:       1,
					Pid:      1,
					Title:    "标题1",
					Answer:   "回答1",
					Analysis: "分析1",
					Status:   domain.QuestionStatusPublished.ToUint8(),
				}, pubRsm)
			},
			req: web.QuestionSaveReq{
				Pid: 1,
				Question: web.Question{
					Title:    "标题1",
					Answer:   "回答1",
					Analysis: "分析1",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{Data: 1},
		},
		{
			name: "发表成功-更新",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.ProjectQuestion{
					Id:       2,
					Pid:      1,
					Title:    "老的标题2",
					Answer:   "老的回答2",
					Analysis: "老的分析2",
					Status:   domain.QuestionStatusUnpublished.ToUint8(),
					Ctime:    123,
					Utime:    123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				rsm, err := s.adminPrjDAO.QuestionById(ctx, 2)
				require.NoError(t, err)
				assert.True(t, rsm.Utime > 0)
				rsm.Utime = 0
				assert.Equal(t, dao.ProjectQuestion{
					Id:       2,
					Pid:      1,
					Title:    "标题2",
					Answer:   "回答2",
					Analysis: "分析2",
					Status:   domain.QuestionStatusPublished.ToUint8(),
					Ctime:    123,
				}, rsm)

				var pubRsm dao.PubProjectQuestion
				err = s.db.WithContext(ctx).Where("id = ?", 2).
					First(&pubRsm).Error
				require.NoError(t, err)
				assert.True(t, pubRsm.Ctime > 0)
				pubRsm.Ctime = 0
				assert.True(t, pubRsm.Utime > 0)
				pubRsm.Utime = 0
				assert.Equal(t, dao.PubProjectQuestion{
					Id:       2,
					Pid:      1,
					Title:    "标题2",
					Answer:   "回答2",
					Analysis: "分析2",
					Status:   domain.QuestionStatusPublished.ToUint8(),
				}, pubRsm)
			},
			req: web.QuestionSaveReq{
				Pid: 1,
				Question: web.Question{
					Id:       2,
					Title:    "标题2",
					Answer:   "回答2",
					Analysis: "分析2",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{Data: 2},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/project/question/publish", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
		})
	}
}

func (s *AdminProjectTestSuite) TestQuestionDetail() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		req    web.IdReq

		wantCode int
		wantResp test.Result[web.Question]
	}{
		{
			name: "获取成功",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.ProjectQuestion{
					Id:       1,
					Pid:      2,
					Title:    "标题",
					Answer:   "回答",
					Analysis: "分析",
					Status:   domain.ProjectStatusUnpublished.ToUint8(),
					Utime:    123,
					Ctime:    123,
				}).Error
				require.NoError(t, err)
			},
			req:      web.IdReq{Id: 1},
			wantCode: 200,
			wantResp: test.Result[web.Question]{
				Data: web.Question{
					Id:       1,
					Title:    "标题",
					Answer:   "回答",
					Analysis: "分析",
					Status:   domain.ProjectStatusUnpublished.ToUint8(),
					Utime:    123,
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/project/question/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Question]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}

}
