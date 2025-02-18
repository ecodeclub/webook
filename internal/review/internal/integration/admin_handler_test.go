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

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/ecodeclub/ecache"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"
	"github.com/ecodeclub/webook/internal/review/internal/domain"
	"github.com/ecodeclub/webook/internal/review/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/review/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/review/internal/web"
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

type AdminHandlerTestSuite struct {
	suite.Suite
	db        *egorm.Component
	server    *egin.Component
	reviewDao dao.ReviewDAO
	rdb       ecache.Cache
}

func (s *AdminHandlerTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `reviews`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `publish_reviews`").Error
	require.NoError(s.T(), err)
}

func (s *AdminHandlerTestSuite) SetupSuite() {
	db := testioc.InitDB()
	testmq := testioc.InitMQ()
	rdb := testioc.InitCache()
	ctrl := gomock.NewController(s.T())
	svc := intrmocks.NewMockService(ctrl)
	svc.EXPECT().GetByIds(gomock.Any(), "review", gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, biz string, uid int64, ids []int64) (map[int64]interactive.Interactive, error) {
		res := make(map[int64]interactive.Interactive, len(ids))
		for _, id := range ids {
			intr := mockInteractive(biz, id)
			res[id] = intr
		}
		return res, nil
	}).AnyTimes()
	mou := startup.InitModule(db, &interactive.Module{
		Svc: svc,
	}, testmq, rdb, session.DefaultProvider())
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: uid,
			Data: map[string]string{
				"creator":   "true",
				"memberDDL": strconv.FormatInt(time.Now().Add(time.Hour).UnixMilli(), 10),
			},
		}))
	})
	mou.AdminHdl.PrivateRoutes(server.Engine)
	reviewDao := dao.NewReviewDAO(db)
	s.db = db
	s.server = server
	s.reviewDao = reviewDao
	s.rdb = rdb
}

