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
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/ecodeclub/webook/internal/interactive/internal/domain"

	"github.com/ecodeclub/webook/internal/interactive"

	"github.com/ecodeclub/webook/internal/interactive/internal/event"

	"gorm.io/gorm"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/mq-api"
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

type InteractiveTestSuite struct {
	suite.Suite
	server   *egin.Component
	producer mq.Producer
	db       *egorm.Component
	intrDAO  dao.InteractiveDAO
	svc      interactive.Service
}

func (i *InteractiveTestSuite) TearDownSuite() {
	err := i.db.Exec("DROP TABLE `interactives`").Error
	require.NoError(i.T(), err)
	err = i.db.Exec("DROP TABLE `user_like_bizs`").Error
	require.NoError(i.T(), err)
	err = i.db.Exec("DROP TABLE `user_collection_bizs`").Error
	require.NoError(i.T(), err)
	err = i.db.Exec("DROP TABLE `collections`").Error
	require.NoError(i.T(), err)
}

func (i *InteractiveTestSuite) TearDownTest() {
	err := i.db.Exec("TRUNCATE TABLE `interactives`").Error
	require.NoError(i.T(), err)
	err = i.db.Exec("TRUNCATE TABLE `user_like_bizs`").Error
	require.NoError(i.T(), err)
	err = i.db.Exec("TRUNCATE TABLE `user_collection_bizs`").Error
	require.NoError(i.T(), err)
	err = i.db.Exec("TRUNCATE TABLE `collections`").Error
	require.NoError(i.T(), err)
}

func (i *InteractiveTestSuite) SetupSuite() {
	module, err := startup.InitModule()
	require.NoError(i.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	handler := module.Hdl
	i.svc = module.Svc
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

func (i *InteractiveTestSuite) Test_LikeToggle() {
	testcases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.LikeReq
		wantCode int
	}{
		{
			name: "用户未点赞过_点赞后_点赞计数+1",
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
			name: "用户点赞过_点赞后（相当于取消点赞）_点赞计数-1",
			before: func(t *testing.T) {
				// 直接使用intrDAO下的LikeToggle方法，表示调用一次like/toggle接口
				err := i.intrDAO.LikeToggle(context.Background(), "case", 3, uid)
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				_, err := i.intrDAO.GetLikeInfo(context.Background(), "case", 3, uid)
				assert.Equal(t, gorm.ErrRecordNotFound, err)
				intr, err := i.intrDAO.Get(context.Background(), "case", 3)
				require.NoError(t, err)
				i.assertInteractive(dao.Interactive{
					Biz:     "case",
					BizId:   3,
					LikeCnt: 0,
				}, intr)
			},
			req: web.LikeReq{
				BizId: 3,
				Biz:   "case",
			},
			wantCode: 200,
		},
		{
			name: "用户点赞过_再点赞后(相当于取消点赞)_又点赞_点赞计数+1",
			before: func(t *testing.T) {
				err := i.intrDAO.LikeToggle(context.Background(), "case", 4, uid)
				require.NoError(t, err)
				err = i.intrDAO.LikeToggle(context.Background(), "case", 4, uid)
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
					LikeCnt: 1,
				}, intr)
			},
			req: web.LikeReq{
				BizId: 4,
				Biz:   "case",
			},
			wantCode: 200,
		},
		{
			name: "从未点赞过的两个用户点赞_点赞计数+2",
			before: func(t *testing.T) {
				err := i.intrDAO.LikeToggle(context.Background(), "case", 5, 77)
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				likeInfo, err := i.intrDAO.GetLikeInfo(context.Background(), "case", 5, uid)
				require.NoError(t, err)
				i.assertLikeBiz(dao.UserLikeBiz{
					Uid:   uid,
					Biz:   "case",
					BizId: 5,
				}, likeInfo)
				intr, err := i.intrDAO.Get(context.Background(), "case", 5)
				require.NoError(t, err)
				i.assertInteractive(dao.Interactive{
					Biz:     "case",
					BizId:   5,
					LikeCnt: 2,
				}, intr)
			},
			req: web.LikeReq{
				BizId: 5,
				Biz:   "case",
			},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		tc := tc
		i.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/interactive/like/toggle", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			i.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t)
		})
	}
}

