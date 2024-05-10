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
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/interactive/internal/events"
	"github.com/ecodeclub/webook/internal/interactive/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/interactive/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/interactive/internal/web"
	"github.com/ecodeclub/webook/internal/pkg/middleware"
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

const uid = 1234

type InteractiveSuite struct {
	suite.Suite
	server   *egin.Component
	producer mq.Producer
	db       *egorm.Component
	intrDAO  dao.InteractiveDAO
}

func (i *InteractiveSuite) TearDownSuite() {
	err := i.db.Exec("DROP TABLE `interactives`").Error
	require.NoError(i.T(), err)
	err = i.db.Exec("DROP TABLE `user_like_bizs`").Error
	require.NoError(i.T(), err)
	err = i.db.Exec("DROP TABLE `user_collection_bizs`").Error
	require.NoError(i.T(), err)
}

func (i *InteractiveSuite) TearDownTest() {
	err := i.db.Exec("TRUNCATE TABLE `interactives`").Error
	require.NoError(i.T(), err)
	err = i.db.Exec("TRUNCATE TABLE `user_like_bizs`").Error
	require.NoError(i.T(), err)
	err = i.db.Exec("TRUNCATE TABLE `user_collection_bizs`").Error
	require.NoError(i.T(), err)
}

func (i *InteractiveSuite) SetupSuite() {
	handler, err := startup.InitHandler()
	require.NoError(i.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()

	handler.PublicRoutes(server.Engine)
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: uid,
			Data: map[string]string{
				"creator":   "true",
				"memberDDL": strconv.FormatInt(time.Now().Add(time.Hour).UnixMilli(), 10),
			},
		}))
	})
	handler.PrivateRoutes(server.Engine)
	server.Use(middleware.NewCheckMembershipMiddlewareBuilder(nil).Build())
	i.server = server
	i.db = testioc.InitDB()
	testmq := testioc.InitMQ()
	i.producer, err = testmq.Producer("interactive_events")
	require.NoError(i.T(), err)
	i.intrDAO = dao.NewInteractiveDAO(i.db)
}

func (i *InteractiveSuite) Test_Like() {
	testcases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.LikeReq
		wantCode int
	}{
		{
			name: "用户点赞一次，有一条记录，计数加一",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				likeInfo, err := i.intrDAO.GetLikeInfo(context.Background(), "case", 2, uid)
				require.NoError(t, err)
				i.assertLikeBiz(dao.UserLikeBiz{
					Uid:   uid,
					Biz:   "case",
					BizId: 2,
				}, likeInfo)
				intr, err := i.intrDAO.Get(context.Background(), "case", 2)
				require.NoError(t, err)
				i.assertInteractive(dao.Interactive{
					Biz:     "case",
					BizId:   2,
					LikeCnt: 1,
				}, intr)
			},
			req: web.LikeReq{
				BizId: 2,
				Biz:   "case",
			},
			wantCode: 200,
		},
		{
			name: "同一用户多次点赞,只记录一次",
			before: func(t *testing.T) {
				err := i.intrDAO.InsertLikeInfo(context.Background(), "case", 3, uid)
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				likeInfo, err := i.intrDAO.GetLikeInfo(context.Background(), "case", 3, uid)
				require.NoError(t, err)
				i.assertLikeBiz(dao.UserLikeBiz{
					Uid:   uid,
					Biz:   "case",
					BizId: 3,
				}, likeInfo)
			},
			req: web.LikeReq{
				BizId: 3,
				Biz:   "case",
			},
			wantCode: 500,
		},
		{
			name: "不同的人点赞，次数会增加",
			before: func(t *testing.T) {
				err := i.intrDAO.InsertLikeInfo(context.Background(), "case", 4, 77)
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				likeInfo, err := i.intrDAO.GetLikeInfo(context.Background(), "case", 4, uid)
				require.NoError(t, err)
				i.assertLikeBiz(dao.UserLikeBiz{
					Uid:   uid,
					Biz:   "case",
					BizId: 4,
				}, likeInfo)
				intr, err := i.intrDAO.Get(context.Background(), "case", 4)
				require.NoError(t, err)
				i.assertInteractive(dao.Interactive{
					Biz:     "case",
					BizId:   4,
					LikeCnt: 2,
				}, intr)
			},
			req: web.LikeReq{
				BizId: 4,
				Biz:   "case",
			},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		i.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/intr/like", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			i.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)

			tc.after(t)
		})
	}
}

