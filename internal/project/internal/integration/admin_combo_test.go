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
	"fmt"
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

func (s *AdminProjectTestSuite) TestComboSave() {
	const pid = 123
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		req      web.ComboSaveReq
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name:   "新建",
			before: func(t *testing.T) {},
			after: func(t *testing.T) {
				// 验证数据库的数据
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				c, err := s.adminPrjDAO.ComboById(ctx, 1)
				require.NoError(t, err)
				assert.True(t, c.Utime > 0)
				c.Utime = 0
				assert.True(t, c.Ctime > 0)
				c.Ctime = 0
				assert.Equal(t, dao.ProjectCombo{
					Id:      1,
					Pid:     pid,
					Title:   "标题1",
					Content: "内容1",
					Status:  domain.ComboStatusUnpublished.ToUint8(),
				}, c)
			},
			req: web.ComboSaveReq{
				Pid: pid,
				Combo: web.Combo{
					Title:   "标题1",
					Content: "内容1",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{Data: 1},
		},
		{
			name: "更新",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.ProjectCombo{
					Id:      2,
					Pid:     pid,
					Title:   "老的标题1",
					Content: "老的内容1",
					Status:  domain.ComboStatusPublished.ToUint8(),
					Utime:   123,
					Ctime:   123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				// 验证数据库的数据
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				c, err := s.adminPrjDAO.ComboById(ctx, 2)
				require.NoError(t, err)
				// 更新时间变了
				assert.True(t, c.Utime > 123)
				c.Utime = 0
				assert.Equal(t, dao.ProjectCombo{
					Id: 2,
					// pid 不会发生变化
					Pid:     123,
					Title:   "标题1",
					Content: "内容1",
					Status:  domain.ComboStatusUnpublished.ToUint8(),
					// Ctime 也不会发生变化
					Ctime: 123,
				}, c)
			},
			req: web.ComboSaveReq{
				Pid: 1234,
				Combo: web.Combo{
					Id:      2,
					Title:   "标题1",
					Content: "内容1",
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
				"/project/combo/save", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			val := recorder.MustScan()
			assert.Equal(t, tc.wantResp, val)
			tc.after(t)
		})
	}
}

func (s *AdminProjectTestSuite) TestComboDetail() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	const pid = 123
	err := s.db.WithContext(ctx).Create(s.mockCombo(pid, 1)).Error
	require.NoError(s.T(), err)
	testCases := []struct {
		name     string
		req      web.IdReq
		wantCode int
		wantResp test.Result[web.Combo]
	}{
		{
			name: "成功",
			req: web.IdReq{
				Id: 1,
			},

			wantCode: 200,
			wantResp: test.Result[web.Combo]{
				Data: web.Combo{
					Id:      1,
					Title:   "标题1",
					Content: "内容1",
					Status:  domain.ComboStatusUnpublished.ToUint8(),
					Utime:   1,
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/project/combo/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Combo]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			val := recorder.MustScan()
			assert.Equal(t, tc.wantResp, val)
		})
	}
}