func (s *AdminHandlerTestSuite) TestSave() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.ReviewSaveReq
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "新建面经",
			before: func(t *testing.T) {
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				review, err := s.reviewDao.Get(ctx, 1)
				require.NoError(t, err)
				assertReview(t, dao.Review{
					ID:    1,
					Uid:   uid,
					Title: "标题",
					Desc:  "简介",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
					JD:               "测试JD",
					JDAnalysis:       "JD分析",
					Questions:        "面试问题",
					QuestionAnalysis: "问题分析",
					Resume:           "简历内容",
					Status:           domain.UnPublishedStatus.ToUint8(), // 未发布状态
				}, review)
			},
			req: web.ReviewSaveReq{
				Review: web.Review{
					Title:            "标题",
					Desc:             "简介",
					Labels:           []string{"MySQL"},
					JD:               "测试JD",
					JDAnalysis:       "JD分析",
					Questions:        "面试问题",
					QuestionAnalysis: "问题分析",
					Resume:           "简历内容",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		},
		{
			name: "更新面经",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				_, err := s.reviewDao.Save(ctx, dao.Review{
					ID:    2,
					Uid:   uid,
					Title: "旧的标题",
					Desc:  "旧的简介",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"旧MySQL"},
					},
					JD:               "旧的JD",
					JDAnalysis:       "旧的分析",
					Questions:        "旧的问题",
					QuestionAnalysis: "旧的分析",
					Resume:           "旧的简历",
					Status:           domain.UnPublishedStatus.ToUint8(),
				})
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				review, err := s.reviewDao.Get(ctx, 2)
				require.NoError(t, err)
				assertReview(t, dao.Review{
					ID:    2,
					Uid:   uid,
					Title: "新的标题",
					Desc:  "新的简介",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"新MySQL"},
					},
					JD:               "新的JD",
					JDAnalysis:       "新的分析",
					Questions:        "新的问题",
					QuestionAnalysis: "新的分析",
					Resume:           "新的简历",
					Status:           domain.UnPublishedStatus.ToUint8(),
				}, review)
			},
			req: web.ReviewSaveReq{
				Review: web.Review{
					ID:               2,
					Title:            "新的标题",
					Desc:             "新的简介",
					Labels:           []string{"新MySQL"},
					JD:               "新的JD",
					JDAnalysis:       "新的分析",
					Questions:        "新的问题",
					QuestionAnalysis: "新的分析",
					Resume:           "新的简历",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 2,
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/review/save", iox.NewJSONReader(tc.req))
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

func (s *AdminHandlerTestSuite) TestPublish() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.ReviewSaveReq
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "新建并发布",
			before: func(t *testing.T) {
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				// 检查原始表中的数据
				review, err := s.reviewDao.Get(ctx, 1)
				require.NoError(t, err)
				wantReview := dao.Review{
					ID:    1,
					Title: "标题",
					Desc:  "简介",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
					Uid:              uid,
					JD:               "测试JD",
					JDAnalysis:       "JD分析",
					Questions:        "面试问题",
					QuestionAnalysis: "问题分析",
					Resume:           "简历内容",
					Status:           domain.PublishedStatus.ToUint8(), // 已发布状态
				}
				assertReview(t, wantReview, review)

				// 检查发布表中的数据
				pubReview, err := s.reviewDao.GetPublishReview(ctx, 1)
				require.NoError(t, err)
				assertReview(t, wantReview, dao.Review(pubReview))
				s.assertCachedReview(t, domain.Review{
					ID:               1,
					Title:            "标题",
					Desc:             "简介",
					Labels:           []string{"MySQL"},
					Uid:              uid,
					JD:               "测试JD",
					JDAnalysis:       "JD分析",
					Questions:        "面试问题",
					QuestionAnalysis: "问题分析",
					Resume:           "简历内容",
					Status:           domain.PublishedStatus,
				})
			},
			req: web.ReviewSaveReq{
				Review: web.Review{
					Title:            "标题",
					Desc:             "简介",
					Labels:           []string{"MySQL"},
					JD:               "测试JD",
					JDAnalysis:       "JD分析",
					Questions:        "面试问题",
					QuestionAnalysis: "问题分析",
					Resume:           "简历内容",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		},
		{
			name: "更新并发布",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				// 先创建一条记录
				_, err := s.reviewDao.Save(ctx, dao.Review{
					ID:    2,
					Uid:   uid,
					Title: "旧的标题",
					Desc:  "旧的简介",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"旧MySQL"},
					},
					JD:               "旧的JD",
					JDAnalysis:       "旧的分析",
					Questions:        "旧的问题",
					QuestionAnalysis: "旧的分析",
					Resume:           "旧的简历",
					Status:           1,
					Ctime:            123,
					Utime:            234,
				})
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				// 检查原始表
				review, err := s.reviewDao.Get(ctx, 2)
				require.NoError(t, err)

				wantReview := dao.Review{
					ID:    2,
					Uid:   uid,
					Title: "新的标题",
					Desc:  "新的简介",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"新MySQL"},
					},
					JD:               "新的JD",
					JDAnalysis:       "新的分析",
					Questions:        "新的问题",
					QuestionAnalysis: "新的分析",
					Resume:           "新的简历",
					Status:           2, // 已发布状态
				}
				assertReview(t, wantReview, review)

				// 检查发布表
				pubReview, err := s.reviewDao.GetPublishReview(ctx, 2)
				require.NoError(t, err)
				assertReview(t, dao.Review(wantReview), dao.Review(pubReview))
				s.assertCachedReview(t, domain.Review{
					ID:               2,
					Uid:              uid,
					Title:            "新的标题",
					Desc:             "新的简介",
					Labels:           []string{"新MySQL"},
					JD:               "新的JD",
					JDAnalysis:       "新的分析",
					Questions:        "新的问题",
					QuestionAnalysis: "新的分析",
					Resume:           "新的简历",
					Status:           2, // 已发布状态
				})
			},
			req: web.ReviewSaveReq{
				Review: web.Review{
					ID:               2,
					Title:            "新的标题",
					Desc:             "新的简介",
					Labels:           []string{"新MySQL"},
					JD:               "新的JD",
					JDAnalysis:       "新的分析",
					Questions:        "新的问题",
					QuestionAnalysis: "新的分析",
					Resume:           "新的简历",
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 2,
			},
		},
		{
			name: "发布表已有记录时更新发布",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				// 创建原始记录
				oldReview := dao.Review{
					ID:    3,
					Uid:   uid,
					Title: "旧的标题",
					Desc:  "旧的简介",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"旧MySQL"},
					},
					JD:               "旧的JD",
					JDAnalysis:       "旧的分析",
					Questions:        "旧的问题",
					QuestionAnalysis: "旧的分析",
					Resume:           "旧的简历",
					Status:           domain.UnPublishedStatus.ToUint8(),
				}
				_, err := s.reviewDao.Save(ctx, oldReview)
				require.NoError(t, err)

				// 创建发布表记录
				_, err = s.reviewDao.Sync(ctx, oldReview)
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				wantReview := dao.Review{
					ID:    3,
					Uid:   uid,
					Title: "最新标题",
					Desc:  "最新简介",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"最新MySQL"},
					},
					JD:               "最新JD",
					JDAnalysis:       "最新分析",
					Questions:        "最新问题",
					QuestionAnalysis: "最新分析",
					Resume:           "最新简历",
					Status:           domain.PublishedStatus.ToUint8(),
				}

				// 检查原始表
				review, err := s.reviewDao.Get(ctx, 3)
				require.NoError(t, err)
				assertReview(t, wantReview, review)

				// 检查发布表
				pubReview, err := s.reviewDao.GetPublishReview(ctx, 3)
				require.NoError(t, err)
				assertReview(t, dao.Review(wantReview), dao.Review(pubReview))

				s.assertCachedReview(t, domain.Review{
					ID:               3,
					Uid:              uid,
					Title:            "最新标题",
					Desc:             "最新简介",
					Labels:           []string{"最新MySQL"},
					JD:               "最新JD",
					JDAnalysis:       "最新分析",
					Questions:        "最新问题",
					QuestionAnalysis: "最新分析",
					Resume:           "最新简历",
					Status:           domain.PublishedStatus,
				})
			},
			req: web.ReviewSaveReq{
				Review: web.Review{
					ID:               3,
					Title:            "最新标题",
					Desc:             "最新简介",
					Labels:           []string{"最新MySQL"},
					JD:               "最新JD",
					JDAnalysis:       "最新分析",
					Questions:        "最新问题",
					QuestionAnalysis: "最新分析",
					Resume:           "最新简历",
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
			// 运行前置操作
			tc.before(t)

			// 构造请求
			req, err := http.NewRequest(http.MethodPost,
				"/review/publish", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)

			// 发送请求并记录响应
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)

			// 验证响应
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())

			// 运行后置检查
			tc.after(t)

			// 清理数据
			err = s.db.Exec("TRUNCATE table `reviews`").Error
			require.NoError(t, err)
			err = s.db.Exec("TRUNCATE table `publish_reviews`").Error
			require.NoError(t, err)
		})
	}
}