func (i *InteractiveSuite) Test_Collect() {
	testcases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.CollectReq
		wantCode int
	}{
		{
			name: "用户收藏一次，有一条记录，计数加一",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				collectInfo, err := i.intrDAO.GetCollectInfo(context.Background(), "question", 2, uid)
				require.NoError(t, err)
				i.assertCollectBiz(dao.UserCollectionBiz{
					Uid:   uid,
					Biz:   "question",
					BizId: 2,
				}, collectInfo)
				intr, err := i.intrDAO.Get(context.Background(), "question", 2)
				require.NoError(t, err)
				i.assertInteractive(dao.Interactive{
					Biz:        "question",
					BizId:      2,
					CollectCnt: 1,
				}, intr)
			},
			req: web.CollectReq{
				BizId: 2,
				Biz:   "question",
			},
			wantCode: 200,
		},
		{
			name: "同一用户多次收藏,只记录一次",
			before: func(t *testing.T) {
				err := i.intrDAO.InsertCollectionBiz(context.Background(), dao.UserCollectionBiz{
					Uid:   uid,
					Biz:   "question",
					BizId: 3,
				})
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				collectInfo, err := i.intrDAO.GetCollectInfo(context.Background(), "question", 3, uid)
				require.NoError(t, err)
				i.assertCollectBiz(dao.UserCollectionBiz{
					Uid:   uid,
					Biz:   "question",
					BizId: 3,
				}, collectInfo)
			},
			req: web.CollectReq{
				BizId: 3,
				Biz:   "question",
			},
			wantCode: 500,
		},
		{
			name: "不同的人收藏，次数会增加",
			before: func(t *testing.T) {
				err := i.intrDAO.InsertCollectionBiz(context.Background(), dao.UserCollectionBiz{
					Biz:   "question",
					BizId: 4,
					Uid:   34,
				})
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				collectInfo, err := i.intrDAO.GetCollectInfo(context.Background(), "question", 4, uid)
				require.NoError(t, err)
				i.assertCollectBiz(dao.UserCollectionBiz{
					Uid:   uid,
					Biz:   "question",
					BizId: 4,
				}, collectInfo)
				intr, err := i.intrDAO.Get(context.Background(), "question", 4)
				require.NoError(t, err)
				i.assertInteractive(dao.Interactive{
					Biz:        "question",
					BizId:      4,
					CollectCnt: 2,
				}, intr)
			},
			req: web.CollectReq{
				BizId: 4,
				Biz:   "question",
			},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		i.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/intr/collect", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			i.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t)
		})
	}
}

func (i *InteractiveSuite) Test_View() {
	testcases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.CollectReq
		wantCode int
	}{
		{
			name: "用户首次浏览资源 资源浏览计数加1",
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
			wantCode: 200,
		},
		{
			name: "用户重复浏览资源 资源浏览计数加1",
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
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		i.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/intr/view", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			i.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t)
		})
	}
}

