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
	"testing"
	"time"

	"github.com/ecodeclub/webook/internal/interactive/internal/domain"
	"github.com/ecodeclub/webook/internal/interactive/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/interactive/internal/web"
	"github.com/stretchr/testify/require"
)

func (i *InteractiveTestSuite) Test_View() {
	testcases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)
		req    web.CollectReq
	}{
		{
			name: "用户首次浏览资源，资源浏览计数加1",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				intr, err := i.intrDAO.Get(context.Background(), "order", 3)
				require.NoError(t, err)
				i.assertInteractive(dao.Interactive{
					Biz:     "order",
					BizId:   3,
					ViewCnt: 1,
				}, intr)
			},
			req: web.CollectReq{
				BizId: 3,
				Biz:   "order",
			},
		},
		{
			name: "用户重复浏览资源，资源浏览计数加1",
			before: func(t *testing.T) {
				err := i.intrDAO.IncrViewCnt(context.Background(), "order", 4)
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				intr, err := i.intrDAO.Get(context.Background(), "order", 4)
				require.NoError(t, err)
				i.assertInteractive(dao.Interactive{
					Biz:     "order",
					BizId:   4,
					ViewCnt: 2,
				}, intr)
			},
			req: web.CollectReq{
				BizId: 4,
				Biz:   "order",
			},
		},
	}
	for _, tc := range testcases {
		i.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			err := i.svc.IncrReadCnt(ctx, tc.req.Biz, tc.req.BizId)
			require.NoError(t, err)
			tc.after(t)
		})
	}
}

func (i *InteractiveTestSuite) Test_Cnt() {
	testcases := []struct {
		name     string
		before   func(t *testing.T)
		biz      string
		bizId    int64
		wantErr  error
		wantResp domain.Interactive
	}{
		{
			name: "获取被点赞过的计数信息",
			before: func(t *testing.T) {
				err := i.intrDAO.IncrViewCnt(context.Background(), "product", 1)
				require.NoError(i.T(), err)
				err = i.intrDAO.LikeToggle(context.Background(), "product", 1, uid)
				require.NoError(i.T(), err)
				err = i.intrDAO.LikeToggle(context.Background(), "product", 1, 11)
				require.NoError(i.T(), err)
				err = i.intrDAO.LikeToggle(context.Background(), "product", 1, 22)
				require.NoError(i.T(), err)
				err = i.intrDAO.CollectToggle(context.Background(), dao.UserCollectionBiz{
					Uid:   33,
					Biz:   "product",
					BizId: 1,
				})
				require.NoError(i.T(), err)
			},
			biz:   "product",
			bizId: 1,
			wantResp: domain.Interactive{
				Biz:        "product",
				BizId:      1,
				CollectCnt: 1,
				Liked:      true,
				ViewCnt:    1,
				LikeCnt:    3,
			},
		},
		{
			name: "获取被收藏过的计数信息",
			before: func(t *testing.T) {
				err := i.intrDAO.IncrViewCnt(context.Background(), "product", 2)
				require.NoError(i.T(), err)
				err = i.intrDAO.LikeToggle(context.Background(), "product", 2, uid)
				require.NoError(i.T(), err)
				err = i.intrDAO.LikeToggle(context.Background(), "product", 2, 11)
				require.NoError(i.T(), err)
				err = i.intrDAO.LikeToggle(context.Background(), "product", 2, 22)
				require.NoError(i.T(), err)
				err = i.intrDAO.CollectToggle(context.Background(), dao.UserCollectionBiz{
					Uid:   uid,
					Biz:   "product",
					BizId: 2,
				})
				require.NoError(i.T(), err)
			},
			biz:   "product",
			bizId: 2,
			wantResp: domain.Interactive{
				Biz:        "product",
				BizId:      2,
				CollectCnt: 1,
				Collected:  true,
				Liked:      true,
				ViewCnt:    1,
				LikeCnt:    3,
			},
		},
		{
			name: "获取没有点赞，收藏，阅读过的计数信息",
			before: func(t *testing.T) {
			},
			biz:   "product",
			bizId: 3,
			wantResp: domain.Interactive{
				CollectCnt: 0,
				Collected:  false,
				Liked:      false,
				ViewCnt:    0,
				LikeCnt:    0,
			},
		},
	}
	for _, tc := range testcases {
		i.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			res, err := i.svc.Get(ctx, tc.biz, tc.bizId, uid)
			require.NoError(t, err)
			require.Equal(t, tc.wantResp, res)
		})
	}
}

func (i *InteractiveTestSuite) TestGetByIds() {
	// 批量获取skill模块的id为1,2,3,4的点赞收藏数据
	t := i.T()
	i.initInteractiveData()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	res, err := i.svc.GetByIds(ctx, "skill", []int64{1, 2, 3, 4})
	require.NoError(i.T(), err)
	require.EqualValues(t, map[int64]domain.Interactive{
		4: {
			Biz:   "skill",
			BizId: 4,
		},
		3: {
			Biz:        "skill",
			BizId:      3,
			ViewCnt:    99,
			LikeCnt:    88,
			CollectCnt: 79,
		},
		2: {
			Biz:        "skill",
			BizId:      2,
			ViewCnt:    3,
			LikeCnt:    2,
			CollectCnt: 9,
		},
		1: {
			Biz:        "skill",
			BizId:      1,
			ViewCnt:    1,
			LikeCnt:    1,
			CollectCnt: 3,
		},
	}, res)
}