func (s *AdminHandlerTestSuite) TestDetail() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.DetailReq
		wantCode int
		wantResp test.Result[web.Review]
	}{
		{
			name: "查询存在的记录",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				_, err := s.reviewDao.Save(ctx, dao.Review{
					ID:    1,
					Uid:   uid,
					Title: "测试标题",
					Desc:  "测试描述",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"测试标签"},
					},
					JD:               "测试JD",
					JDAnalysis:       "JD分析",
					Questions:        "面试问题",
					QuestionAnalysis: "问题分析",
					Resume:           "简历内容",
					Status:           domain.UnPublishedStatus.ToUint8(),
				})
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
			},
			req: web.DetailReq{
				ID: 1,
			},
			wantCode: 200,
			wantResp: test.Result[web.Review]{
				Data: web.Review{
					ID:    1,
					Title: "测试标题",
					Desc:  "测试描述",
					Labels: []string{
						"测试标签",
					},
					JD:               "测试JD",
					JDAnalysis:       "JD分析",
					Questions:        "面试问题",
					QuestionAnalysis: "问题分析",
					Resume:           "简历内容",
					Status:           domain.UnPublishedStatus.ToUint8(),
				},
			},
		},
		{
			name: "查询不存在的记录",
			before: func(t *testing.T) {
			},
			after: func(t *testing.T) {
			},
			req: web.DetailReq{
				ID: 999,
			},
			wantCode: 500,
			wantResp: test.Result[web.Review]{
				Code: 516001,
				Msg:  "系统错误",
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// 运行前置操作
			tc.before(t)

			// 构造请求
			req, err := http.NewRequest(http.MethodPost,
				"/review/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)

			// 发送请求并记录响应
			recorder := test.NewJSONResponseRecorder[web.Review]()
			s.server.ServeHTTP(recorder, req)

			// 验证响应
			require.Equal(t, tc.wantCode, recorder.Code)
			if tc.wantCode != 200 {
				return
			}
			resp := recorder.MustScan()
			require.True(t, resp.Data.Utime != 0)
			resp.Data.Utime = 0
			assert.Equal(t, tc.wantResp, resp)

			// 运行后置检查
			tc.after(t)

			// 清理数据
			err = s.db.Exec("TRUNCATE table `reviews`").Error
			require.NoError(t, err)
			err = s.db.Exec("TRUNCATE table `publish_reviews`").Error
			require.NoError(t, err)
		})
	}
}