func (i *InteractiveSuite) Test_Cnt() {

	testcases := []struct {
		name     string
		before   func(t *testing.T)
		req      web.GetCntReq
		wantResp web.GetCntResp
		wantCode int
	}{
		{
			name: "用户点赞过的详情",
			before: func(t *testing.T) {
				err := i.intrDAO.IncrViewCnt(context.Background(), "product", 1)
				require.NoError(i.T(), err)
				err = i.intrDAO.InsertLikeInfo(context.Background(), "product", 1, uid)
				require.NoError(i.T(), err)
				err = i.intrDAO.InsertLikeInfo(context.Background(), "product", 1, 11)
				require.NoError(i.T(), err)
				err = i.intrDAO.InsertLikeInfo(context.Background(), "product", 1, 22)
				require.NoError(i.T(), err)
				err = i.intrDAO.InsertCollectionBiz(context.Background(), dao.UserCollectionBiz{
					Uid:   33,
					Biz:   "product",
					BizId: 1,
				})
				require.NoError(i.T(), err)
			},
			req: web.GetCntReq{
				Biz:   "product",
				BizId: 1,
			},
			wantCode: 200,
			wantResp: web.GetCntResp{
				CollectCnt: 1,
				Liked:      true,
				ViewCnt:    1,
				LikeCnt:    3,
			},
		},
		{
			name: "用户收藏过的详情",
			before: func(t *testing.T) {
				err := i.intrDAO.IncrViewCnt(context.Background(), "product", 2)
				require.NoError(i.T(), err)
				err = i.intrDAO.InsertLikeInfo(context.Background(), "product", 2, uid)
				require.NoError(i.T(), err)
				err = i.intrDAO.InsertLikeInfo(context.Background(), "product", 2, 11)
				require.NoError(i.T(), err)
				err = i.intrDAO.InsertLikeInfo(context.Background(), "product", 2, 22)
				require.NoError(i.T(), err)
				err = i.intrDAO.InsertCollectionBiz(context.Background(), dao.UserCollectionBiz{
					Uid:   uid,
					Biz:   "product",
					BizId: 2,
				})
				require.NoError(i.T(), err)
			},
			req: web.GetCntReq{
				Biz:   "product",
				BizId: 2,
			},
			wantCode: 200,
			wantResp: web.GetCntResp{
				CollectCnt: 1,
				Collected:  true,
				Liked:      true,
				ViewCnt:    1,
				LikeCnt:    3,
			},
		},
	}
	for _, tc := range testcases {
		i.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/intr/cnt", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.GetCntResp]()
			i.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			require.Equal(t, tc.wantResp, recorder.MustScan().Data)
		})
	}
}

func (i *InteractiveSuite) Test_Detail() {
	// 批量获取skill模块的id为1,2,3的点赞收藏数据
	t := i.T()
	i.initInteractiveData()
	req, err := http.NewRequest(http.MethodPost,
		"/intr/detail", iox.NewJSONReader(web.BatchGetCntReq{
			Biz:    "skill",
			BizIds: []int64{1, 2, 3},
		}))
	req.Header.Set("content-type", "application/json")
	require.NoError(t, err)
	recorder := test.NewJSONResponseRecorder[web.BatatGetCntResp]()
	i.server.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
	require.Equal(t, web.BatatGetCntResp{
		List: []web.Interactive{
			{
				ID:         3,
				ViewCnt:    99,
				LikeCnt:    88,
				CollectCnt: 79,
			},
			{
				ID:         2,
				ViewCnt:    3,
				LikeCnt:    2,
				CollectCnt: 9,
			},
			{
				ID:         1,
				ViewCnt:    1,
				LikeCnt:    1,
				CollectCnt: 3,
			},
		},
	}, recorder.MustScan().Data)
}