func (i *InteractiveTestSuite) Test_CollectToggle() {
	testcases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.CollectReq
		wantCode int
	}{
		{
			name: "用户未收藏过_收藏后_收藏计数+1",
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
			name: "用户收藏过_收藏后(相当于取消收藏)_收藏计数-1",
			before: func(t *testing.T) {
				err := i.intrDAO.CollectToggle(context.Background(), dao.UserCollectionBiz{
					Uid:   uid,
					Biz:   "question",
					BizId: 3,
				})
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				_, err := i.intrDAO.GetCollectInfo(context.Background(), "question", 3, uid)
				assert.Equal(t, gorm.ErrRecordNotFound, err)
				intr, err := i.intrDAO.Get(context.Background(), "question", 3)
				require.NoError(t, err)
				i.assertInteractive(dao.Interactive{
					Biz:        "question",
					BizId:      3,
					CollectCnt: 0,
				}, intr)
			},
			req: web.CollectReq{
				BizId: 3,
				Biz:   "question",
			},
			wantCode: 200,
		},
		{
			name: "用户收藏过_收藏后(相当于取消收藏)_再点击收藏_收藏计数+1",
			before: func(t *testing.T) {
				err := i.intrDAO.CollectToggle(context.Background(), dao.UserCollectionBiz{
					Biz:   "question",
					BizId: 4,
					Uid:   uid,
				})
				require.NoError(t, err)
				err = i.intrDAO.CollectToggle(context.Background(), dao.UserCollectionBiz{
					Biz:   "question",
					BizId: 4,
					Uid:   uid,
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
					CollectCnt: 1,
				}, intr)
			},
			req: web.CollectReq{
				BizId: 4,
				Biz:   "question",
			},
			wantCode: 200,
		},
		{
			name: "从未收藏过的两个用户收藏_收藏计数+2",
			before: func(t *testing.T) {
				err := i.intrDAO.CollectToggle(context.Background(), dao.UserCollectionBiz{
					Biz:   "question",
					BizId: 5,
					Uid:   34,
				})
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				collectInfo, err := i.intrDAO.GetCollectInfo(context.Background(), "question", 5, uid)
				require.NoError(t, err)
				i.assertCollectBiz(dao.UserCollectionBiz{
					Uid:   uid,
					Biz:   "question",
					BizId: 5,
				}, collectInfo)
				intr, err := i.intrDAO.Get(context.Background(), "question", 5)
				require.NoError(t, err)
				i.assertInteractive(dao.Interactive{
					Biz:        "question",
					BizId:      5,
					CollectCnt: 2,
				}, intr)
			},
			req: web.CollectReq{
				BizId: 5,
				Biz:   "question",
			},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		i.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/interactive/collect/toggle", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			i.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t)
		})
	}
}

