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

	"github.com/ecodeclub/webook/internal/member"
	membermocks "github.com/ecodeclub/webook/internal/member/mocks"

	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"
	"github.com/lithammer/shortuuid/v4"

	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/cases/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/cases/internal/web"
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

const uid = 2051

type HandlerTestSuite struct {
	suite.Suite
	server  *egin.Component
	db      *egorm.Component
	rdb     ecache.Cache
	dao     dao.CaseDAO
	examDAO dao.ExamineDAO
	ctrl    *gomock.Controller
}

func (s *HandlerTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `cases`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `publish_cases`").Error
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())
	intrSvc := intrmocks.NewMockService(s.ctrl)
	intrModule := &interactive.Module{
		Svc: intrSvc,
	}
	// 模拟返回的数据
	// 使用如下规律:
	// 1. liked == id % 2 == 1 (奇数为 true)
	// 2. collected = id %2 == 0 (偶数为 true)
	// 3. viewCnt = id + 1
	// 4. likeCnt = id + 2
	// 5. collectCnt = id + 3
	intrSvc.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(ctx context.Context, biz string, id int64, uid int64) (interactive.Interactive, error) {
			intr := s.mockInteractive(biz, id)
			return intr, nil
		})
	intrSvc.EXPECT().GetByIds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context,
		biz string, uid int64, ids []int64) (map[int64]interactive.Interactive, error) {
		res := make(map[int64]interactive.Interactive, len(ids))
		for _, id := range ids {
			intr := s.mockInteractive(biz, id)
			res[id] = intr
		}
		return res, nil
	}).AnyTimes()
	memSvc := membermocks.NewMockService(s.ctrl)
	memSvc.EXPECT().GetMembershipInfo(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (member.Member, error) {
		if id%2 == 0 {
			return member.Member{
				Uid:   uid,
				EndAt: time.Now().Add(10 * time.Hour).UnixMilli(),
			}, nil
		}
		return member.Member{
			Uid:   uid,
			EndAt: 0,
		}, nil
	}).AnyTimes()
	module, err := startup.InitModule(nil, nil, &ai.Module{}, &member.Module{
		Svc: memSvc,
	}, session.DefaultProvider(), intrModule)
	require.NoError(s.T(), err)
	handler := module.Hdl
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		notMember := ctx.GetHeader("not_member") == "1"
		notlogin := ctx.GetHeader("not_login") == "1"
		uidStr := ctx.GetHeader("uid")
		nuid := uid
		data := map[string]string{
			"creator": "true",
		}
		if uidStr != "" {
			nuid, err = strconv.Atoi(uidStr)
			require.NoError(s.T(), err)
		}

		// 如果是会员,添加memberDDL
		if !notMember {
			data["memberDDL"] = strconv.FormatInt(time.Now().Add(time.Hour).UnixMilli(), 10)
		}
		if notlogin {
			return
		}
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid:  int64(nuid),
			Data: data,
		}))
	})
	handler.PublicRoutes(server.Engine)
	server.Use(middleware.NewCheckMembershipMiddlewareBuilder(nil).Build())

	s.server = server
	s.db = testioc.InitDB()
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewCaseDao(s.db)
	s.examDAO = dao.NewGORMExamineDAO(s.db)
	s.rdb = testioc.InitCache()
}