func (i *InteractiveSuite) Test_Event() {
	testcases := []struct {
		name  string
		msg   events.Event
		after func(t *testing.T)
	}{
		{
			name: "点赞",
			msg: events.Event{
				Biz:    "label",
				BizId:  1,
				Action: "like",
				Uid:    33,
			},
			after: func(t *testing.T) {
				likeInfo, err := i.intrDAO.GetLikeInfo(context.Background(), "label", 1, 33)
				require.NoError(t, err)
				i.assertLikeBiz(dao.UserLikeBiz{
					Uid:   33,
					Biz:   "label",
					BizId: 1,
				}, likeInfo)
				intr, err := i.intrDAO.Get(context.Background(), "label", 1)
				require.NoError(t, err)
				i.assertInteractive(dao.Interactive{
					Biz:     "label",
					BizId:   1,
					LikeCnt: 1,
				}, intr)
			},
		},
		{
			name: "收藏",
			msg: events.Event{
				Biz:    "label",
				BizId:  2,
				Action: "collect",
				Uid:    33,
			},
			after: func(t *testing.T) {
				collectInfo, err := i.intrDAO.GetCollectInfo(context.Background(), "label", 2, 33)
				require.NoError(t, err)
				i.assertCollectBiz(dao.UserCollectionBiz{
					Uid:   33,
					Biz:   "label",
					BizId: 2,
				}, collectInfo)
				intr, err := i.intrDAO.Get(context.Background(), "label", 2)
				require.NoError(t, err)
				i.assertInteractive(dao.Interactive{
					Biz:        "label",
					BizId:      2,
					CollectCnt: 1,
				}, intr)
			},
		},
		{
			name: "浏览",
			msg: events.Event{
				Biz:    "label",
				BizId:  3,
				Action: "view",
				Uid:    33,
			},
			after: func(t *testing.T) {
				intr, err := i.intrDAO.Get(context.Background(), "label", 3)
				require.NoError(t, err)
				i.assertInteractive(dao.Interactive{
					Biz:     "label",
					BizId:   3,
					ViewCnt: 1,
				}, intr)
			},
		},
	}
	for _, tc := range testcases {
		i.T().Run(tc.name, func(t *testing.T) {
			v, err := json.Marshal(tc.msg)
			require.NoError(t, err)
			_, err = i.producer.Produce(context.Background(), &mq.Message{
				Value: v,
			})
			require.NoError(t, err)
			time.Sleep(10 * time.Second)
			tc.after(t)

		})
	}

}

func (i *InteractiveSuite) assertLikeBiz(want dao.UserLikeBiz, actual dao.UserLikeBiz) {
	t := i.T()
	require.True(t, actual.Id != 0)
	require.True(t, actual.Ctime != 0)
	require.True(t, actual.Utime != 0)
	actual.Id = 0
	actual.Ctime = 0
	actual.Utime = 0
	assert.Equal(t, want, actual)
}

func (i *InteractiveSuite) assertInteractive(want dao.Interactive, actual dao.Interactive) {
	t := i.T()
	require.True(t, actual.Id != 0)
	require.True(t, actual.Ctime != 0)
	require.True(t, actual.Utime != 0)
	actual.Id = 0
	actual.Ctime = 0
	actual.Utime = 0
	assert.Equal(t, want, actual)

}

func (i *InteractiveSuite) assertCollectBiz(want dao.UserCollectionBiz, actual dao.UserCollectionBiz) {
	t := i.T()
	require.True(t, actual.Id != 0)
	require.True(t, actual.Ctime != 0)
	require.True(t, actual.Utime != 0)
	actual.Id = 0
	actual.Ctime = 0
	actual.Utime = 0
	assert.Equal(t, want, actual)
}

func (i *InteractiveSuite) initInteractiveData() {
	biz := "skill"
	i.initInteractiveBizData(biz, 1, 1, 1, 3)
	i.initInteractiveBizData(biz, 2, 3, 2, 9)
	i.initInteractiveBizData(biz, 3, 99, 88, 79)
}

func (i *InteractiveSuite) initInteractiveBizData(biz string, bizId int64, viewCnt, likeCnt, collectCnt int) {
	for j := 0; j < viewCnt; j++ {
		err := i.intrDAO.IncrViewCnt(context.Background(), biz, bizId)
		require.NoError(i.T(), err)
	}
	for j := 0; j < likeCnt; j++ {
		err := i.intrDAO.InsertLikeInfo(context.Background(), biz, bizId, int64(j+3))
		require.NoError(i.T(), err)
	}
	for j := 0; j < collectCnt; j++ {
		err := i.intrDAO.InsertCollectionBiz(context.Background(), dao.UserCollectionBiz{
			Uid:   int64(j + 4),
			Biz:   biz,
			BizId: bizId,
		})
		require.NoError(i.T(), err)
	}
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(InteractiveSuite))
}