func (i *InteractiveTestSuite) Test_Event() {
	testcases := []struct {
		name  string
		msg   event.Event
		after func(t *testing.T)
	}{
		{
			name: "同步点赞事件",
			msg: event.Event{
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
			name: "同步收藏事件",
			msg: event.Event{
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
			name: "同步浏览事件",
			msg: event.Event{
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

func (i *InteractiveTestSuite) TestCollection_Save() {
	testcases := []struct {
		name     string
		req      web.Collection
		before   func(t *testing.T)
		after    func(t *testing.T, id int64)
		wantCode int
	}{
		{
			name: "新建",
			req: web.Collection{
				Name: "收藏夹",
			},
			before: func(t *testing.T) {
			},
			after: func(t *testing.T, id int64) {
				var collection dao.Collection
				err := i.db.WithContext(context.Background()).
					Where("id = ?", id).First(&collection).Error
				require.NoError(t, err)
				require.True(t, collection.Utime > 0)
				require.True(t, collection.Ctime > 0)
				collection.Utime = 0
				collection.Ctime = 0
				assert.Equal(t, dao.Collection{
					Id:   id,
					Uid:  uid,
					Name: "收藏夹",
				}, collection)
			},
			wantCode: 200,
		},
		{
			name: "编辑",
			req: web.Collection{
				Id:   2,
				Name: "旧收藏夹",
			},
			before: func(t *testing.T) {
				err := i.db.WithContext(context.Background()).Create(&dao.Collection{
					Id:    2,
					Uid:   uid,
					Name:  "新收藏夹",
					Ctime: 123,
					Utime: 123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T, id int64) {
				var collection dao.Collection
				err := i.db.WithContext(context.Background()).
					Where("id = ?", id).First(&collection).Error
				require.NoError(t, err)
				require.True(t, collection.Utime > 0)
				require.True(t, collection.Ctime > 0)
				collection.Utime = 0
				collection.Ctime = 0
				assert.Equal(t, dao.Collection{
					Id:   id,
					Uid:  uid,
					Name: "旧收藏夹",
				}, collection)
			},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		i.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/interactive/collection/save", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			i.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t, recorder.MustScan().Data)
		})
	}
}

func (i *InteractiveTestSuite) TestCollection_Delete() {
	testcases := []struct {
		name   string
		before func(t *testing.T) int64
		after  func(t *testing.T, id int64)
		code   int
	}{
		{
			name: "删除收藏夹及其收藏的记录",
			before: func(t *testing.T) int64 {
				// 收藏1
				id, err := i.svc.SaveCollection(context.Background(), domain.Collection{
					Uid:  uid,
					Name: "case收藏夹",
				})
				require.NoError(t, err)
				err = i.svc.CollectToggle(context.Background(), "case", 1, uid)
				require.NoError(t, err)
				err = i.svc.CollectToggle(context.Background(), "case", 2, uid)
				require.NoError(t, err)
				err = i.svc.MoveToCollection(context.Background(), "case", 1, uid, id)
				require.NoError(t, err)
				err = i.svc.MoveToCollection(context.Background(), "case", 2, uid, id)
				require.NoError(t, err)
				records, err := i.svc.CollectionInfo(context.Background(), uid, id, 0, 10)
				require.NoError(t, err)
				assert.ElementsMatch(t, []domain.CollectionRecord{
					{
						Id:   1,
						Biz:  "case",
						Case: 1,
					},
					{
						Id:   2,
						Biz:  "case",
						Case: 2,
					},
				}, records)
				case1Interactive, err := i.svc.Get(context.Background(), "case", 1, uid)
				require.NoError(t, err)
				case2Interactive, err := i.svc.Get(context.Background(), "case", 2, uid)
				require.NoError(t, err)
				assert.Equal(t, 1, case1Interactive.CollectCnt)
				assert.Equal(t, 1, case2Interactive.CollectCnt)
				return id
			},
			after: func(t *testing.T, id int64) {
				case1Interactive, err := i.svc.Get(context.Background(), "case", 1, uid)
				require.NoError(t, err)
				case2Interactive, err := i.svc.Get(context.Background(), "case", 2, uid)
				require.NoError(t, err)
				assert.Equal(t, 0, case1Interactive.CollectCnt)
				assert.Equal(t, 0, case2Interactive.CollectCnt)
				var count int64
				err = i.db.WithContext(context.Background()).
					Model(&dao.UserCollectionBiz{}).
					Where("biz_id IN ? AND biz = ?", []int64{1, 2}, "case").Count(&count).Error
				require.NoError(t, err)
				require.Equal(t, int64(0), count)
				var collection dao.Collection
				err = i.db.WithContext(context.Background()).
					Where("id = ?", id).First(&collection).Error
				assert.Equal(t, gorm.ErrRecordNotFound, err)
			},
			code: 200,
		},
		{
			name: "删除别人文件夹",
			before: func(t *testing.T) int64 {
				// 收藏1
				id, err := i.svc.SaveCollection(context.Background(), domain.Collection{
					Uid:  456,
					Name: "case收藏夹",
				})
				require.NoError(t, err)
				return id
			},
			code: 500,
			after: func(t *testing.T, id int64) {
				var collection dao.Collection
				err := i.db.WithContext(context.Background()).
					Where("id = ?", id).First(&collection).Error
				require.NoError(t, err)
				collection.Utime = 0
				collection.Ctime = 0
				assert.Equal(t, dao.Collection{
					Id:   id,
					Name: "case收藏夹",
					Uid:  456,
				}, collection)
			},
		},
	}
	for _, tc := range testcases {
		i.T().Run(tc.name, func(t *testing.T) {
			id := tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/interactive/collection/delete", iox.NewJSONReader(web.IdReq{
					Id: id,
				}))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			i.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.code, recorder.Code)
			time.Sleep(1 * time.Second)
			tc.after(t, id)
		})
	}
}

func (i *InteractiveTestSuite) TestCollection_List() {
	for j := 1; j <= 4; j++ {
		err := i.db.Create(&dao.Collection{
			Id:   int64(j),
			Uid:  uid,
			Name: fmt.Sprintf("%d", j),
		}).Error
		require.NoError(i.T(), err)
	}
	err := i.db.Create(&dao.Collection{
		Id:   int64(33),
		Uid:  222,
		Name: fmt.Sprintf("%d", 33),
	}).Error
	require.NoError(i.T(), err)

	testcases := []struct {
		name     string
		offset   int
		limit    int
		wantVal  []web.Collection
		wantCode int
	}{
		{
			name:     "偏移2",
			offset:   2,
			limit:    2,
			wantCode: 200,
			wantVal: []web.Collection{
				{
					Id:   2,
					Name: "2",
				},
				{
					Id:   1,
					Name: "1",
				},
			},
		},
		{
			name:     "不包含别人的",
			offset:   0,
			limit:    10,
			wantCode: 200,
			wantVal: []web.Collection{
				{
					Id:   4,
					Name: "4",
				},
				{
					Id:   3,
					Name: "3",
				},
				{
					Id:   2,
					Name: "2",
				},
				{
					Id:   1,
					Name: "1",
				},
			},
		},
	}
	for _, tc := range testcases {
		i.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/interactive/collection/list", iox.NewJSONReader(web.Page{
					Offset: tc.offset,
					Limit:  tc.limit,
				}))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[[]web.Collection]()
			i.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantVal, recorder.MustScan().Data)
		})
	}
}