func (s *HandlerTestSuite) TestPubList() {
	testCases := []struct {
		name     string
		req      web.Page
		before   func(t *testing.T)
		after    func(t *testing.T)
		wantCode int
		wantResp test.Result[web.CasesList]
	}{
		{
			name: "首次获取前50条，设置缓存",
			req: web.Page{
				Limit:  2,
				Offset: 0,
			},
			before: func(t *testing.T) {
				data := make([]dao.PublishCase, 0, 100)
				for idx := 0; idx < 100; idx++ {
					data = append(data, dao.PublishCase{
						Id:           int64(idx + 1),
						Uid:          uid,
						Title:        fmt.Sprintf("这是发布的案例标题 %d", idx),
						Introduction: fmt.Sprintf("这是发布的案例介绍 %d", idx),
						Utime:        1739779178000,
					})
				}
				err := s.db.Create(&data).Error
				require.NoError(s.T(), err)
			},
			after: func(t *testing.T) {
				// 校验缓存数据
				wantDomainCases := make([]domain.Case, 0, 50)
				index := 99
				for idx := 0; idx < 50; idx++ {
					wantDomainCases = append(wantDomainCases, domain.Case{
						Id:           int64(index - idx + 1),
						Title:        fmt.Sprintf("这是发布的案例标题 %d", index-idx),
						Introduction: fmt.Sprintf("这是发布的案例介绍 %d", index-idx),
						Utime:        time.UnixMilli(1739779178000),
					})
				}
				s.cacheAssertCaseList(domain.DefaultBiz, wantDomainCases)
			},
			wantCode: 200,
			wantResp: test.Result[web.CasesList]{
				Data: web.CasesList{
					Total: 100,
					Cases: []web.Case{
						{
							Id:           100,
							Title:        "这是发布的案例标题 99",
							Introduction: "这是发布的案例介绍 99",
							Utime:        1739779178000,
							Interactive: web.Interactive{
								Liked:      false,
								Collected:  true,
								ViewCnt:    101,
								LikeCnt:    102,
								CollectCnt: 103,
							},
						},
						{
							Id:           99,
							Title:        "这是发布的案例标题 98",
							Introduction: "这是发布的案例介绍 98",
							Utime:        1739779178000,
							Interactive: web.Interactive{
								Liked:      true,
								Collected:  false,
								ViewCnt:    100,
								LikeCnt:    101,
								CollectCnt: 102,
							},
						},
					},
				},
			},
		},
		{name: "命中缓存直接返回",
			req: web.Page{
				Limit:  2,
				Offset: 0,
			},
			before: func(t *testing.T) {
				// 直接设置缓存
				wantDomainCases := make([]domain.Case, 0, 50)
				index := 99
				for idx := 0; idx < 50; idx++ {
					wantDomainCases = append(wantDomainCases, domain.Case{
						Id:           int64(index - idx + 1),
						Title:        fmt.Sprintf("这是发布的案例标题 %d", index-idx),
						Introduction: fmt.Sprintf("这是发布的案例介绍 %d", index-idx),
						Utime:        time.UnixMilli(1739779178000),
					})
				}
				caseByte, err := json.Marshal(wantDomainCases)
				require.NoError(t, err)
				err = s.rdb.Set(context.Background(), "cases:list:baguwen", string(caseByte), 24*time.Hour)
				require.NoError(t, err)
				err = s.rdb.Set(context.Background(), "cases:total:baguwen", 100, 24*time.Hour)
				require.NoError(t, err)

			},
			after: func(t *testing.T) {
			},
			wantCode: 200,
			wantResp: test.Result[web.CasesList]{
				Data: web.CasesList{
					Total: 100,
					Cases: []web.Case{
						{
							Id:           100,
							Title:        "这是发布的案例标题 99",
							Introduction: "这是发布的案例介绍 99",
							Utime:        1739779178000,
							Interactive: web.Interactive{
								Liked:      false,
								Collected:  true,
								ViewCnt:    101,
								LikeCnt:    102,
								CollectCnt: 103,
							},
						},
						{
							Id:           99,
							Title:        "这是发布的案例标题 98",
							Introduction: "这是发布的案例介绍 98",
							Utime:        1739779178000,
							Interactive: web.Interactive{
								Liked:      true,
								Collected:  false,
								ViewCnt:    100,
								LikeCnt:    101,
								CollectCnt: 102,
							},
						},
					},
				},
			},
		},
		{
			name: "超出缓存范围走数据库",
			req: web.Page{
				Limit:  2,
				Offset: 99,
			},
			before: func(t *testing.T) {
				data := make([]dao.PublishCase, 0, 100)
				for idx := 0; idx < 100; idx++ {
					data = append(data, dao.PublishCase{
						Id:           int64(idx + 1),
						Title:        fmt.Sprintf("这是发布的案例标题 %d", idx),
						Introduction: fmt.Sprintf("这是发布的案例介绍 %d", idx),
						Utime:        1739779178000,
					})
				}
				err := s.db.Create(&data).Error
				require.NoError(s.T(), err)
			},
			after: func(t *testing.T) {
			},
			wantCode: 200,
			wantResp: test.Result[web.CasesList]{
				Data: web.CasesList{
					Total: 100,
					Cases: []web.Case{
						{
							Id:           1,
							Title:        "这是发布的案例标题 0",
							Introduction: "这是发布的案例介绍 0",
							Utime:        1739779178000,
							Interactive: web.Interactive{
								ViewCnt:    2,
								LikeCnt:    3,
								CollectCnt: 4,
								Liked:      true,
								Collected:  false,
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
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/case/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.CasesList]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
			err = s.db.Exec("TRUNCATE TABLE `cases`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE TABLE `publish_cases`").Error
			require.NoError(s.T(), err)
		})
	}
}

func (s *HandlerTestSuite) TestPubDetail() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := s.db.Create(&dao.PublishCase{
		Id:           3,
		Uid:          uid,
		Introduction: "redis案例介绍",
		Labels: sqlx.JsonColumn[[]string]{
			Valid: true,
			Val:   []string{"Redis"},
		},
		Status:     domain.PublishedStatus.ToUint8(),
		Title:      "redis案例标题",
		Content:    `<p>在微服务超时控制和重试策略里面，经常会引用这个案例。代码在 <a href="https://github.com/meoying/interview-cases/tree/main/case31_40/case32" rel="noopener noreferrer" target="_blank">interview-cases/case31_40/case32 at main · meoying/interview-cases (github.com)</a></p><p></p><h3>普通版</h3><p>普通版本就是使用一个普通的滑动窗口算法，关键点是：</p><ul><li>在窗口里面，记录了每个请求的时间戳，以及这个请求是否超时；</li><li>每个请求在收到了超时响应之后，统计这个窗口内部的超时请求数量；</li><li>如果超时请求数量超过阈值了，那么就认为不应该重试了；</li><li>如果超时请求数量没有超过阈值，那么就认为可以重试；</li></ul><p></p><h3>进阶版</h3><p>在高并发的情况下，滑动窗口算法的开销是难以忍受的。举个例子来说，即便将窗口限制在 1s，高并发的时候 1s 也有几万个请求，这几万个请求会占据你很多内存。而且在判定要不要重试的时候还要遍历，这也是一个极大的问题。</p><p></p><p>所以在进阶版里面，需要做优化：</p><ul><li>使用 ring buffer 来实现滑动窗口，进一步使用一个比特来标记请求是否超时；</li><li>每个请求超时之后，判定要不要执行重试，就要统计这个 ring buffer 里面为 1 的比特数量；</li><li>如果超时请求数量超过阈值了，那么就认为不应该重试；</li><li>如果超时请求数量没有超过阈值，那么就认为可以重试；</li><li>为了进一步提高性能，这里可以使用原子操作；</li></ul><p></p><p>举个例子来说，假设我们现在用 128 个比特来记录请求的超时信息，也就是说窗口大小是 128 个请求（不再是以时间来作为窗口大小了）。而后第一个请求过来标记下标 0 的比特，第二个请求标记下标 1 的比特...当都标记满了，就再次从头开始。</p><p></p><p>要注意，在我们的实现里面使用了大量的原子操作，所以你同样可以用这个案例来证明你很擅长并发编程。如果你觉得代码奇怪，请不要惊讶，它确实是对的，只是说违背一般的并发编程的模式而已，但是它确实是并发安全的，并且性能很好。</p><p></p><h3>适用场景和话术</h3><p>它可以用在这些场景：</p><ul><li>介绍你的提高可用性的方案，你可以将重试作为作为提高系统可用性的一环，而后介绍你这个规避了重试风暴的重试策略；</li><li>在面试中讨论到了超时问题</li><li>在面试中讨论到了重试问题，你可以主动提及重试风暴的问题，以及你解决了这个问题，也就是这个案例；</li></ul><p></p><p>而在介绍的时候，你要用一种演进的预期来介绍，也就是你不要上来就介绍进阶版，你要先介绍普通版，而后介绍进阶版。从这个演进中能够体现你对系统的思考，以及你是一个精益求精的人。</p><p></p><p>那么你可以参考这个话术：</p><blockquote>在处理大量请求时，我们经常会遇到超时的情况。为了合理控制重试行为，避免所谓的“重试风暴”，我设计了一个基于时间窗口的算法。在这个算法中，我们维护了一个滑动窗口，窗口内记录了每个请求的时间戳以及该请求是否超时。每当一个请求超时后，我们会统计窗口内超时的请求数量。如果超时请求的数量超过了设定的阈值，我们就认为当前系统压力较大，不适合进行重试；否则，我们认为可以安全地进行重试。</blockquote><blockquote>然而，随着并发量的增加，普通版的滑动窗口算法暴露出了一些问题。特别是在高并发场景下，窗口内需要维护的请求数量可能非常大，这不仅占用了大量内存，而且在判定是否需要重试时还需要遍历整个窗口，这大大增加了算法的时间复杂度。</blockquote><blockquote>为了解决这个问题，我们进一步设计了进阶版的算法。在这个版本中，我们引入了ring buffer 来优化滑动窗口的实现。具体来说，我们不再以时间为窗口大小，而是使用固定数量的比特位来记录请求的超时信息。每个比特位对应一个请求，用1表示超时，用0表示未超时。当所有比特位都被标记后，我们从头开始再次标记。</blockquote><blockquote>这种设计极大地降低了内存占用，因为无论并发量多高，我们只需要固定数量的比特位来记录请求的超时状态。同时，在判定是否需要重试时，我们只需要统计ring buffer中为1的比特数量，这大大简化了算法的实现并提高了效率。</blockquote><p></p><h3>模拟面试题</h3><p><strong>你的 ring buffer 设置得多大？</strong></p><blockquote>你可以随便说，并不需要很多。比如说你可以这么回答：</blockquote><blockquote>默认情况下是 128 字节，也就是 128 * 8 比特，1024 个比特。相当于每次都是只统计 1024 个请求中超时的比例</blockquote>`,
		GithubRepo: "redis github仓库",
		GiteeRepo:  "redis gitee仓库",
		Keywords:   "redis_keywords",
		Shorthand:  "redis_shorthand",
		Highlight:  "redis_highlight",
		Guidance:   "redis_guidance",
		Biz:        "ai",
		BizId:      13,
		Utime:      13,
	}).Error
	require.NoError(s.T(), err)
	// 插入测试记录
	err = s.examDAO.SaveResult(ctx, dao.CaseExamineRecord{
		Uid:    uid,
		Cid:    3,
		Tid:    shortuuid.New(),
		Result: 1,
	})
	require.NoError(s.T(), err)

	testCases := []struct {
		name   string
		before func(req *http.Request)
		// 校验数据
		after    func()
		req      web.CaseId
		wantCode int
		wantResp test.Result[web.Case]
	}{
		{
			name: "非会员获取部分数据",
			before: func(req *http.Request) {
				req.Header.Set("not_member", "1")
			},
			req: web.CaseId{
				Cid: 3,
			},
			after: func() {

			},
			wantCode: 200,
			wantResp: test.Result[web.Case]{
				Data: web.Case{
					Id:           3,
					Labels:       []string{"Redis"},
					Title:        "redis案例标题",
					Introduction: "redis案例介绍",
					Content:      `<p>在微服务超时控制和重试策略里面，经常会引用这个案例。代码在 <a href="https://github.com/meoying/interview-cases/tree/main/case31_40/case32" rel="noopener noreferrer" target="_blank">interview-cases/case31_40/case32 at main · meoying/interview-cases (github.com)</a></p><p></p><h3>普通版</h3><p>普通版本就是使用一个普通的滑动窗口算法，关键点是：</p><ul><li>在窗口里面，记录了每个请求的时间戳，以及这个请求是否超时；</li><li>每个请求在收到了超时响应之后，统计这个窗口内部的超时请求数量；</li><li>如果超时请求数量超过阈值了，那么就认为不应该重试了；</li><li>如果超时请求数量没有超过阈值，那么就认为可以重试；</li></ul><p></p><h3>进阶版</h3><p>在高并发的情况下，滑动窗口算法的开销是难以忍受的。举个例子来说，即便将窗口限制在 1s，高并发的时候 1s 也有几万个请求，这几万个请求会占据你很多内存。而且在判定要不要重试的时候还要遍历，这也是一个极大的问题。</p>`,
					GithubRepo:   "redis github仓库",
					GiteeRepo:    "redis gitee仓库",
					Status:       domain.PublishedStatus.ToUint8(),
					Keywords:     "redis_keywords",
					Shorthand:    "redis_shorthand",
					Highlight:    "redis_highlight",
					Guidance:     "redis_guidance",
					Biz:          "ai",
					BizId:        13,
					Utime:        13,
					Interactive: web.Interactive{
						Liked:      true,
						Collected:  false,
						ViewCnt:    4,
						LikeCnt:    5,
						CollectCnt: 6,
					},
					ExamineResult: 1,
				},
			},
		},
		{
			name: "未登录获取部分数据",
			before: func(req *http.Request) {
				req.Header.Set("not_login", "1")
			},
			req: web.CaseId{
				Cid: 3,
			},
			after: func() {

			},
			wantCode: 200,
			wantResp: test.Result[web.Case]{
				Data: web.Case{
					Id:           3,
					Labels:       []string{"Redis"},
					Title:        "redis案例标题",
					Introduction: "redis案例介绍",
					Content:      `<p>在微服务超时控制和重试策略里面，经常会引用这个案例。代码在 <a href="https://github.com/meoying/interview-cases/tree/main/case31_40/case32" rel="noopener noreferrer" target="_blank">interview-cases/case31_40/case32 at main · meoying/interview-cases (github.com)</a></p><p></p><h3>普通版</h3><p>普通版本就是使用一个普通的滑动窗口算法，关键点是：</p><ul><li>在窗口里面，记录了每个请求的时间戳，以及这个请求是否超时；</li><li>每个请求在收到了超时响应之后，统计这个窗口内部的超时请求数量；</li><li>如果超时请求数量超过阈值了，那么就认为不应该重试了；</li><li>如果超时请求数量没有超过阈值，那么就认为可以重试；</li></ul><p></p><h3>进阶版</h3><p>在高并发的情况下，滑动窗口算法的开销是难以忍受的。举个例子来说，即便将窗口限制在 1s，高并发的时候 1s 也有几万个请求，这几万个请求会占据你很多内存。而且在判定要不要重试的时候还要遍历，这也是一个极大的问题。</p>`,
					GithubRepo:   "redis github仓库",
					GiteeRepo:    "redis gitee仓库",
					Status:       domain.PublishedStatus.ToUint8(),
					Keywords:     "redis_keywords",
					Shorthand:    "redis_shorthand",
					Highlight:    "redis_highlight",
					Guidance:     "redis_guidance",
					Biz:          "ai",
					BizId:        13,
					Utime:        13,
					Interactive: web.Interactive{
						Liked:      true,
						Collected:  false,
						ViewCnt:    4,
						LikeCnt:    5,
						CollectCnt: 6,
					},
					ExamineResult: 0,
				},
			},
		},
		{
			name: "会员获取全部数据",
			before: func(req *http.Request) {
			},
			req: web.CaseId{
				Cid: 3,
			},
			after: func() {

			},
			wantCode: 200,
			wantResp: test.Result[web.Case]{
				Data: web.Case{
					Id:           3,
					Labels:       []string{"Redis"},
					Title:        "redis案例标题",
					Introduction: "redis案例介绍",
					Content:      `<p>在微服务超时控制和重试策略里面，经常会引用这个案例。代码在 <a href="https://github.com/meoying/interview-cases/tree/main/case31_40/case32" rel="noopener noreferrer" target="_blank">interview-cases/case31_40/case32 at main · meoying/interview-cases (github.com)</a></p><p></p><h3>普通版</h3><p>普通版本就是使用一个普通的滑动窗口算法，关键点是：</p><ul><li>在窗口里面，记录了每个请求的时间戳，以及这个请求是否超时；</li><li>每个请求在收到了超时响应之后，统计这个窗口内部的超时请求数量；</li><li>如果超时请求数量超过阈值了，那么就认为不应该重试了；</li><li>如果超时请求数量没有超过阈值，那么就认为可以重试；</li></ul><p></p><h3>进阶版</h3><p>在高并发的情况下，滑动窗口算法的开销是难以忍受的。举个例子来说，即便将窗口限制在 1s，高并发的时候 1s 也有几万个请求，这几万个请求会占据你很多内存。而且在判定要不要重试的时候还要遍历，这也是一个极大的问题。</p><p></p><p>所以在进阶版里面，需要做优化：</p><ul><li>使用 ring buffer 来实现滑动窗口，进一步使用一个比特来标记请求是否超时；</li><li>每个请求超时之后，判定要不要执行重试，就要统计这个 ring buffer 里面为 1 的比特数量；</li><li>如果超时请求数量超过阈值了，那么就认为不应该重试；</li><li>如果超时请求数量没有超过阈值，那么就认为可以重试；</li><li>为了进一步提高性能，这里可以使用原子操作；</li></ul><p></p><p>举个例子来说，假设我们现在用 128 个比特来记录请求的超时信息，也就是说窗口大小是 128 个请求（不再是以时间来作为窗口大小了）。而后第一个请求过来标记下标 0 的比特，第二个请求标记下标 1 的比特...当都标记满了，就再次从头开始。</p><p></p><p>要注意，在我们的实现里面使用了大量的原子操作，所以你同样可以用这个案例来证明你很擅长并发编程。如果你觉得代码奇怪，请不要惊讶，它确实是对的，只是说违背一般的并发编程的模式而已，但是它确实是并发安全的，并且性能很好。</p><p></p><h3>适用场景和话术</h3><p>它可以用在这些场景：</p><ul><li>介绍你的提高可用性的方案，你可以将重试作为作为提高系统可用性的一环，而后介绍你这个规避了重试风暴的重试策略；</li><li>在面试中讨论到了超时问题</li><li>在面试中讨论到了重试问题，你可以主动提及重试风暴的问题，以及你解决了这个问题，也就是这个案例；</li></ul><p></p><p>而在介绍的时候，你要用一种演进的预期来介绍，也就是你不要上来就介绍进阶版，你要先介绍普通版，而后介绍进阶版。从这个演进中能够体现你对系统的思考，以及你是一个精益求精的人。</p><p></p><p>那么你可以参考这个话术：</p><blockquote>在处理大量请求时，我们经常会遇到超时的情况。为了合理控制重试行为，避免所谓的“重试风暴”，我设计了一个基于时间窗口的算法。在这个算法中，我们维护了一个滑动窗口，窗口内记录了每个请求的时间戳以及该请求是否超时。每当一个请求超时后，我们会统计窗口内超时的请求数量。如果超时请求的数量超过了设定的阈值，我们就认为当前系统压力较大，不适合进行重试；否则，我们认为可以安全地进行重试。</blockquote><blockquote>然而，随着并发量的增加，普通版的滑动窗口算法暴露出了一些问题。特别是在高并发场景下，窗口内需要维护的请求数量可能非常大，这不仅占用了大量内存，而且在判定是否需要重试时还需要遍历整个窗口，这大大增加了算法的时间复杂度。</blockquote><blockquote>为了解决这个问题，我们进一步设计了进阶版的算法。在这个版本中，我们引入了ring buffer 来优化滑动窗口的实现。具体来说，我们不再以时间为窗口大小，而是使用固定数量的比特位来记录请求的超时信息。每个比特位对应一个请求，用1表示超时，用0表示未超时。当所有比特位都被标记后，我们从头开始再次标记。</blockquote><blockquote>这种设计极大地降低了内存占用，因为无论并发量多高，我们只需要固定数量的比特位来记录请求的超时状态。同时，在判定是否需要重试时，我们只需要统计ring buffer中为1的比特数量，这大大简化了算法的实现并提高了效率。</blockquote><p></p><h3>模拟面试题</h3><p><strong>你的 ring buffer 设置得多大？</strong></p><blockquote>你可以随便说，并不需要很多。比如说你可以这么回答：</blockquote><blockquote>默认情况下是 128 字节，也就是 128 * 8 比特，1024 个比特。相当于每次都是只统计 1024 个请求中超时的比例</blockquote>`,
					GithubRepo:   "redis github仓库",
					GiteeRepo:    "redis gitee仓库",
					Status:       domain.PublishedStatus.ToUint8(),
					Keywords:     "redis_keywords",
					Shorthand:    "redis_shorthand",
					Highlight:    "redis_highlight",
					Guidance:     "redis_guidance",
					Biz:          "ai",
					BizId:        13,
					Utime:        13,
					Interactive: web.Interactive{
						Liked:      true,
						Collected:  false,
						ViewCnt:    4,
						LikeCnt:    5,
						CollectCnt: 6,
					},
					ExamineResult: 1,
					Permitted:     true,
				},
			},
		},
		{
			name: "token中会员过期，但是是会员,返回全部数据",
			before: func(req *http.Request) {
				req.Header.Set("uid", "4")
			},
			req: web.CaseId{
				Cid: 3,
			},
			after: func() {

			},
			wantCode: 200,
			wantResp: test.Result[web.Case]{
				Data: web.Case{
					Id:           3,
					Labels:       []string{"Redis"},
					Title:        "redis案例标题",
					Introduction: "redis案例介绍",
					Content:      `<p>在微服务超时控制和重试策略里面，经常会引用这个案例。代码在 <a href="https://github.com/meoying/interview-cases/tree/main/case31_40/case32" rel="noopener noreferrer" target="_blank">interview-cases/case31_40/case32 at main · meoying/interview-cases (github.com)</a></p><p></p><h3>普通版</h3><p>普通版本就是使用一个普通的滑动窗口算法，关键点是：</p><ul><li>在窗口里面，记录了每个请求的时间戳，以及这个请求是否超时；</li><li>每个请求在收到了超时响应之后，统计这个窗口内部的超时请求数量；</li><li>如果超时请求数量超过阈值了，那么就认为不应该重试了；</li><li>如果超时请求数量没有超过阈值，那么就认为可以重试；</li></ul><p></p><h3>进阶版</h3><p>在高并发的情况下，滑动窗口算法的开销是难以忍受的。举个例子来说，即便将窗口限制在 1s，高并发的时候 1s 也有几万个请求，这几万个请求会占据你很多内存。而且在判定要不要重试的时候还要遍历，这也是一个极大的问题。</p><p></p><p>所以在进阶版里面，需要做优化：</p><ul><li>使用 ring buffer 来实现滑动窗口，进一步使用一个比特来标记请求是否超时；</li><li>每个请求超时之后，判定要不要执行重试，就要统计这个 ring buffer 里面为 1 的比特数量；</li><li>如果超时请求数量超过阈值了，那么就认为不应该重试；</li><li>如果超时请求数量没有超过阈值，那么就认为可以重试；</li><li>为了进一步提高性能，这里可以使用原子操作；</li></ul><p></p><p>举个例子来说，假设我们现在用 128 个比特来记录请求的超时信息，也就是说窗口大小是 128 个请求（不再是以时间来作为窗口大小了）。而后第一个请求过来标记下标 0 的比特，第二个请求标记下标 1 的比特...当都标记满了，就再次从头开始。</p><p></p><p>要注意，在我们的实现里面使用了大量的原子操作，所以你同样可以用这个案例来证明你很擅长并发编程。如果你觉得代码奇怪，请不要惊讶，它确实是对的，只是说违背一般的并发编程的模式而已，但是它确实是并发安全的，并且性能很好。</p><p></p><h3>适用场景和话术</h3><p>它可以用在这些场景：</p><ul><li>介绍你的提高可用性的方案，你可以将重试作为作为提高系统可用性的一环，而后介绍你这个规避了重试风暴的重试策略；</li><li>在面试中讨论到了超时问题</li><li>在面试中讨论到了重试问题，你可以主动提及重试风暴的问题，以及你解决了这个问题，也就是这个案例；</li></ul><p></p><p>而在介绍的时候，你要用一种演进的预期来介绍，也就是你不要上来就介绍进阶版，你要先介绍普通版，而后介绍进阶版。从这个演进中能够体现你对系统的思考，以及你是一个精益求精的人。</p><p></p><p>那么你可以参考这个话术：</p><blockquote>在处理大量请求时，我们经常会遇到超时的情况。为了合理控制重试行为，避免所谓的“重试风暴”，我设计了一个基于时间窗口的算法。在这个算法中，我们维护了一个滑动窗口，窗口内记录了每个请求的时间戳以及该请求是否超时。每当一个请求超时后，我们会统计窗口内超时的请求数量。如果超时请求的数量超过了设定的阈值，我们就认为当前系统压力较大，不适合进行重试；否则，我们认为可以安全地进行重试。</blockquote><blockquote>然而，随着并发量的增加，普通版的滑动窗口算法暴露出了一些问题。特别是在高并发场景下，窗口内需要维护的请求数量可能非常大，这不仅占用了大量内存，而且在判定是否需要重试时还需要遍历整个窗口，这大大增加了算法的时间复杂度。</blockquote><blockquote>为了解决这个问题，我们进一步设计了进阶版的算法。在这个版本中，我们引入了ring buffer 来优化滑动窗口的实现。具体来说，我们不再以时间为窗口大小，而是使用固定数量的比特位来记录请求的超时信息。每个比特位对应一个请求，用1表示超时，用0表示未超时。当所有比特位都被标记后，我们从头开始再次标记。</blockquote><blockquote>这种设计极大地降低了内存占用，因为无论并发量多高，我们只需要固定数量的比特位来记录请求的超时状态。同时，在判定是否需要重试时，我们只需要统计ring buffer中为1的比特数量，这大大简化了算法的实现并提高了效率。</blockquote><p></p><h3>模拟面试题</h3><p><strong>你的 ring buffer 设置得多大？</strong></p><blockquote>你可以随便说，并不需要很多。比如说你可以这么回答：</blockquote><blockquote>默认情况下是 128 字节，也就是 128 * 8 比特，1024 个比特。相当于每次都是只统计 1024 个请求中超时的比例</blockquote>`,
					GithubRepo:   "redis github仓库",
					GiteeRepo:    "redis gitee仓库",
					Status:       domain.PublishedStatus.ToUint8(),
					Keywords:     "redis_keywords",
					Shorthand:    "redis_shorthand",
					Highlight:    "redis_highlight",
					Guidance:     "redis_guidance",
					Biz:          "ai",
					BizId:        13,
					Utime:        13,
					Interactive: web.Interactive{
						Liked:      true,
						Collected:  false,
						ViewCnt:    4,
						LikeCnt:    5,
						CollectCnt: 6,
					},
					ExamineResult: 0,
					Permitted:     true,
				},
			},
		},
		{
			name: "未命中缓存",
			before: func(req *http.Request) {
				err = s.db.Create(&dao.PublishCase{
					Id:           4,
					Uid:          uid,
					Introduction: "redis案例介绍",
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"Redis"},
					},
					Status:     domain.PublishedStatus.ToUint8(),
					Title:      "redis案例标题",
					Content:    `123321`,
					GithubRepo: "redis github仓库",
					GiteeRepo:  "redis gitee仓库",
					Keywords:   "redis_keywords",
					Shorthand:  "redis_shorthand",
					Highlight:  "redis_highlight",
					Guidance:   "redis_guidance",
					Biz:        "ai",
					BizId:      13,
					Utime:      1739519892000,
					Ctime:      1739519892000,
				}).Error
				require.NoError(s.T(), err)
			},
			req: web.CaseId{
				Cid: 4,
			},
			wantCode: 200,
			after: func() {
				s.cacheAssertCase(domain.Case{
					Id:           4,
					Uid:          uid,
					Introduction: "redis案例介绍",
					Labels:       []string{"Redis"},
					Status:       domain.PublishedStatus,
					Title:        "redis案例标题",
					Content:      `123321`,
					GithubRepo:   "redis github仓库",
					GiteeRepo:    "redis gitee仓库",
					Keywords:     "redis_keywords",
					Shorthand:    "redis_shorthand",
					Highlight:    "redis_highlight",
					Guidance:     "redis_guidance",
					Biz:          "ai",
					BizId:        13,
				})
			},
			wantResp: test.Result[web.Case]{
				Data: web.Case{
					Id:           4,
					Introduction: "redis案例介绍",
					Labels:       []string{"Redis"},
					Status:       domain.PublishedStatus.ToUint8(),
					Title:        "redis案例标题",
					Content:      `123321`,
					GithubRepo:   "redis github仓库",
					GiteeRepo:    "redis gitee仓库",
					Keywords:     "redis_keywords",
					Shorthand:    "redis_shorthand",
					Highlight:    "redis_highlight",
					Guidance:     "redis_guidance",
					Biz:          "ai",
					BizId:        13,
					Utime:        1739519892000,
					Interactive: web.Interactive{
						Liked:      false,
						Collected:  true,
						ViewCnt:    5,
						LikeCnt:    6,
						CollectCnt: 7,
					},
					ExamineResult: 0,
					Permitted:     true,
				},
			},
		},
		{
			name: "命中缓存",
			before: func(req *http.Request) {
				ca := domain.Case{
					Id:           5,
					Uid:          uid,
					Introduction: "redis案例介绍",
					Labels:       []string{"Redis"},
					Status:       domain.PublishedStatus,
					Title:        "redis案例标题",
					Content:      `123321`,
					GithubRepo:   "redis github仓库",
					GiteeRepo:    "redis gitee仓库",
					Keywords:     "redis_keywords",
					Shorthand:    "redis_shorthand",
					Highlight:    "redis_highlight",
					Guidance:     "redis_guidance",
					Biz:          "ai",
					BizId:        13,
					Utime:        time.UnixMilli(1739519892000),
					Ctime:        time.UnixMilli(1739519892000),
				}
				caByte, _ := json.Marshal(ca)
				err = s.rdb.Set(context.Background(), "cases:publish:5", string(caByte), 24*time.Hour)
				require.NoError(s.T(), err)
			},
			req: web.CaseId{
				Cid: 5,
			},
			wantCode: 200,
			after: func() {
				s.cacheAssertCase(domain.Case{
					Id:           5,
					Uid:          uid,
					Introduction: "redis案例介绍",
					Labels:       []string{"Redis"},
					Status:       domain.PublishedStatus,
					Title:        "redis案例标题",
					Content:      `123321`,
					GithubRepo:   "redis github仓库",
					GiteeRepo:    "redis gitee仓库",
					Keywords:     "redis_keywords",
					Shorthand:    "redis_shorthand",
					Highlight:    "redis_highlight",
					Guidance:     "redis_guidance",
					Biz:          "ai",
					BizId:        13,
				})
			},
			wantResp: test.Result[web.Case]{
				Data: web.Case{
					Id:           5,
					Introduction: "redis案例介绍",
					Labels:       []string{"Redis"},
					Status:       domain.PublishedStatus.ToUint8(),
					Title:        "redis案例标题",
					Content:      `123321`,
					GithubRepo:   "redis github仓库",
					GiteeRepo:    "redis gitee仓库",
					Keywords:     "redis_keywords",
					Shorthand:    "redis_shorthand",
					Highlight:    "redis_highlight",
					Guidance:     "redis_guidance",
					Biz:          "ai",
					BizId:        13,
					Utime:        1739519892000,
					Interactive: web.Interactive{
						Liked:      true,
						Collected:  false,
						ViewCnt:    6,
						LikeCnt:    7,
						CollectCnt: 8,
					},
					ExamineResult: 0,
					Permitted:     true,
				},
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/case/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			tc.before(req)
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Case]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after()
		})
	}
}

