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

	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
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
)

const uid = 123

type TestSuite struct {
	suite.Suite
	db        *egorm.Component
	server    *egin.Component
	reviewDao dao.ReviewDAO
	rdb       ecache.Cache
}

func mockInteractive(biz string, id int64) interactive.Interactive {
	liked := id%2 == 1
	collected := id%2 == 0
	return interactive.Interactive{
		Biz:        biz,
		BizId:      id,
		ViewCnt:    int(id + 1),
		LikeCnt:    int(id + 2),
		CollectCnt: int(id + 3),
		Liked:      liked,
		Collected:  collected,
	}
}

func (s *TestSuite) SetupSuite() {
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

	svc.EXPECT().Get(gomock.Any(), "review", gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, biz string, id, uid int64) (interactive.Interactive, error) {
		return mockInteractive(biz, id), nil
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
	mou.Hdl.PublicRoutes(server.Engine)
	reviewDao := dao.NewReviewDAO(db)
	s.db = db
	s.server = server
	s.reviewDao = reviewDao
	s.rdb = rdb
}

func (s *TestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `reviews`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `publish_reviews`").Error
	require.NoError(s.T(), err)
}

func (s *TestSuite) TestPubList() {
	// 插入一百条测试数据到原始表和发布表
	data := make([]dao.Review, 0, 100)
	pubData := make([]dao.PublishReview, 0, 50) // 只发布一半的数据

	for idx := 1; idx <= 100; idx++ {
		review := dao.Review{
			ID:    int64(idx),
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
		}
		data = append(data, review)

		// 偶数ID的记录设为已发布
		if idx%2 == 0 {
			review.Status = domain.PublishedStatus.ToUint8()
			pubData = append(pubData, dao.PublishReview(review))
		}
	}

	err := s.db.Create(&data).Error
	require.NoError(s.T(), err)
	err = s.db.Create(&pubData).Error
	require.NoError(s.T(), err)

	testCases := []struct {
		name     string
		req      web.Page
		wantCode int
		wantResp test.Result[web.ReviewListResp]
	}{
		{
			name: "获取第一页已发布记录",
			req: web.Page{
				Limit:  2,
				Offset: 0,
			},
			wantCode: 200,
			wantResp: test.Result[web.ReviewListResp]{
				Data: web.ReviewListResp{
					Total: 0, // 只有50条已发布的记录
					List: []web.Review{
						{
							ID:               100,
							Title:            "标题 100",
							Desc:             "描述 100",
							Labels:           []string{"标签 100"},
							JD:               "这是JD 100",
							JDAnalysis:       "这是JD分析 100",
							Questions:        "这是面试问题 100",
							QuestionAnalysis: "这是问题分析 100",
							Resume:           "这是简历 100",
							Status:           domain.PublishedStatus.ToUint8(),
							Utime:            123,
							Interactive: web.Interactive{
								CollectCnt: 103,   // id + 3
								LikeCnt:    102,   // id + 2
								ViewCnt:    101,   // id + 1
								Liked:      false, // id 为偶数时为 false
								Collected:  true,  // id 为偶数时为 true
							},
						},
						{
							ID:               98,
							Title:            "标题 98",
							Desc:             "描述 98",
							Labels:           []string{"标签 98"},
							JD:               "这是JD 98",
							JDAnalysis:       "这是JD分析 98",
							Questions:        "这是面试问题 98",
							QuestionAnalysis: "这是问题分析 98",
							Resume:           "这是简历 98",
							Status:           domain.PublishedStatus.ToUint8(),
							Utime:            123,
							Interactive: web.Interactive{
								CollectCnt: 101,   // id + 3
								LikeCnt:    100,   // id + 2
								ViewCnt:    99,    // id + 1
								Liked:      false, // id 为偶数时为 false
								Collected:  true,  // id 为偶数时为 true
							},
						},
					},
				},
			},
		},
		{
			name: "获取最后一页已发布记录",
			req: web.Page{
				Limit:  2,
				Offset: 48,
			},
			wantCode: 200,
			wantResp: test.Result[web.ReviewListResp]{
				Data: web.ReviewListResp{
					Total: 0,
					List: []web.Review{
						{
							ID:               4,
							Title:            "标题 4",
							Desc:             "描述 4",
							Labels:           []string{"标签 4"},
							JD:               "这是JD 4",
							JDAnalysis:       "这是JD分析 4",
							Questions:        "这是面试问题 4",
							QuestionAnalysis: "这是问题分析 4",
							Resume:           "这是简历 4",
							Status:           domain.PublishedStatus.ToUint8(),
							Utime:            123,
							Interactive: web.Interactive{
								CollectCnt: 7,     // id + 3
								LikeCnt:    6,     // id + 2
								ViewCnt:    5,     // id + 1
								Liked:      false, // id 为偶数时为 false
								Collected:  true,  // id 为偶数时为 true
							},
						},
						{
							ID:               2,
							Title:            "标题 2",
							Desc:             "描述 2",
							Labels:           []string{"标签 2"},
							JD:               "这是JD 2",
							JDAnalysis:       "这是JD分析 2",
							Questions:        "这是面试问题 2",
							QuestionAnalysis: "这是问题分析 2",
							Resume:           "这是简历 2",
							Status:           domain.PublishedStatus.ToUint8(),
							Utime:            123,
							Interactive: web.Interactive{
								CollectCnt: 5,     // id + 3
								LikeCnt:    4,     // id + 2
								ViewCnt:    3,     // id + 1
								Liked:      false, // id 为偶数时为 false
								Collected:  true,  // id 为偶数时为 true
							},
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

func (s *TestSuite) TestPubDetail() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.DetailReq
		wantCode int
		wantResp test.Result[web.Review]
	}{
		{
			name: "获取已发布的记录",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				// 创建原始记录
				review := dao.Review{
					ID:    1,
					Uid:   uid,
					Title: "已发布的标题",
					Desc:  "已发布的描述",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"已发布的标签"},
					},
					JD:               "已发布的JD",
					JDAnalysis:       "已发布的JD分析",
					Questions:        "已发布的面试问题",
					QuestionAnalysis: "已发布的问题分析",
					Resume:           "已发布的简历",
					Status:           domain.PublishedStatus.ToUint8(),
				}
				_, err := s.reviewDao.Save(ctx, review)
				require.NoError(t, err)

				// 同步到发布表
				_, err = s.reviewDao.Sync(ctx, review)
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				s.assertCachedReview(t, domain.Review{
					ID:               1,
					Uid:              uid,
					Title:            "已发布的标题",
					Desc:             "已发布的描述",
					Labels:           []string{"已发布的标签"},
					JD:               "已发布的JD",
					JDAnalysis:       "已发布的JD分析",
					Questions:        "已发布的面试问题",
					QuestionAnalysis: "已发布的问题分析",
					Resume:           "已发布的简历",
					Status:           domain.PublishedStatus,
				})
			},
			req: web.DetailReq{
				ID: 1,
			},
			wantCode: 200,
			wantResp: test.Result[web.Review]{
				Data: web.Review{
					ID:               1,
					Title:            "已发布的标题",
					Desc:             "已发布的描述",
					Labels:           []string{"已发布的标签"},
					JD:               "已发布的JD",
					JDAnalysis:       "已发布的JD分析",
					Questions:        "已发布的面试问题",
					QuestionAnalysis: "已发布的问题分析",
					Resume:           "已发布的简历",
					Status:           domain.PublishedStatus.ToUint8(),
					Interactive: web.Interactive{
						CollectCnt: 4,
						LikeCnt:    3,
						ViewCnt:    2,
						Liked:      true,
						Collected:  false,
					},
				},
			},
		},
		{
			name: "直接命中缓存",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				re := domain.Review{
					ID:               1,
					Uid:              uid,
					Title:            "已发布的标题",
					Desc:             "已发布的描述",
					Labels:           []string{"已发布的标签"},
					JD:               "已发布的JD",
					JDAnalysis:       "已发布的JD分析",
					Questions:        "已发布的面试问题",
					QuestionAnalysis: "已发布的问题分析",
					Resume:           "已发布的简历",
					Status:           domain.PublishedStatus,
					Utime:            1111111,
				}
				reByte, err := json.Marshal(re)
				require.NoError(t, err)
				err = s.rdb.Set(ctx, "review:publish:1", string(reByte), 24*time.Hour)
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				s.assertCachedReview(t, domain.Review{
					ID:               1,
					Uid:              uid,
					Title:            "已发布的标题",
					Desc:             "已发布的描述",
					Labels:           []string{"已发布的标签"},
					JD:               "已发布的JD",
					JDAnalysis:       "已发布的JD分析",
					Questions:        "已发布的面试问题",
					QuestionAnalysis: "已发布的问题分析",
					Resume:           "已发布的简历",
					Status:           domain.PublishedStatus,
				})
			},
			req: web.DetailReq{
				ID: 1,
			},
			wantCode: 200,
			wantResp: test.Result[web.Review]{
				Data: web.Review{
					ID:               1,
					Title:            "已发布的标题",
					Desc:             "已发布的描述",
					Labels:           []string{"已发布的标签"},
					JD:               "已发布的JD",
					JDAnalysis:       "已发布的JD分析",
					Questions:        "已发布的面试问题",
					QuestionAnalysis: "已发布的问题分析",
					Resume:           "已发布的简历",
					Status:           domain.PublishedStatus.ToUint8(),
					Interactive: web.Interactive{
						CollectCnt: 4,
						LikeCnt:    3,
						ViewCnt:    2,
						Liked:      true,
						Collected:  false,
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
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
			resp := recorder.MustScan()
			assert.True(t, resp.Data.Utime != 0)
			resp.Data.Utime = 0
			assert.Equal(t, tc.wantResp, resp)
			tc.after(t)
			// 清理数据
			err = s.db.Exec("TRUNCATE table `reviews`").Error
			require.NoError(t, err)
			err = s.db.Exec("TRUNCATE table `publish_reviews`").Error
			require.NoError(t, err)
		})
	}
}

// assertReview 比较两个 Review 对象，忽略时间字段
func assertReview(t *testing.T, expect dao.Review, actual dao.Review) {
	require.True(t, actual.Ctime != 0)
	require.True(t, actual.Utime != 0)
	actual.Ctime = 0
	actual.Utime = 0

	assert.Equal(t, expect, actual)
}

func TestReviewHandler(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) assertCachedReview(t *testing.T, want domain.Review) {
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