func (i *InteractiveTestSuite) TestCollection_Move() {
	testcases := []struct {
		name     string
		before   func(t *testing.T) int64
		after    func(t *testing.T, id int64)
		req      web.MoveCollectionReq
		wantCode int
	}{
		{
			req: web.MoveCollectionReq{
				Biz:   "case",
				BizId: 1,
			},
			name: "转移收藏夹",
			before: func(t *testing.T) int64 {
				// 收藏1
				id, err := i.svc.SaveCollection(context.Background(), domain.Collection{
					Uid:  uid,
					Name: "case收藏夹",
				})
				require.NoError(t, err)
				err = i.svc.CollectToggle(context.Background(), "case", 1, uid)
				require.NoError(t, err)
				return id
			},
			after: func(t *testing.T, id int64) {
				var collectionRecords []dao.UserCollectionBiz
				err := i.db.WithContext(context.Background()).
					Model(&dao.UserCollectionBiz{}).
					Where("biz_id IN ? AND biz = ? and uid = ?", []int64{1}, "case", uid).Find(&collectionRecords).Error
				require.NoError(t, err)
				for idx, _ := range collectionRecords {
					collectionRecords[idx].Ctime = 0
					collectionRecords[idx].Utime = 0
				}
				require.Equal(t, []dao.UserCollectionBiz{
					{
						Id:    1,
						Uid:   uid,
						Biz:   "case",
						BizId: 1,
						Cid:   id,
					},
				}, collectionRecords)

			},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		i.T().Run(tc.name, func(t *testing.T) {
			id := tc.before(t)
			tc.req.Cid = id
			req, err := http.NewRequest(http.MethodPost,
				"/interactive/collection/move", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			i.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t, id)
		})
	}
}

func (i *InteractiveTestSuite) assertLikeBiz(want dao.UserLikeBiz, actual dao.UserLikeBiz) {
	t := i.T()
	require.True(t, actual.Id != 0)
	require.True(t, actual.Ctime != 0)
	require.True(t, actual.Utime != 0)
	actual.Id = 0
	actual.Ctime = 0
	actual.Utime = 0
	assert.Equal(t, want, actual)
}

func (i *InteractiveTestSuite) assertInteractive(want dao.Interactive, actual dao.Interactive) {
	t := i.T()
	require.True(t, actual.Id != 0)
	require.True(t, actual.Ctime != 0)
	require.True(t, actual.Utime != 0)
	actual.Id = 0
	actual.Ctime = 0
	actual.Utime = 0
	assert.Equal(t, want, actual)

}

func (i *InteractiveTestSuite) assertCollectBiz(want dao.UserCollectionBiz, actual dao.UserCollectionBiz) {
	t := i.T()
	require.True(t, actual.Id != 0)
	require.True(t, actual.Ctime != 0)
	require.True(t, actual.Utime != 0)
	actual.Id = 0
	actual.Ctime = 0
	actual.Utime = 0
	assert.Equal(t, want, actual)
}

func (i *InteractiveTestSuite) initInteractiveData() {
	biz := "skill"
	i.initInteractiveBizData(biz, 1, 1, 1, 3)
	i.initInteractiveBizData(biz, 2, 3, 2, 9)
	i.initInteractiveBizData(biz, 3, 99, 88, 79)
}

func (i *InteractiveTestSuite) initInteractiveBizData(biz string, bizId int64, viewCnt, likeCnt, collectCnt int) {
	for j := 0; j < viewCnt; j++ {
		err := i.intrDAO.IncrViewCnt(context.Background(), biz, bizId)
		require.NoError(i.T(), err)
	}
	for j := 0; j < likeCnt; j++ {
		err := i.intrDAO.LikeToggle(context.Background(), biz, bizId, int64(j+3))
		require.NoError(i.T(), err)
	}
	for j := 0; j < collectCnt; j++ {
		err := i.intrDAO.CollectToggle(context.Background(), dao.UserCollectionBiz{
			Uid:   int64(j + 4),
			Biz:   biz,
			BizId: bizId,
		})
		require.NoError(i.T(), err)
	}
}

func TestInteractive(t *testing.T) {
	suite.Run(t, new(InteractiveTestSuite))
}