func (s *HandlerTestSuite) cacheAssertCase(ca domain.Case) {
	t := s.T()
	key := fmt.Sprintf("cases:publish:%d", ca.Id)
	val := s.rdb.Get(context.Background(), key)
	require.NoError(t, val.Err)
	valStr, err := val.String()
	require.NoError(t, err)
	actualCa := domain.Case{}
	json.Unmarshal([]byte(valStr), &actualCa)
	require.True(t, actualCa.Ctime.Unix() > 0)
	require.True(t, actualCa.Utime.Unix() > 0)
	ca.Ctime = actualCa.Ctime
	ca.Utime = actualCa.Utime
	assert.Equal(t, ca, actualCa)
}

func (s *HandlerTestSuite) mockInteractive(biz string, id int64) interactive.Interactive {
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

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

func (s *HandlerTestSuite) cacheAssertCaseList(biz string, cases []domain.Case) {
	key := fmt.Sprintf("cases:list:%s", biz)
	val := s.rdb.Get(context.Background(), key)
	require.NoError(s.T(), val.Err)

	var cs []domain.Case
	err := json.Unmarshal([]byte(val.Val.(string)), &cs)
	require.NoError(s.T(), err)
	require.Equal(s.T(), len(cases), len(cs))
	for idx, q := range cs {
		require.True(s.T(), q.Utime.UnixMilli() > 0)
		require.True(s.T(), q.Id > 0)
		cs[idx].Id = cases[idx].Id
		cs[idx].Utime = cases[idx].Utime
		cs[idx].Ctime = cases[idx].Ctime

	}
	assert.Equal(s.T(), cases, cs)
	_, err = s.rdb.Delete(context.Background(), key)
	require.NoError(s.T(), err)
}