func (s *AdminHandlerTestSuite) TestList() {
	data := make([]dao.Review, 0, 100)
	for idx := 0; idx < 100; idx++ {
		data = append(data, dao.Review{
			Uid:   uid,
			Title: fmt.Sprintf("标题 %d", idx),
			Desc:  fmt.Sprintf("描述 %d", idx),
			Labels: sqlx.JsonColumn[[]string]{
				Valid: true,
				Val:   []string{fmt.Sprintf("标签 %d", idx)},
			},
			JD:               fmt.Sprintf("这是JD %d", idx),
			JDAnalysis:       fmt.Sprintf("这是JD分析 %d", idx),
			Questions:        fmt.Sprintf("这是面试问题 %d", idx),
			QuestionAnalysis: fmt.Sprintf("这是问题分析 %d", idx),
			Resume:           fmt.Sprintf("这是简历 %d", idx),
			Status:           domain.UnPublishedStatus.ToUint8(),
			Utime:            123,
		})
	}
	err := s.db.Create(&data).Error
	require.NoError(s.T(), err)
	testCases := []struct {
		name     string
		req      web.Page
		wantCode int
		wantResp test.Result[web.ReviewListResp]
	}{
		{
			name: "获取第一页",
			req: web.Page{
				Limit:  2,
				Offset: 0,
			},
			wantCode: 200,
			wantResp: test.Result[web.ReviewListResp]{
				Data: web.ReviewListResp{
					Total: 100,
					List: []web.Review{
						{
							ID:               100,
							Title:            "标题 99",
							Desc:             "描述 99",
							Labels:           []string{"标签 99"},
							JD:               "这是JD 99",
							JDAnalysis:       "这是JD分析 99",
							Questions:        "这是面试问题 99",
							QuestionAnalysis: "这是问题分析 99",
							Resume:           "这是简历 99",
							Status:           domain.UnPublishedStatus.ToUint8(),
							Utime:            123,
						},
						{
							ID:               99,
							Title:            "标题 98",
							Desc:             "描述 98",
							Labels:           []string{"标签 98"},
							JD:               "这是JD 98",
							JDAnalysis:       "这是JD分析 98",
							Questions:        "这是面试问题 98",
							QuestionAnalysis: "这是问题分析 98",
							Resume:           "这是简历 98",
							Status:           domain.UnPublishedStatus.ToUint8(),
							Utime:            123,
						},
					},
				},
			},
		},
		{
			name: "获取最后一页",
			req: web.Page{
				Limit:  2,
				Offset: 99,
			},
			wantCode: 200,
			wantResp: test.Result[web.ReviewListResp]{
				Data: web.ReviewListResp{
					Total: 100,
					List: []web.Review{
						{
							ID:               1,
							Title:            "标题 0",
							Desc:             "描述 0",
							Labels:           []string{"标签 0"},
							JD:               "这是JD 0",
							JDAnalysis:       "这是JD分析 0",
							Questions:        "这是面试问题 0",
							QuestionAnalysis: "这是问题分析 0",
							Resume:           "这是简历 0",
							Status:           domain.UnPublishedStatus.ToUint8(),
							Utime:            123,
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
				"/review/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.ReviewListResp]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *AdminHandlerTestSuite) assertCachedReview(t *testing.T, want domain.Review) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	key := fmt.Sprintf("review:publish:%d", want.ID)
	// 获取缓存值
	cachedVal := s.rdb.Get(ctx, key)
	require.NoError(t, cachedVal.Err)

	// 反序列化
	var cachedReview domain.Review
	err := json.Unmarshal([]byte(cachedVal.Val.(string)), &cachedReview)
	require.NoError(t, err)
	require.True(t, cachedReview.Utime > 0)
	cachedReview.Utime = 0
	// 断言内容
	assert.Equal(t, want, cachedReview)
	_, err = s.rdb.Delete(context.Background(), key)
	require.NoError(t, err)
}

func TestReviewAdminHandler(t *testing.T) {
	suite.Run(t, new(AdminHandlerTestSuite))
}