func (s *AdminProjectTestSuite) TestComboPublish() {
	const pid = 123
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		req      web.ComboSaveReq
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "全新建",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				// 验证两个库都有数据
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				c, err := s.adminPrjDAO.ComboById(ctx, 1)
				require.NoError(t, err)
				assert.True(t, c.Utime > 0)
				c.Utime = 0
				assert.True(t, c.Ctime > 0)
				c.Ctime = 0
				assert.Equal(t, dao.ProjectCombo{
					Id:      1,
					Pid:     pid,
					Title:   "标题1",
					Content: "内容1",
					Status:  domain.ComboStatusPublished.ToUint8(),
				}, c)

				var pub dao.PubProjectCombo
				err = s.db.WithContext(ctx).Where("id = ?", 1).First(&pub).Error
				require.NoError(t, err)
				assert.True(t, pub.Utime > 0)
				pub.Utime = 0
				assert.True(t, pub.Ctime > 0)
				pub.Ctime = 0
				assert.Equal(t, dao.PubProjectCombo{
					Id:      1,
					Pid:     pid,
					Title:   "标题1",
					Content: "内容1",
					Status:  domain.ComboStatusPublished.ToUint8(),
				}, pub)
			},
			req: web.ComboSaveReq{
				Pid: pid,
				Combo: web.Combo{
					Title:   "标题1",
					Content: "内容1",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		},

		{
			name: "制作库存在，线上库新建",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				err := s.db.WithContext(ctx).Create(dao.ProjectCombo{
					Id:      2,
					Pid:     pid,
					Title:   "老的标题1",
					Content: "老的内容1",
					Status:  domain.ComboStatusPublished.ToUint8(),
					Utime:   123,
					Ctime:   123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				// 验证两个库都有数据
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				c, err := s.adminPrjDAO.ComboById(ctx, 2)
				require.NoError(t, err)
				assert.True(t, c.Utime > 123)
				c.Utime = 0
				assert.Equal(t, dao.ProjectCombo{
					Id:      2,
					Pid:     pid,
					Title:   "标题1",
					Content: "内容1",
					Status:  domain.ComboStatusPublished.ToUint8(),
					Ctime:   123,
				}, c)

				var pub dao.PubProjectCombo
				err = s.db.WithContext(ctx).Where("id = ?", 2).First(&pub).Error
				require.NoError(t, err)
				assert.True(t, pub.Utime > 0)
				pub.Utime = 0
				assert.True(t, pub.Ctime > 0)
				pub.Ctime = 0
				assert.Equal(t, dao.PubProjectCombo{
					Id:      2,
					Pid:     pid,
					Title:   "标题1",
					Content: "内容1",
					Status:  domain.ComboStatusPublished.ToUint8(),
				}, pub)
			},
			req: web.ComboSaveReq{
				Pid: pid,
				Combo: web.Combo{
					Id:      2,
					Title:   "标题1",
					Content: "内容1",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 2,
			},
		},

		{
			name: "制作库线上库更新",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				err := s.db.WithContext(ctx).Create(dao.ProjectCombo{
					Id:      3,
					Pid:     pid,
					Title:   "老的标题1",
					Content: "老的内容1",
					Status:  domain.ComboStatusPublished.ToUint8(),
					Utime:   123,
					Ctime:   123,
				}).Error
				require.NoError(t, err)
				err = s.db.WithContext(ctx).Create(dao.PubProjectCombo{
					Id:      3,
					Pid:     pid,
					Title:   "老的标题1",
					Content: "老的内容1",
					Status:  domain.ComboStatusPublished.ToUint8(),
					Utime:   123,
					Ctime:   123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				// 验证两个库都有数据
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				c, err := s.adminPrjDAO.ComboById(ctx, 3)
				require.NoError(t, err)
				assert.True(t, c.Utime > 123)
				c.Utime = 0
				assert.Equal(t, dao.ProjectCombo{
					Id:      3,
					Pid:     pid,
					Title:   "标题1",
					Content: "内容1",
					Status:  domain.ComboStatusPublished.ToUint8(),
					Ctime:   123,
				}, c)

				var pub dao.PubProjectCombo
				err = s.db.WithContext(ctx).Where("id = ?", 3).First(&pub).Error
				require.NoError(t, err)
				assert.True(t, pub.Utime > 123)
				pub.Utime = 0
				assert.Equal(t, dao.PubProjectCombo{
					Id:      3,
					Pid:     pid,
					Title:   "标题1",
					Content: "内容1",
					Ctime:   123,
					Status:  domain.ComboStatusPublished.ToUint8(),
				}, pub)
			},
			req: web.ComboSaveReq{
				Pid: pid,
				Combo: web.Combo{
					Id:      3,
					Title:   "标题1",
					Content: "内容1",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 3,
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/project/combo/publish", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			val := recorder.MustScan()
			assert.Equal(t, tc.wantResp, val)
			tc.after(t)
		})
	}
}

func (s *AdminProjectTestSuite) mockCombo(pid, id int64) dao.ProjectCombo {
	return dao.ProjectCombo{
		Id:      id,
		Pid:     pid,
		Title:   fmt.Sprintf("标题%d", id),
		Content: fmt.Sprintf("内容%d", id),
		Status:  domain.ComboStatusUnpublished.ToUint8(),
		Ctime:   id,
		Utime:   id,
	}
}
