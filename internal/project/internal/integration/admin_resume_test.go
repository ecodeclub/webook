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

func (s *AdminProjectTestSuite) TestResumeSave() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		req      web.ResumeSaveReq
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
				rsm, err := s.adminPrjDAO.ResumeById(ctx, 1)
				require.NoError(t, err)
				assert.True(t, rsm.Ctime > 0)
				rsm.Ctime = 0
				assert.True(t, rsm.Utime > 0)
				rsm.Utime = 0
				assert.Equal(t, dao.ProjectResume{
					Id:       1,
					Pid:      1,
					Role:     domain.RoleManager.ToUint8(),
					Content:  "内容1",
					Analysis: "分析1",
					Status:   domain.ResumeStatusUnpublished.ToUint8(),
				}, rsm)
			},
			req: web.ResumeSaveReq{
				Pid: 1,
				Resume: web.Resume{
					Role:     domain.RoleManager.ToUint8(),
					Content:  "内容1",
					Analysis: "分析1",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{Data: 1},
		},
		{
			name: "保存成功-更新",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.ProjectResume{
					Id:       2,
					Pid:      1,
					Role:     domain.RoleIntern.ToUint8(),
					Content:  "老的内容2",
					Analysis: "老的分析2",
					Status:   domain.ResumeStatusPublished.ToUint8(),
					Ctime:    123,
					Utime:    123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				rsm, err := s.adminPrjDAO.ResumeById(ctx, 2)
				require.NoError(t, err)
				assert.True(t, rsm.Ctime > 0)
				rsm.Ctime = 0
				assert.True(t, rsm.Utime > 0)
				rsm.Utime = 0
				assert.Equal(t, dao.ProjectResume{
					Id:       2,
					Pid:      1,
					Role:     domain.RoleManager.ToUint8(),
					Content:  "内容2",
					Analysis: "分析2",
					Status:   domain.ResumeStatusUnpublished.ToUint8(),
				}, rsm)
			},
			req: web.ResumeSaveReq{
				Pid: 1,
				Resume: web.Resume{
					Id:       2,
					Role:     domain.RoleManager.ToUint8(),
					Content:  "内容2",
					Analysis: "分析2",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{Data: 2},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/project/resume/save", iox.NewJSONReader(tc.req))
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

func (s *AdminProjectTestSuite) TestResumePublish() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		req      web.ResumeSaveReq
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
				rsm, err := s.adminPrjDAO.ResumeById(ctx, 1)
				require.NoError(t, err)
				assert.True(t, rsm.Ctime > 0)
				rsm.Ctime = 0
				assert.True(t, rsm.Utime > 0)
				rsm.Utime = 0
				assert.Equal(t, dao.ProjectResume{
					Id:       1,
					Pid:      1,
					Role:     domain.RoleManager.ToUint8(),
					Content:  "内容1",
					Analysis: "分析1",
					Status:   domain.ResumeStatusPublished.ToUint8(),
				}, rsm)

				var pubRsm dao.PubProjectResume
				err = s.db.WithContext(ctx).Where("id = ?", 1).
					First(&pubRsm).Error
				require.NoError(t, err)
				assert.True(t, pubRsm.Ctime > 0)
				pubRsm.Ctime = 0
				assert.True(t, pubRsm.Utime > 0)
				pubRsm.Utime = 0
				assert.Equal(t, dao.PubProjectResume{
					Id:       1,
					Pid:      1,
					Role:     domain.RoleManager.ToUint8(),
					Content:  "内容1",
					Analysis: "分析1",
					Status:   domain.ResumeStatusPublished.ToUint8(),
				}, pubRsm)
			},
			req: web.ResumeSaveReq{
				Pid: 1,
				Resume: web.Resume{
					Role:     domain.RoleManager.ToUint8(),
					Content:  "内容1",
					Analysis: "分析1",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{Data: 1},
		},
		{
			name: "发表成功-更新",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.ProjectResume{
					Id:       2,
					Pid:      1,
					Role:     domain.RoleIntern.ToUint8(),
					Content:  "老的内容2",
					Analysis: "老的分析2",
					Status:   domain.ResumeStatusPublished.ToUint8(),
					Ctime:    123,
					Utime:    123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				rsm, err := s.adminPrjDAO.ResumeById(ctx, 2)
				require.NoError(t, err)
				assert.True(t, rsm.Ctime > 0)
				rsm.Ctime = 0
				assert.True(t, rsm.Utime > 0)
				rsm.Utime = 0
				assert.Equal(t, dao.ProjectResume{
					Id:       2,
					Pid:      1,
					Role:     domain.RoleManager.ToUint8(),
					Content:  "内容2",
					Analysis: "分析2",
					Status:   domain.ResumeStatusPublished.ToUint8(),
				}, rsm)

				var pubRsm dao.PubProjectResume
				err = s.db.WithContext(ctx).Where("id = ?", 2).
					First(&pubRsm).Error
				require.NoError(t, err)
				assert.True(t, pubRsm.Ctime > 0)
				pubRsm.Ctime = 0
				assert.True(t, pubRsm.Utime > 0)
				pubRsm.Utime = 0
				assert.Equal(t, dao.PubProjectResume{
					Id:       2,
					Pid:      1,
					Role:     domain.RoleManager.ToUint8(),
					Content:  "内容2",
					Analysis: "分析2",
					Status:   domain.ResumeStatusPublished.ToUint8(),
				}, pubRsm)
			},
			req: web.ResumeSaveReq{
				Pid: 1,
				Resume: web.Resume{
					Id:       2,
					Role:     domain.RoleManager.ToUint8(),
					Content:  "内容2",
					Analysis: "分析2",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{Data: 2},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/project/resume/publish", iox.NewJSONReader(tc.req))
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

func (s *AdminProjectTestSuite) TestResumeDetail() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		req    web.IdReq

		wantCode int
		wantResp test.Result[web.Resume]
	}{
		{
			name: "获取成功",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.ProjectResume{
					Id:       1,
					Pid:      2,
					Role:     domain.RoleManager.ToUint8(),
					Content:  "内容",
					Analysis: "分析",
					Status:   domain.ProjectStatusUnpublished.ToUint8(),
					Utime:    123,
					Ctime:    123,
				}).Error
				require.NoError(t, err)
			},
			req:      web.IdReq{Id: 1},
			wantCode: 200,
			wantResp: test.Result[web.Resume]{
				Data: web.Resume{
					Id:       1,
					Role:     domain.RoleManager.ToUint8(),
					Content:  "内容",
					Analysis: "分析",
					Status:   domain.ProjectStatusUnpublished.ToUint8(),
					Utime:    123,
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/project/resume/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Resume]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}

}
