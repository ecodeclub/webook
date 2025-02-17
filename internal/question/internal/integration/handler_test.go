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

	"github.com/ecodeclub/ekit/sqlx"

	"github.com/ecodeclub/webook/internal/member"
	membermocks "github.com/ecodeclub/webook/internal/member/mocks"

	"github.com/ecodeclub/webook/internal/ai"

	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"
	"github.com/ecodeclub/webook/internal/permission"
	permissionmocks "github.com/ecodeclub/webook/internal/permission/mocks"

	eveMocks "github.com/ecodeclub/webook/internal/question/internal/event/mocks"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/webook/internal/question/internal/domain"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/question/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/question/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/question/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const uid = 123

type HandlerTestSuite struct {
	BaseTestSuite
	server *egin.Component
	rdb    ecache.Cache
}

func (s *HandlerTestSuite) SetupSuite() {
	ctrl := gomock.NewController(s.T())
	producer := eveMocks.NewMockSyncEventProducer(ctrl)

	intrSvc := intrmocks.NewMockService(ctrl)
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
	intrSvc.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().DoAndReturn(func(ctx context.Context,
		biz string, id int64, uid int64) (interactive.Interactive, error) {
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

	permSvc := permissionmocks.NewMockService(ctrl)
	// biz id 为偶数就有权限
	permSvc.EXPECT().HasPermission(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context,
		perm permission.Permission) (bool, error) {
		return perm.BizID%2 == 0 && perm.Uid%2 == 1, nil
	}).AnyTimes()
	// uid 为偶数就有权限
	memSvc := membermocks.NewMockService(ctrl)
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
	module, err := startup.InitModule(producer, nil, intrModule,
		&permission.Module{Svc: permSvc}, &ai.Module{},
		session.DefaultProvider(),
		&member.Module{
			Svc: memSvc,
		})
	require.NoError(s.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()

	module.QsHdl.PublicRoutes(server.Engine)
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
	module.Hdl.PublicRoutes(server.Engine)

	s.server = server
	s.db = testioc.InitDB()
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.rdb = testioc.InitCache()
}

func (s *HandlerTestSuite) TestPubList() {

	testCases := []struct {
		name     string
		req      web.Page
		before   func(t *testing.T)
		after    func(t *testing.T)
		wantCode int
		wantResp test.Result[web.QuestionList]
	}{
		{
			name: "获取的数据位于前50条，未命中缓存，会写入缓存",
			req: web.Page{
				Limit:  2,
				Offset: 0,
			},
			before: func(t *testing.T) {
				// 插入一百条
				data := make([]dao.PublishQuestion, 0, 100)
				for idx := 0; idx < 100; idx++ {
					id := int64(idx + 1)
					data = append(data, dao.PublishQuestion{
						Id:      id,
						Uid:     uid,
						Biz:     domain.DefaultBiz,
						BizId:   id,
						Status:  domain.PublishedStatus.ToUint8(),
						Title:   fmt.Sprintf("这是标题 %d", idx),
						Content: fmt.Sprintf("这是解析 %d", idx),
						Utime:   123,
					})
				}
				// project 的不会被搜索到
				data = append(data, dao.PublishQuestion{
					Id:      101,
					Uid:     uid,
					Biz:     "project",
					BizId:   101,
					Status:  domain.PublishedStatus.ToUint8(),
					Title:   fmt.Sprintf("这是标题 %d", 101),
					Content: fmt.Sprintf("这是解析 %d", 101),
					Utime:   123,
				})
				err := s.db.Create(&data).Error
				require.NoError(s.T(), err)
			},
			wantCode: 200,
			wantResp: test.Result[web.QuestionList]{
				Data: web.QuestionList{
					Total: 100,
					Questions: []web.Question{
						{
							Id:      100,
							Title:   "这是标题 99",
							Content: "这是解析 99",
							Status:  domain.PublishedStatus.ToUint8(),
							Utime:   123,
							Biz:     domain.DefaultBiz,
							BizId:   100,
							Interactive: web.Interactive{
								ViewCnt:    101,
								LikeCnt:    102,
								CollectCnt: 103,
								Liked:      false,
								Collected:  true,
							},
						},
						{
							Id:      99,
							Title:   "这是标题 98",
							Content: "这是解析 98",
							Status:  domain.PublishedStatus.ToUint8(),
							Utime:   123,
							Biz:     domain.DefaultBiz,
							BizId:   99,
							Interactive: web.Interactive{
								ViewCnt:    100,
								LikeCnt:    101,
								CollectCnt: 102,
								Liked:      true,
								Collected:  false,
							},
						},
					},
				},
			},
			after: func(t *testing.T) {
				// 校验缓存中的数据
				wantDomainQuestions := make([]domain.Question, 0, 50)
				index := 99
				for idx := 0; idx < 50; idx++ {
					id := int64(index - idx + 1)
					wantDomainQuestions = append(wantDomainQuestions, domain.Question{
						Id:      id,
						Uid:     uid,
						Biz:     domain.DefaultBiz,
						BizId:   id,
						Status:  domain.PublishedStatus,
						Title:   fmt.Sprintf("这是标题 %d", index-idx),
						Content: fmt.Sprintf("这是解析 %d", index-idx),
					})
				}
				s.cacheAssertQuestionList(domain.DefaultBiz, wantDomainQuestions)
				_, err := s.rdb.Delete(context.Background(), "question:total")
				require.NoError(s.T(), err)
			},
		},
		{
			name: "获取的数据位于前50条，命中缓存，直接返回",
			req: web.Page{
				Limit:  2,
				Offset: 0,
			},
			before: func(t *testing.T) {
				// 只写缓存
				wantDomainQuestions := make([]domain.Question, 0, 50)
				index := 99
				for idx := 0; idx < 50; idx++ {
					id := int64(index - idx + 1)
					wantDomainQuestions = append(wantDomainQuestions, domain.Question{
						Id:      id,
						Uid:     uid,
						Biz:     domain.DefaultBiz,
						BizId:   id,
						Utime:   time.UnixMilli(1739779178000),
						Status:  domain.PublishedStatus,
						Title:   fmt.Sprintf("这是标题 %d", index-idx),
						Content: fmt.Sprintf("这是解析 %d", index-idx),
					})
				}
				queByte, err := json.Marshal(wantDomainQuestions)
				require.NoError(t, err)
				err = s.rdb.Set(context.Background(), "question:list:baguwen", string(queByte), 24*time.Hour)
				require.NoError(t, err)
			},
			wantCode: 200,
			wantResp: test.Result[web.QuestionList]{
				Data: web.QuestionList{
					Total: 100,
					Questions: []web.Question{
						{
							Id:      100,
							Title:   "这是标题 99",
							Content: "这是解析 99",
							Status:  domain.PublishedStatus.ToUint8(),
							Utime:   1739779178000,
							Biz:     domain.DefaultBiz,
							BizId:   100,
							Interactive: web.Interactive{
								ViewCnt:    101,
								LikeCnt:    102,
								CollectCnt: 103,
								Liked:      false,
								Collected:  true,
							},
						},
						{
							Id:      99,
							Title:   "这是标题 98",
							Content: "这是解析 98",
							Status:  domain.PublishedStatus.ToUint8(),
							Utime:   1739779178000,
							Biz:     domain.DefaultBiz,
							BizId:   99,
							Interactive: web.Interactive{
								ViewCnt:    100,
								LikeCnt:    101,
								CollectCnt: 102,
								Liked:      true,
								Collected:  false,
							},
						},
					},
				},
			},
			after: func(t *testing.T) {
			},
		},
		{
			name: "获取部分，不在前50从数据库中返回",
			req: web.Page{
				Limit:  2,
				Offset: 99,
			},
			before: func(t *testing.T) {
				// 插入一百条
				data := make([]dao.PublishQuestion, 0, 100)
				for idx := 0; idx < 100; idx++ {
					id := int64(idx + 1)
					data = append(data, dao.PublishQuestion{
						Id:      id,
						Uid:     uid,
						Biz:     domain.DefaultBiz,
						BizId:   id,
						Status:  domain.PublishedStatus.ToUint8(),
						Title:   fmt.Sprintf("这是标题 %d", idx),
						Content: fmt.Sprintf("这是解析 %d", idx),
						Utime:   123,
					})
				}
				// project 的不会被搜索到
				data = append(data, dao.PublishQuestion{
					Id:      101,
					Uid:     uid,
					Biz:     "project",
					BizId:   101,
					Status:  domain.PublishedStatus.ToUint8(),
					Title:   fmt.Sprintf("这是标题 %d", 101),
					Content: fmt.Sprintf("这是解析 %d", 101),
					Utime:   123,
				})
				err := s.db.Create(&data).Error
				require.NoError(s.T(), err)
			},
			after: func(t *testing.T) {
				key := fmt.Sprintf("question:list:%s", domain.DefaultBiz)
				val := s.rdb.Get(context.Background(), key)
				require.True(t, val.KeyNotFound())
			},
			wantCode: 200,
			wantResp: test.Result[web.QuestionList]{
				Data: web.QuestionList{
					Total: 100,
					Questions: []web.Question{
						{
							Id:      1,
							Title:   "这是标题 0",
							Content: "这是解析 0",
							Biz:     domain.DefaultBiz,
							BizId:   1,
							Status:  domain.PublishedStatus.ToUint8(),
							Utime:   123,
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
		{
			name: "有部分在前五十，有部分不在。命中数据库直接返回",
			req: web.Page{
				Limit:  3,
				Offset: 48,
			},
			before: func(t *testing.T) {
				// 插入一百条
				data := make([]dao.PublishQuestion, 0, 100)
				for idx := 0; idx < 100; idx++ {
					id := int64(idx + 1)
					data = append(data, dao.PublishQuestion{
						Id:      id,
						Uid:     uid,
						Biz:     domain.DefaultBiz,
						BizId:   id,
						Status:  domain.PublishedStatus.ToUint8(),
						Title:   fmt.Sprintf("这是标题 %d", idx),
						Content: fmt.Sprintf("这是解析 %d", idx),
						Utime:   123,
					})
				}
				// project 的不会被搜索到
				data = append(data, dao.PublishQuestion{
					Id:      101,
					Uid:     uid,
					Biz:     "project",
					BizId:   101,
					Status:  domain.PublishedStatus.ToUint8(),
					Title:   fmt.Sprintf("这是标题 %d", 101),
					Content: fmt.Sprintf("这是解析 %d", 101),
					Utime:   123,
				})
				err := s.db.Create(&data).Error
				require.NoError(s.T(), err)
			},
			after: func(t *testing.T) {
				key := fmt.Sprintf("question:list:%s", domain.DefaultBiz)
				val := s.rdb.Get(context.Background(), key)
				require.True(t, val.KeyNotFound())
			},
			wantCode: 200,
			wantResp: test.Result[web.QuestionList]{
				Data: web.QuestionList{
					Total: 100,
					Questions: []web.Question{
						{
							Id:      52, // 缓存最后第二条（100-48=52）
							Title:   "这是标题 51",
							Content: "这是解析 51",
							Utime:   123,
							Status:  domain.PublishedStatus.ToUint8(),
							Biz:     domain.DefaultBiz,
							BizId:   52,
							Interactive: web.Interactive{
								ViewCnt:    53,
								LikeCnt:    54,
								CollectCnt: 55,
								Collected:  true,
							},
						},
						{
							Id:      51, // 缓存最后一条
							Title:   "这是标题 50",
							Content: "这是解析 50",
							Utime:   123,
							Status:  domain.PublishedStatus.ToUint8(),
							Biz:     domain.DefaultBiz,
							BizId:   51,
							Interactive: web.Interactive{
								ViewCnt:    52,
								LikeCnt:    53,
								CollectCnt: 54,
								Liked:      true,
							},
						},
						{
							Id:      50, // 数据库第一条（按倒序查询）
							Title:   "这是标题 49",
							Content: "这是解析 49",
							Utime:   123,
							Status:  domain.PublishedStatus.ToUint8(),
							Biz:     domain.DefaultBiz,
							BizId:   50,
							Interactive: web.Interactive{
								ViewCnt:    51,
								LikeCnt:    52,
								CollectCnt: 53,
								Collected:  true,
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
				"/question/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.QuestionList]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			data := recorder.MustScan()
			assert.Equal(t, tc.wantResp, data)
			tc.after(t)
			err = s.db.Exec("TRUNCATE TABLE `questions`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE TABLE `publish_questions`").Error
			require.NoError(s.T(), err)
			_, err = s.rdb.Delete(context.Background(), "question:list:baguwen")
			require.NoError(s.T(), err)
		})
	}

}

func (s *HandlerTestSuite) TestPubDetail() {
	s.initData()
	testcases := []struct {
		name     string
		req      web.Qid
		before   func(req *http.Request)
		after    func()
		wantData web.Question
	}{
		{
			name: "没有会员返回部分数据",
			req: web.Qid{
				Qid: 1041,
			},
			before: func(req *http.Request) {
				req.Header.Set("not_member", "1")
			},
			after: func() {

			},
			wantData: web.Question{
				Id:      1041,
				BizId:   0,
				Biz:     "baguwen",
				Status:  domain.PublishedStatus.ToUint8(),
				Utime:   321,
				Title:   `在微服务架构中，如何处理服务实例的动态变化（如上线、下线、故障）？`,
				Content: `<p>略难的题，一般只会出现在社招中。</p><p></p><p>其实这种问法会让你觉得摸不着头脑，但是如果你把问题换成如果服务实例动态变化了，注册中心和客户端会怎样，就清晰多了。要在这个问题之下刷亮点，赢得竞争优势，你可以讨论客户端容错策略，以及高并发场景下服务实例频繁变化会给注册中心带来庞大的压力这两个点。</p>`,
				Analysis: web.AnswerElement{
					Id:      6110,
					Content: `<p>前置知识：</p><ul><li><a href="https://wsn.com/question/detail" rel="noopener noreferrer" target="_blank">你知道注册中心吗？</a></li></ul><p></p><p>在服务注册与发现中，服务健康检查是确保服务实例可用性的重要机制。通过健康检查，注册中心可以动态感知服务实例的状态变化（如健康、故障、下线等），从而保障消费者调用的服务始终可用。常见的健康检查方式主要有以下两类：</p><p></p><ol><li>主动健康检查：主动健康检查由注册中心或消费者主动发起探测请求，定期检测服务实例的健康状态。常见实现方式包括：<ul><li>HTTP 检查：注册中心向服务实例的健康检查端点（如 /health）发送 HTTP 请求，根据返回状态码（如 2xx）判断健康状态。<ul><li>优点：简单易用，适合 HTTP 服务。</li><li>缺点：只能检测服务的基本可达性，无法深入检测内部状态。</li><li>示例：Spring Boot Actuator 提供了 /actuator/health 端点，Nacos 和 Consul 支持通过 HTTP 检查服务健康。</li></ul></li><li>TCP 检查：注册中心尝试连接服务实例的指定端口，判断端口是否可用。<ul><li>优点：适合非 HTTP 服务（如数据库、消息队列）。</li><li>缺点：仅能检测端口连通性，无法反映业务逻辑状态。</li><li>示例：Consul 支持通过 TCP 检查服务端口。</li></ul></li><li>gRPC 检查：注册中心调用服务实例的 gRPC 健康检查接口（如 grpc.health.v1.Health/Check），判断服务是否健康。<ul><li>优点：适用于 gRPC 服务，通信高效。</li><li>缺点：需要服务实例实现 gRPC 健康检查接口。</li><li>示例：gRPC 官方提供了健康检查协议，适用于 gRPC 服务。</li></ul></li><li>自定义脚本检查：注册中心通过运行自定义脚本或命令检测服务状态。<ul><li>优点：灵活性高，可根据业务需求定制。</li><li>缺点：实现复杂，可能增加系统开销。</li><li>示例：Consul 支持通过 Shell 脚本实现自定义健康检查。</li></ul></li></ul></li><li>被动健康检查：被动健康检查通过监控服务实例的运行状态或调用结果，间接判断健康状况。常见实现方式包括：<ul><li>心跳检测：服务实例定期向注册中心发送心跳信号。如果在规定时间内未收到心跳，则认为实例不可用。<ul><li>优点：实现简单，适合大规模服务实例监控。</li><li>缺点：无法检测服务内部的业务逻辑状态。</li><li>示例：Eureka 和 Nacos 使用心跳机制维持服务健康状态。</li></ul></li><li>请求失败率监控：注册中心或消费者监控服务实例的请求失败率（如超时、错误响应等），当失败率超过阈值时，将实例标记为不可用。<ul><li>优点：能反映服务的实际运行状态。</li><li>缺点：需要额外的监控逻辑，可能存在延迟。</li><li>示例：Hystrix 和 Sentinel 可基于失败率隔离故障实例。</li></ul></li><li>日志监控：通过分析服务实例的运行日志，检测是否存在异常（如错误日志、超时日志）。<ul><li>优点：能深入了解服务运行状态。</li><li>缺点：实现复杂，实时性较差。</li><li>示例：使用 ELK（Elasticsearch、Logstash、Kibana）分析服务日志。</li></ul></li></ul></li></ol><p></p><p>在实际场景中，单一健康检查方式往往不足以全面反映服务状态，因此通常结合多种方式使用，并通过优化策略提升效率和准确性。例如：</p>`,
				},
				Basic: web.AnswerElement{
					Id:        6111,
					Content:   `<p>在微服务架构中，服务实例的动态变化很常见，比如服务实例因为扩容、缩容或者故障而上下线。为了保障系统的稳定性和高可用性，我们需要一套完善的机制来处理这些变化，主要从以下几个方面入手：</p><p></p><p>首先是服务注册中心的动态管理。注册中心是处理服务实例动态变化的核心，它主要通过心跳检测和实例状态同步来实现。注册中心会定期检测实例的健康状态，如果超时未响应，就会移除实例。注册中心要把实例状态的变化实时同步给消费者。同步方式有推送和拉取两种，在大规模分布式系统场景下，可以采用增量同步、注册中心分区或分层架构等优化策略。</p><p></p><p>其次是服务消费者侧的容错策略。服务消费者这边需要一些容错策略来应对服务实例的动态变化。比如：服务消费者可以通过缓存和动态更新来应对实例的变化。还可以采用负载均衡策略，比如轮询、加权轮询、一致性哈希等，来分发流量。以及一些容错机制，比如重试机制、熔断机制、降级处理和限流保护等，确保在部分实例不可用时，服务仍然可以正常运行。</p>`,
					Guidance:  `心跳检测；健康检查；负载均衡；一致性哈希；重试；熔断；限流；降级；扩容；缩容；平滑处理；`,
					Shorthand: `注册中心实时监测，客户端做好容错，扩容要平滑；`,
				},
				Intermediate: web.AnswerElement{
					Id:        6112,
					Guidance:  "客户端容错；failover",
					Highlight: "客户端处理注册信息异常的容错机制；",
					Content:   `<p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">在服务实例节点动态变化的时候，最经常遇到的问题就是注册中心不能及时地将最新的状态同步给客户端，因此客户端容错就非常重要。</span></p>`,
				},
				Advanced: web.AnswerElement{
					Id:        6113,
					Content:   `<p>而在大规模分布式系统下，如果服务节点动态变化频繁，那么会给注册中心带来庞大的压力。</p><p></p><p>一方面如果变化是服务节点主动上线下线引起的，那么它们就会触发写操作，更新注册中心中注册的信息。</p>`,
					Guidance:  "CAP；",
					Highlight: "节点变化在大规模集群下对注册中心的影响",
					Shorthand: "注册中心选AP不选CP；",
				},
				Interactive: web.Interactive{
					CollectCnt: 1044,
					LikeCnt:    1043,
					ViewCnt:    1042,
					Liked:      true,
				},
				ExamineResult: 2,
			},
		},
		{
			name: "会员返回全部数据",
			req: web.Qid{
				Qid: 1041,
			},
			after: func() {

			},
			before: func(req *http.Request) {},
			wantData: web.Question{
				Id:      1041,
				BizId:   0,
				Biz:     "baguwen",
				Status:  domain.PublishedStatus.ToUint8(),
				Utime:   321,
				Title:   `在微服务架构中，如何处理服务实例的动态变化（如上线、下线、故障）？`,
				Content: `<p>略难的题，一般只会出现在社招中。</p><p></p><p>其实这种问法会让你觉得摸不着头脑，但是如果你把问题换成如果服务实例动态变化了，注册中心和客户端会怎样，就清晰多了。要在这个问题之下刷亮点，赢得竞争优势，你可以讨论客户端容错策略，以及高并发场景下服务实例频繁变化会给注册中心带来庞大的压力这两个点。</p>`,
				Analysis: web.AnswerElement{
					Id:      6110,
					Content: `<p>前置知识：</p><ul><li><a href="https://wsn.com/question/detail" rel="noopener noreferrer" target="_blank">你知道注册中心吗？</a></li></ul><p></p><p>在服务注册与发现中，服务健康检查是确保服务实例可用性的重要机制。通过健康检查，注册中心可以动态感知服务实例的状态变化（如健康、故障、下线等），从而保障消费者调用的服务始终可用。常见的健康检查方式主要有以下两类：</p><p></p><ol><li>主动健康检查：主动健康检查由注册中心或消费者主动发起探测请求，定期检测服务实例的健康状态。常见实现方式包括：<ul><li>HTTP 检查：注册中心向服务实例的健康检查端点（如 /health）发送 HTTP 请求，根据返回状态码（如 2xx）判断健康状态。<ul><li>优点：简单易用，适合 HTTP 服务。</li><li>缺点：只能检测服务的基本可达性，无法深入检测内部状态。</li><li>示例：Spring Boot Actuator 提供了 /actuator/health 端点，Nacos 和 Consul 支持通过 HTTP 检查服务健康。</li></ul></li><li>TCP 检查：注册中心尝试连接服务实例的指定端口，判断端口是否可用。<ul><li>优点：适合非 HTTP 服务（如数据库、消息队列）。</li><li>缺点：仅能检测端口连通性，无法反映业务逻辑状态。</li><li>示例：Consul 支持通过 TCP 检查服务端口。</li></ul></li><li>gRPC 检查：注册中心调用服务实例的 gRPC 健康检查接口（如 grpc.health.v1.Health/Check），判断服务是否健康。<ul><li>优点：适用于 gRPC 服务，通信高效。</li><li>缺点：需要服务实例实现 gRPC 健康检查接口。</li><li>示例：gRPC 官方提供了健康检查协议，适用于 gRPC 服务。</li></ul></li><li>自定义脚本检查：注册中心通过运行自定义脚本或命令检测服务状态。<ul><li>优点：灵活性高，可根据业务需求定制。</li><li>缺点：实现复杂，可能增加系统开销。</li><li>示例：Consul 支持通过 Shell 脚本实现自定义健康检查。</li></ul></li></ul></li><li>被动健康检查：被动健康检查通过监控服务实例的运行状态或调用结果，间接判断健康状况。常见实现方式包括：<ul><li>心跳检测：服务实例定期向注册中心发送心跳信号。如果在规定时间内未收到心跳，则认为实例不可用。<ul><li>优点：实现简单，适合大规模服务实例监控。</li><li>缺点：无法检测服务内部的业务逻辑状态。</li><li>示例：Eureka 和 Nacos 使用心跳机制维持服务健康状态。</li></ul></li><li>请求失败率监控：注册中心或消费者监控服务实例的请求失败率（如超时、错误响应等），当失败率超过阈值时，将实例标记为不可用。<ul><li>优点：能反映服务的实际运行状态。</li><li>缺点：需要额外的监控逻辑，可能存在延迟。</li><li>示例：Hystrix 和 Sentinel 可基于失败率隔离故障实例。</li></ul></li><li>日志监控：通过分析服务实例的运行日志，检测是否存在异常（如错误日志、超时日志）。<ul><li>优点：能深入了解服务运行状态。</li><li>缺点：实现复杂，实时性较差。</li><li>示例：使用 ELK（Elasticsearch、Logstash、Kibana）分析服务日志。</li></ul></li></ul></li></ol><p></p><p>在实际场景中，单一健康检查方式往往不足以全面反映服务状态，因此通常结合多种方式使用，并通过优化策略提升效率和准确性。例如：</p><ul><li>组合检查：<ul><li>主动 + 被动检查：通过 HTTP 检查服务的基本可达性，同时结合心跳检测判断服务是否仍然活跃。</li><li>多级检查：先通过 TCP 检查端口连通性，再通过 HTTP 检查服务业务逻辑状态。</li></ul></li><li>优化策略：<ul><li>调整检查频率：根据服务的重要性和负载情况，合理设置检查频率，避免过于频繁导致性能开销。</li><li>健康状态缓存：对健康检查结果进行短时间缓存，减少重复检查的开销。</li><li>多次失败判定：避免因短暂网络波动或服务抖动导致误判，可设置连续多次失败后才标记为不可用。</li><li>分布式健康检查：在大规模分布式系统中，将健康检查任务分散到多个节点，降低注册中心的压力。</li></ul></li></ul><p>在复杂场景中，还可以基于以下方式提升健康检查的深度和智能化：</p><ul><li><ul><li>依赖检查：检测服务依赖的资源（如数据库、缓存）是否正常。</li><li>业务指标检查：通过关键业务指标（如订单处理速度）判断服务健康状态。</li><li>AI大模型预测：利用AI大模型分析历史数据，提前预测潜在故障。</li></ul></li></ul><p></p><p>服务健康检查是服务注册与发现的关键环节，常见方式包括主动健康检查（如 HTTP、TCP、gRPC、自定义脚本）和被动健康检查（如心跳检测、失败率监控、日志分析）。主动检查适合检测服务的基本可达性，被动检查更能反映服务的实际运行状态。在实际应用中，通常结合多种方式，并通过优化策略提升健康检查的效率和准确性，从而保障微服务架构的稳定性和可用性。</p>`,
				},
				Basic: web.AnswerElement{
					Id:        6111,
					Content:   `<p>在微服务架构中，服务实例的动态变化很常见，比如服务实例因为扩容、缩容或者故障而上下线。为了保障系统的稳定性和高可用性，我们需要一套完善的机制来处理这些变化，主要从以下几个方面入手：</p><p></p><p>首先是服务注册中心的动态管理。注册中心是处理服务实例动态变化的核心，它主要通过心跳检测和实例状态同步来实现。注册中心会定期检测实例的健康状态，如果超时未响应，就会移除实例。注册中心要把实例状态的变化实时同步给消费者。同步方式有推送和拉取两种，在大规模分布式系统场景下，可以采用增量同步、注册中心分区或分层架构等优化策略。</p><p></p><p>其次是服务消费者侧的容错策略。服务消费者这边需要一些容错策略来应对服务实例的动态变化。比如：服务消费者可以通过缓存和动态更新来应对实例的变化。还可以采用负载均衡策略，比如轮询、加权轮询、一致性哈希等，来分发流量。以及一些容错机制，比如重试机制、熔断机制、降级处理和限流保护等，确保在部分实例不可用时，服务仍然可以正常运行。</p><p></p><p>最后是动态扩容与缩容的平滑处理。扩容时，新服务实例上线后，注册中心会自动注册并同步给消费者，负载均衡组件会逐步增加新实例的权重，实现流量的平滑过渡。缩容时，在下线实例之前，会逐步减少它的权重，等它处理完已有请求后再注销，避免流量损失。</p><p></p><p>总而言之，在微服务架构中，服务实例的动态变化是不可避免的，而应对这些变化的关键就在于服务注册中心和服务消费者的协同配合。</p>`,
					Guidance:  `心跳检测；健康检查；负载均衡；一致性哈希；重试；熔断；限流；降级；扩容；缩容；平滑处理；`,
					Shorthand: `注册中心实时监测，客户端做好容错，扩容要平滑；`,
				},
				Intermediate: web.AnswerElement{
					Id:        6112,
					Guidance:  "客户端容错；failover",
					Highlight: "客户端处理注册信息异常的容错机制；",
					Content:   `<p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">在服务实例节点动态变化的时候，最经常遇到的问题就是注册中心不能及时地将最新的状态同步给客户端，因此客户端容错就非常重要。</span></p><p></p><p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">而容错的做法其实也不难。举个例子来说，客户端在发现调用不通服务端的时候，可以考虑换一个节点重试。在这种最简单的做法之上，还可以考虑引入一些高级的做法，例如说当一个节点频繁的调用不通的时候，客户端可以考虑将该节点标记为不可用。而后客户端尝试向服务端发送心跳，如果心跳恢复了，则认为服务端节点已经恢复了，可以继续发送请求。</span></p><p></p><p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">我在 gRPC 里面就使用过类似的策略，有效提高了我们系统的可用性和稳定性。</span></p><p></p>`,
				},
				Advanced: web.AnswerElement{
					Id:        6113,
					Content:   `<p>而在大规模分布式系统下，如果服务节点动态变化频繁，那么会给注册中心带来庞大的压力。</p><p></p><p>一方面如果变化是服务节点主动上线下线引起的，那么它们就会触发写操作，更新注册中心中注册的信息。</p><p></p><p>另外一方面来说，如果注册中心的设计是实时同步，那么每一次变动注册中心都要通知客户端，这会导致注册中心和客户端之间频繁通信。</p><p></p><p>所以，现在部署大规模分布式微服务架构的时候，通常都是在 CAP 中选择 AP 模型来保证注册中心的高可用，同时确保注册中心能够撑住频繁的节点变化。</p>`,
					Guidance:  "CAP；",
					Highlight: "节点变化在大规模集群下对注册中心的影响",
					Shorthand: "注册中心选AP不选CP；",
				},
				Interactive: web.Interactive{
					CollectCnt: 1044,
					LikeCnt:    1043,
					ViewCnt:    1042,
					Liked:      true,
				},
				ExamineResult: 2,
				Permitted:     true,
			},
		},
		{
			name: "token中会员过期，但是是会员,返回全部数据",
			req: web.Qid{
				Qid: 1041,
			},
			after: func() {

			},
			before: func(req *http.Request) {
				req.Header.Set("uid", "4")
			},
			wantData: web.Question{
				Id:      1041,
				BizId:   0,
				Biz:     "baguwen",
				Status:  domain.PublishedStatus.ToUint8(),
				Utime:   321,
				Title:   `在微服务架构中，如何处理服务实例的动态变化（如上线、下线、故障）？`,
				Content: `<p>略难的题，一般只会出现在社招中。</p><p></p><p>其实这种问法会让你觉得摸不着头脑，但是如果你把问题换成如果服务实例动态变化了，注册中心和客户端会怎样，就清晰多了。要在这个问题之下刷亮点，赢得竞争优势，你可以讨论客户端容错策略，以及高并发场景下服务实例频繁变化会给注册中心带来庞大的压力这两个点。</p>`,
				Analysis: web.AnswerElement{
					Id:      6110,
					Content: `<p>前置知识：</p><ul><li><a href="https://wsn.com/question/detail" rel="noopener noreferrer" target="_blank">你知道注册中心吗？</a></li></ul><p></p><p>在服务注册与发现中，服务健康检查是确保服务实例可用性的重要机制。通过健康检查，注册中心可以动态感知服务实例的状态变化（如健康、故障、下线等），从而保障消费者调用的服务始终可用。常见的健康检查方式主要有以下两类：</p><p></p><ol><li>主动健康检查：主动健康检查由注册中心或消费者主动发起探测请求，定期检测服务实例的健康状态。常见实现方式包括：<ul><li>HTTP 检查：注册中心向服务实例的健康检查端点（如 /health）发送 HTTP 请求，根据返回状态码（如 2xx）判断健康状态。<ul><li>优点：简单易用，适合 HTTP 服务。</li><li>缺点：只能检测服务的基本可达性，无法深入检测内部状态。</li><li>示例：Spring Boot Actuator 提供了 /actuator/health 端点，Nacos 和 Consul 支持通过 HTTP 检查服务健康。</li></ul></li><li>TCP 检查：注册中心尝试连接服务实例的指定端口，判断端口是否可用。<ul><li>优点：适合非 HTTP 服务（如数据库、消息队列）。</li><li>缺点：仅能检测端口连通性，无法反映业务逻辑状态。</li><li>示例：Consul 支持通过 TCP 检查服务端口。</li></ul></li><li>gRPC 检查：注册中心调用服务实例的 gRPC 健康检查接口（如 grpc.health.v1.Health/Check），判断服务是否健康。<ul><li>优点：适用于 gRPC 服务，通信高效。</li><li>缺点：需要服务实例实现 gRPC 健康检查接口。</li><li>示例：gRPC 官方提供了健康检查协议，适用于 gRPC 服务。</li></ul></li><li>自定义脚本检查：注册中心通过运行自定义脚本或命令检测服务状态。<ul><li>优点：灵活性高，可根据业务需求定制。</li><li>缺点：实现复杂，可能增加系统开销。</li><li>示例：Consul 支持通过 Shell 脚本实现自定义健康检查。</li></ul></li></ul></li><li>被动健康检查：被动健康检查通过监控服务实例的运行状态或调用结果，间接判断健康状况。常见实现方式包括：<ul><li>心跳检测：服务实例定期向注册中心发送心跳信号。如果在规定时间内未收到心跳，则认为实例不可用。<ul><li>优点：实现简单，适合大规模服务实例监控。</li><li>缺点：无法检测服务内部的业务逻辑状态。</li><li>示例：Eureka 和 Nacos 使用心跳机制维持服务健康状态。</li></ul></li><li>请求失败率监控：注册中心或消费者监控服务实例的请求失败率（如超时、错误响应等），当失败率超过阈值时，将实例标记为不可用。<ul><li>优点：能反映服务的实际运行状态。</li><li>缺点：需要额外的监控逻辑，可能存在延迟。</li><li>示例：Hystrix 和 Sentinel 可基于失败率隔离故障实例。</li></ul></li><li>日志监控：通过分析服务实例的运行日志，检测是否存在异常（如错误日志、超时日志）。<ul><li>优点：能深入了解服务运行状态。</li><li>缺点：实现复杂，实时性较差。</li><li>示例：使用 ELK（Elasticsearch、Logstash、Kibana）分析服务日志。</li></ul></li></ul></li></ol><p></p><p>在实际场景中，单一健康检查方式往往不足以全面反映服务状态，因此通常结合多种方式使用，并通过优化策略提升效率和准确性。例如：</p><ul><li>组合检查：<ul><li>主动 + 被动检查：通过 HTTP 检查服务的基本可达性，同时结合心跳检测判断服务是否仍然活跃。</li><li>多级检查：先通过 TCP 检查端口连通性，再通过 HTTP 检查服务业务逻辑状态。</li></ul></li><li>优化策略：<ul><li>调整检查频率：根据服务的重要性和负载情况，合理设置检查频率，避免过于频繁导致性能开销。</li><li>健康状态缓存：对健康检查结果进行短时间缓存，减少重复检查的开销。</li><li>多次失败判定：避免因短暂网络波动或服务抖动导致误判，可设置连续多次失败后才标记为不可用。</li><li>分布式健康检查：在大规模分布式系统中，将健康检查任务分散到多个节点，降低注册中心的压力。</li></ul></li></ul><p>在复杂场景中，还可以基于以下方式提升健康检查的深度和智能化：</p><ul><li><ul><li>依赖检查：检测服务依赖的资源（如数据库、缓存）是否正常。</li><li>业务指标检查：通过关键业务指标（如订单处理速度）判断服务健康状态。</li><li>AI大模型预测：利用AI大模型分析历史数据，提前预测潜在故障。</li></ul></li></ul><p></p><p>服务健康检查是服务注册与发现的关键环节，常见方式包括主动健康检查（如 HTTP、TCP、gRPC、自定义脚本）和被动健康检查（如心跳检测、失败率监控、日志分析）。主动检查适合检测服务的基本可达性，被动检查更能反映服务的实际运行状态。在实际应用中，通常结合多种方式，并通过优化策略提升健康检查的效率和准确性，从而保障微服务架构的稳定性和可用性。</p>`,
				},
				Basic: web.AnswerElement{
					Id:        6111,
					Content:   `<p>在微服务架构中，服务实例的动态变化很常见，比如服务实例因为扩容、缩容或者故障而上下线。为了保障系统的稳定性和高可用性，我们需要一套完善的机制来处理这些变化，主要从以下几个方面入手：</p><p></p><p>首先是服务注册中心的动态管理。注册中心是处理服务实例动态变化的核心，它主要通过心跳检测和实例状态同步来实现。注册中心会定期检测实例的健康状态，如果超时未响应，就会移除实例。注册中心要把实例状态的变化实时同步给消费者。同步方式有推送和拉取两种，在大规模分布式系统场景下，可以采用增量同步、注册中心分区或分层架构等优化策略。</p><p></p><p>其次是服务消费者侧的容错策略。服务消费者这边需要一些容错策略来应对服务实例的动态变化。比如：服务消费者可以通过缓存和动态更新来应对实例的变化。还可以采用负载均衡策略，比如轮询、加权轮询、一致性哈希等，来分发流量。以及一些容错机制，比如重试机制、熔断机制、降级处理和限流保护等，确保在部分实例不可用时，服务仍然可以正常运行。</p><p></p><p>最后是动态扩容与缩容的平滑处理。扩容时，新服务实例上线后，注册中心会自动注册并同步给消费者，负载均衡组件会逐步增加新实例的权重，实现流量的平滑过渡。缩容时，在下线实例之前，会逐步减少它的权重，等它处理完已有请求后再注销，避免流量损失。</p><p></p><p>总而言之，在微服务架构中，服务实例的动态变化是不可避免的，而应对这些变化的关键就在于服务注册中心和服务消费者的协同配合。</p>`,
					Guidance:  `心跳检测；健康检查；负载均衡；一致性哈希；重试；熔断；限流；降级；扩容；缩容；平滑处理；`,
					Shorthand: `注册中心实时监测，客户端做好容错，扩容要平滑；`,
				},
				Intermediate: web.AnswerElement{
					Id:        6112,
					Guidance:  "客户端容错；failover",
					Highlight: "客户端处理注册信息异常的容错机制；",
					Content:   `<p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">在服务实例节点动态变化的时候，最经常遇到的问题就是注册中心不能及时地将最新的状态同步给客户端，因此客户端容错就非常重要。</span></p><p></p><p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">而容错的做法其实也不难。举个例子来说，客户端在发现调用不通服务端的时候，可以考虑换一个节点重试。在这种最简单的做法之上，还可以考虑引入一些高级的做法，例如说当一个节点频繁的调用不通的时候，客户端可以考虑将该节点标记为不可用。而后客户端尝试向服务端发送心跳，如果心跳恢复了，则认为服务端节点已经恢复了，可以继续发送请求。</span></p><p></p><p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">我在 gRPC 里面就使用过类似的策略，有效提高了我们系统的可用性和稳定性。</span></p><p></p>`,
				},
				Advanced: web.AnswerElement{
					Id:        6113,
					Content:   `<p>而在大规模分布式系统下，如果服务节点动态变化频繁，那么会给注册中心带来庞大的压力。</p><p></p><p>一方面如果变化是服务节点主动上线下线引起的，那么它们就会触发写操作，更新注册中心中注册的信息。</p><p></p><p>另外一方面来说，如果注册中心的设计是实时同步，那么每一次变动注册中心都要通知客户端，这会导致注册中心和客户端之间频繁通信。</p><p></p><p>所以，现在部署大规模分布式微服务架构的时候，通常都是在 CAP 中选择 AP 模型来保证注册中心的高可用，同时确保注册中心能够撑住频繁的节点变化。</p>`,
					Guidance:  "CAP；",
					Highlight: "节点变化在大规模集群下对注册中心的影响",
					Shorthand: "注册中心选AP不选CP；",
				},
				Interactive: web.Interactive{
					CollectCnt: 1044,
					LikeCnt:    1043,
					ViewCnt:    1042,
					Liked:      true,
				},
				ExamineResult: 0,
				Permitted:     true,
			},
		},
		{
			name: "没有登录返回部分数据",
			req: web.Qid{
				Qid: 1041,
			},
			after: func() {

			},
			before: func(req *http.Request) {
				req.Header.Set("not_login", "1")
			},
			wantData: web.Question{
				Id:      1041,
				BizId:   0,
				Biz:     "baguwen",
				Status:  domain.PublishedStatus.ToUint8(),
				Utime:   321,
				Title:   `在微服务架构中，如何处理服务实例的动态变化（如上线、下线、故障）？`,
				Content: `<p>略难的题，一般只会出现在社招中。</p><p></p><p>其实这种问法会让你觉得摸不着头脑，但是如果你把问题换成如果服务实例动态变化了，注册中心和客户端会怎样，就清晰多了。要在这个问题之下刷亮点，赢得竞争优势，你可以讨论客户端容错策略，以及高并发场景下服务实例频繁变化会给注册中心带来庞大的压力这两个点。</p>`,
				Analysis: web.AnswerElement{
					Id:      6110,
					Content: `<p>前置知识：</p><ul><li><a href="https://wsn.com/question/detail" rel="noopener noreferrer" target="_blank">你知道注册中心吗？</a></li></ul><p></p><p>在服务注册与发现中，服务健康检查是确保服务实例可用性的重要机制。通过健康检查，注册中心可以动态感知服务实例的状态变化（如健康、故障、下线等），从而保障消费者调用的服务始终可用。常见的健康检查方式主要有以下两类：</p><p></p><ol><li>主动健康检查：主动健康检查由注册中心或消费者主动发起探测请求，定期检测服务实例的健康状态。常见实现方式包括：<ul><li>HTTP 检查：注册中心向服务实例的健康检查端点（如 /health）发送 HTTP 请求，根据返回状态码（如 2xx）判断健康状态。<ul><li>优点：简单易用，适合 HTTP 服务。</li><li>缺点：只能检测服务的基本可达性，无法深入检测内部状态。</li><li>示例：Spring Boot Actuator 提供了 /actuator/health 端点，Nacos 和 Consul 支持通过 HTTP 检查服务健康。</li></ul></li><li>TCP 检查：注册中心尝试连接服务实例的指定端口，判断端口是否可用。<ul><li>优点：适合非 HTTP 服务（如数据库、消息队列）。</li><li>缺点：仅能检测端口连通性，无法反映业务逻辑状态。</li><li>示例：Consul 支持通过 TCP 检查服务端口。</li></ul></li><li>gRPC 检查：注册中心调用服务实例的 gRPC 健康检查接口（如 grpc.health.v1.Health/Check），判断服务是否健康。<ul><li>优点：适用于 gRPC 服务，通信高效。</li><li>缺点：需要服务实例实现 gRPC 健康检查接口。</li><li>示例：gRPC 官方提供了健康检查协议，适用于 gRPC 服务。</li></ul></li><li>自定义脚本检查：注册中心通过运行自定义脚本或命令检测服务状态。<ul><li>优点：灵活性高，可根据业务需求定制。</li><li>缺点：实现复杂，可能增加系统开销。</li><li>示例：Consul 支持通过 Shell 脚本实现自定义健康检查。</li></ul></li></ul></li><li>被动健康检查：被动健康检查通过监控服务实例的运行状态或调用结果，间接判断健康状况。常见实现方式包括：<ul><li>心跳检测：服务实例定期向注册中心发送心跳信号。如果在规定时间内未收到心跳，则认为实例不可用。<ul><li>优点：实现简单，适合大规模服务实例监控。</li><li>缺点：无法检测服务内部的业务逻辑状态。</li><li>示例：Eureka 和 Nacos 使用心跳机制维持服务健康状态。</li></ul></li><li>请求失败率监控：注册中心或消费者监控服务实例的请求失败率（如超时、错误响应等），当失败率超过阈值时，将实例标记为不可用。<ul><li>优点：能反映服务的实际运行状态。</li><li>缺点：需要额外的监控逻辑，可能存在延迟。</li><li>示例：Hystrix 和 Sentinel 可基于失败率隔离故障实例。</li></ul></li><li>日志监控：通过分析服务实例的运行日志，检测是否存在异常（如错误日志、超时日志）。<ul><li>优点：能深入了解服务运行状态。</li><li>缺点：实现复杂，实时性较差。</li><li>示例：使用 ELK（Elasticsearch、Logstash、Kibana）分析服务日志。</li></ul></li></ul></li></ol><p></p><p>在实际场景中，单一健康检查方式往往不足以全面反映服务状态，因此通常结合多种方式使用，并通过优化策略提升效率和准确性。例如：</p>`,
				},
				Basic: web.AnswerElement{
					Id:        6111,
					Content:   `<p>在微服务架构中，服务实例的动态变化很常见，比如服务实例因为扩容、缩容或者故障而上下线。为了保障系统的稳定性和高可用性，我们需要一套完善的机制来处理这些变化，主要从以下几个方面入手：</p><p></p><p>首先是服务注册中心的动态管理。注册中心是处理服务实例动态变化的核心，它主要通过心跳检测和实例状态同步来实现。注册中心会定期检测实例的健康状态，如果超时未响应，就会移除实例。注册中心要把实例状态的变化实时同步给消费者。同步方式有推送和拉取两种，在大规模分布式系统场景下，可以采用增量同步、注册中心分区或分层架构等优化策略。</p><p></p><p>其次是服务消费者侧的容错策略。服务消费者这边需要一些容错策略来应对服务实例的动态变化。比如：服务消费者可以通过缓存和动态更新来应对实例的变化。还可以采用负载均衡策略，比如轮询、加权轮询、一致性哈希等，来分发流量。以及一些容错机制，比如重试机制、熔断机制、降级处理和限流保护等，确保在部分实例不可用时，服务仍然可以正常运行。</p>`,
					Guidance:  `心跳检测；健康检查；负载均衡；一致性哈希；重试；熔断；限流；降级；扩容；缩容；平滑处理；`,
					Shorthand: `注册中心实时监测，客户端做好容错，扩容要平滑；`,
				},
				Intermediate: web.AnswerElement{
					Id:        6112,
					Guidance:  "客户端容错；failover",
					Highlight: "客户端处理注册信息异常的容错机制；",
					Content:   `<p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">在服务实例节点动态变化的时候，最经常遇到的问题就是注册中心不能及时地将最新的状态同步给客户端，因此客户端容错就非常重要。</span></p>`,
				},
				Advanced: web.AnswerElement{
					Id:        6113,
					Content:   `<p>而在大规模分布式系统下，如果服务节点动态变化频繁，那么会给注册中心带来庞大的压力。</p><p></p><p>一方面如果变化是服务节点主动上线下线引起的，那么它们就会触发写操作，更新注册中心中注册的信息。</p>`,
					Guidance:  "CAP；",
					Highlight: "节点变化在大规模集群下对注册中心的影响",
					Shorthand: "注册中心选AP不选CP；",
				},
				Interactive: web.Interactive{
					CollectCnt: 1044,
					LikeCnt:    1043,
					ViewCnt:    1042,
					Liked:      true,
				},
				ExamineResult: 0,
			},
		},
		{
			name: "没有权限返回部分数据",
			req: web.Qid{
				Qid: 1042,
			},
			before: func(req *http.Request) {
				req.Header.Set("uid", "4")
			},
			after: func() {

			},
			wantData: web.Question{
				Id:      1042,
				BizId:   32,
				Biz:     "project",
				Status:  domain.PublishedStatus.ToUint8(),
				Utime:   321,
				Title:   `在微服务架构中，如何处理服务实例的动态变化（如上线、下线、故障）？`,
				Content: `<p>略难的题，一般只会出现在社招中。</p><p></p><p>其实这种问法会让你觉得摸不着头脑，但是如果你把问题换成如果服务实例动态变化了，注册中心和客户端会怎样，就清晰多了。要在这个问题之下刷亮点，赢得竞争优势，你可以讨论客户端容错策略，以及高并发场景下服务实例频繁变化会给注册中心带来庞大的压力这两个点。</p>`,
				Analysis: web.AnswerElement{
					Id:      5110,
					Content: `<p>前置知识：</p><ul><li><a href="https://wsn.com/question/detail" rel="noopener noreferrer" target="_blank">你知道注册中心吗？</a></li></ul><p></p><p>在服务注册与发现中，服务健康检查是确保服务实例可用性的重要机制。通过健康检查，注册中心可以动态感知服务实例的状态变化（如健康、故障、下线等），从而保障消费者调用的服务始终可用。常见的健康检查方式主要有以下两类：</p><p></p><ol><li>主动健康检查：主动健康检查由注册中心或消费者主动发起探测请求，定期检测服务实例的健康状态。常见实现方式包括：<ul><li>HTTP 检查：注册中心向服务实例的健康检查端点（如 /health）发送 HTTP 请求，根据返回状态码（如 2xx）判断健康状态。<ul><li>优点：简单易用，适合 HTTP 服务。</li><li>缺点：只能检测服务的基本可达性，无法深入检测内部状态。</li><li>示例：Spring Boot Actuator 提供了 /actuator/health 端点，Nacos 和 Consul 支持通过 HTTP 检查服务健康。</li></ul></li><li>TCP 检查：注册中心尝试连接服务实例的指定端口，判断端口是否可用。<ul><li>优点：适合非 HTTP 服务（如数据库、消息队列）。</li><li>缺点：仅能检测端口连通性，无法反映业务逻辑状态。</li><li>示例：Consul 支持通过 TCP 检查服务端口。</li></ul></li><li>gRPC 检查：注册中心调用服务实例的 gRPC 健康检查接口（如 grpc.health.v1.Health/Check），判断服务是否健康。<ul><li>优点：适用于 gRPC 服务，通信高效。</li><li>缺点：需要服务实例实现 gRPC 健康检查接口。</li><li>示例：gRPC 官方提供了健康检查协议，适用于 gRPC 服务。</li></ul></li><li>自定义脚本检查：注册中心通过运行自定义脚本或命令检测服务状态。<ul><li>优点：灵活性高，可根据业务需求定制。</li><li>缺点：实现复杂，可能增加系统开销。</li><li>示例：Consul 支持通过 Shell 脚本实现自定义健康检查。</li></ul></li></ul></li><li>被动健康检查：被动健康检查通过监控服务实例的运行状态或调用结果，间接判断健康状况。常见实现方式包括：<ul><li>心跳检测：服务实例定期向注册中心发送心跳信号。如果在规定时间内未收到心跳，则认为实例不可用。<ul><li>优点：实现简单，适合大规模服务实例监控。</li><li>缺点：无法检测服务内部的业务逻辑状态。</li><li>示例：Eureka 和 Nacos 使用心跳机制维持服务健康状态。</li></ul></li><li>请求失败率监控：注册中心或消费者监控服务实例的请求失败率（如超时、错误响应等），当失败率超过阈值时，将实例标记为不可用。<ul><li>优点：能反映服务的实际运行状态。</li><li>缺点：需要额外的监控逻辑，可能存在延迟。</li><li>示例：Hystrix 和 Sentinel 可基于失败率隔离故障实例。</li></ul></li><li>日志监控：通过分析服务实例的运行日志，检测是否存在异常（如错误日志、超时日志）。<ul><li>优点：能深入了解服务运行状态。</li><li>缺点：实现复杂，实时性较差。</li><li>示例：使用 ELK（Elasticsearch、Logstash、Kibana）分析服务日志。</li></ul></li></ul></li></ol><p></p><p>在实际场景中，单一健康检查方式往往不足以全面反映服务状态，因此通常结合多种方式使用，并通过优化策略提升效率和准确性。例如：</p>`,
				},
				Basic: web.AnswerElement{
					Id:        5111,
					Content:   `<p>在微服务架构中，服务实例的动态变化很常见，比如服务实例因为扩容、缩容或者故障而上下线。为了保障系统的稳定性和高可用性，我们需要一套完善的机制来处理这些变化，主要从以下几个方面入手：</p><p></p><p>首先是服务注册中心的动态管理。注册中心是处理服务实例动态变化的核心，它主要通过心跳检测和实例状态同步来实现。注册中心会定期检测实例的健康状态，如果超时未响应，就会移除实例。注册中心要把实例状态的变化实时同步给消费者。同步方式有推送和拉取两种，在大规模分布式系统场景下，可以采用增量同步、注册中心分区或分层架构等优化策略。</p><p></p><p>其次是服务消费者侧的容错策略。服务消费者这边需要一些容错策略来应对服务实例的动态变化。比如：服务消费者可以通过缓存和动态更新来应对实例的变化。还可以采用负载均衡策略，比如轮询、加权轮询、一致性哈希等，来分发流量。以及一些容错机制，比如重试机制、熔断机制、降级处理和限流保护等，确保在部分实例不可用时，服务仍然可以正常运行。</p>`,
					Guidance:  `心跳检测；健康检查；负载均衡；一致性哈希；重试；熔断；限流；降级；扩容；缩容；平滑处理；`,
					Shorthand: `注册中心实时监测，客户端做好容错，扩容要平滑；`,
				},
				Intermediate: web.AnswerElement{
					Id:        5113,
					Guidance:  "客户端容错；failover",
					Highlight: "客户端处理注册信息异常的容错机制；",
					Content:   `<p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">在服务实例节点动态变化的时候，最经常遇到的问题就是注册中心不能及时地将最新的状态同步给客户端，因此客户端容错就非常重要。</span></p>`,
				},
				Advanced: web.AnswerElement{
					Id:        5114,
					Content:   `<p>而在大规模分布式系统下，如果服务节点动态变化频繁，那么会给注册中心带来庞大的压力。</p><p></p><p>一方面如果变化是服务节点主动上线下线引起的，那么它们就会触发写操作，更新注册中心中注册的信息。</p>`,
					Guidance:  "CAP；",
					Highlight: "节点变化在大规模集群下对注册中心的影响",
					Shorthand: "注册中心选AP不选CP；",
				},
				Interactive: web.Interactive{
					CollectCnt: 1045,
					LikeCnt:    1044,
					ViewCnt:    1043,
					Collected:  true,
				},
				ExamineResult: 0,
			},
		},
		{
			name: "有权限返回全部数据",
			req: web.Qid{
				Qid: 1042,
			},
			after: func() {

			},
			before: func(req *http.Request) {
			},
			wantData: web.Question{
				Id:      1042,
				BizId:   32,
				Biz:     "project",
				Status:  domain.PublishedStatus.ToUint8(),
				Utime:   321,
				Title:   `在微服务架构中，如何处理服务实例的动态变化（如上线、下线、故障）？`,
				Content: `<p>略难的题，一般只会出现在社招中。</p><p></p><p>其实这种问法会让你觉得摸不着头脑，但是如果你把问题换成如果服务实例动态变化了，注册中心和客户端会怎样，就清晰多了。要在这个问题之下刷亮点，赢得竞争优势，你可以讨论客户端容错策略，以及高并发场景下服务实例频繁变化会给注册中心带来庞大的压力这两个点。</p>`,
				Analysis: web.AnswerElement{
					Id:      5110,
					Content: `<p>前置知识：</p><ul><li><a href="https://wsn.com/question/detail" rel="noopener noreferrer" target="_blank">你知道注册中心吗？</a></li></ul><p></p><p>在服务注册与发现中，服务健康检查是确保服务实例可用性的重要机制。通过健康检查，注册中心可以动态感知服务实例的状态变化（如健康、故障、下线等），从而保障消费者调用的服务始终可用。常见的健康检查方式主要有以下两类：</p><p></p><ol><li>主动健康检查：主动健康检查由注册中心或消费者主动发起探测请求，定期检测服务实例的健康状态。常见实现方式包括：<ul><li>HTTP 检查：注册中心向服务实例的健康检查端点（如 /health）发送 HTTP 请求，根据返回状态码（如 2xx）判断健康状态。<ul><li>优点：简单易用，适合 HTTP 服务。</li><li>缺点：只能检测服务的基本可达性，无法深入检测内部状态。</li><li>示例：Spring Boot Actuator 提供了 /actuator/health 端点，Nacos 和 Consul 支持通过 HTTP 检查服务健康。</li></ul></li><li>TCP 检查：注册中心尝试连接服务实例的指定端口，判断端口是否可用。<ul><li>优点：适合非 HTTP 服务（如数据库、消息队列）。</li><li>缺点：仅能检测端口连通性，无法反映业务逻辑状态。</li><li>示例：Consul 支持通过 TCP 检查服务端口。</li></ul></li><li>gRPC 检查：注册中心调用服务实例的 gRPC 健康检查接口（如 grpc.health.v1.Health/Check），判断服务是否健康。<ul><li>优点：适用于 gRPC 服务，通信高效。</li><li>缺点：需要服务实例实现 gRPC 健康检查接口。</li><li>示例：gRPC 官方提供了健康检查协议，适用于 gRPC 服务。</li></ul></li><li>自定义脚本检查：注册中心通过运行自定义脚本或命令检测服务状态。<ul><li>优点：灵活性高，可根据业务需求定制。</li><li>缺点：实现复杂，可能增加系统开销。</li><li>示例：Consul 支持通过 Shell 脚本实现自定义健康检查。</li></ul></li></ul></li><li>被动健康检查：被动健康检查通过监控服务实例的运行状态或调用结果，间接判断健康状况。常见实现方式包括：<ul><li>心跳检测：服务实例定期向注册中心发送心跳信号。如果在规定时间内未收到心跳，则认为实例不可用。<ul><li>优点：实现简单，适合大规模服务实例监控。</li><li>缺点：无法检测服务内部的业务逻辑状态。</li><li>示例：Eureka 和 Nacos 使用心跳机制维持服务健康状态。</li></ul></li><li>请求失败率监控：注册中心或消费者监控服务实例的请求失败率（如超时、错误响应等），当失败率超过阈值时，将实例标记为不可用。<ul><li>优点：能反映服务的实际运行状态。</li><li>缺点：需要额外的监控逻辑，可能存在延迟。</li><li>示例：Hystrix 和 Sentinel 可基于失败率隔离故障实例。</li></ul></li><li>日志监控：通过分析服务实例的运行日志，检测是否存在异常（如错误日志、超时日志）。<ul><li>优点：能深入了解服务运行状态。</li><li>缺点：实现复杂，实时性较差。</li><li>示例：使用 ELK（Elasticsearch、Logstash、Kibana）分析服务日志。</li></ul></li></ul></li></ol><p></p><p>在实际场景中，单一健康检查方式往往不足以全面反映服务状态，因此通常结合多种方式使用，并通过优化策略提升效率和准确性。例如：</p><ul><li>组合检查：<ul><li>主动 + 被动检查：通过 HTTP 检查服务的基本可达性，同时结合心跳检测判断服务是否仍然活跃。</li><li>多级检查：先通过 TCP 检查端口连通性，再通过 HTTP 检查服务业务逻辑状态。</li></ul></li><li>优化策略：<ul><li>调整检查频率：根据服务的重要性和负载情况，合理设置检查频率，避免过于频繁导致性能开销。</li><li>健康状态缓存：对健康检查结果进行短时间缓存，减少重复检查的开销。</li><li>多次失败判定：避免因短暂网络波动或服务抖动导致误判，可设置连续多次失败后才标记为不可用。</li><li>分布式健康检查：在大规模分布式系统中，将健康检查任务分散到多个节点，降低注册中心的压力。</li></ul></li></ul><p>在复杂场景中，还可以基于以下方式提升健康检查的深度和智能化：</p><ul><li><ul><li>依赖检查：检测服务依赖的资源（如数据库、缓存）是否正常。</li><li>业务指标检查：通过关键业务指标（如订单处理速度）判断服务健康状态。</li><li>AI大模型预测：利用AI大模型分析历史数据，提前预测潜在故障。</li></ul></li></ul><p></p><p>服务健康检查是服务注册与发现的关键环节，常见方式包括主动健康检查（如 HTTP、TCP、gRPC、自定义脚本）和被动健康检查（如心跳检测、失败率监控、日志分析）。主动检查适合检测服务的基本可达性，被动检查更能反映服务的实际运行状态。在实际应用中，通常结合多种方式，并通过优化策略提升健康检查的效率和准确性，从而保障微服务架构的稳定性和可用性。</p>`,
				},
				Basic: web.AnswerElement{
					Id:        5111,
					Content:   `<p>在微服务架构中，服务实例的动态变化很常见，比如服务实例因为扩容、缩容或者故障而上下线。为了保障系统的稳定性和高可用性，我们需要一套完善的机制来处理这些变化，主要从以下几个方面入手：</p><p></p><p>首先是服务注册中心的动态管理。注册中心是处理服务实例动态变化的核心，它主要通过心跳检测和实例状态同步来实现。注册中心会定期检测实例的健康状态，如果超时未响应，就会移除实例。注册中心要把实例状态的变化实时同步给消费者。同步方式有推送和拉取两种，在大规模分布式系统场景下，可以采用增量同步、注册中心分区或分层架构等优化策略。</p><p></p><p>其次是服务消费者侧的容错策略。服务消费者这边需要一些容错策略来应对服务实例的动态变化。比如：服务消费者可以通过缓存和动态更新来应对实例的变化。还可以采用负载均衡策略，比如轮询、加权轮询、一致性哈希等，来分发流量。以及一些容错机制，比如重试机制、熔断机制、降级处理和限流保护等，确保在部分实例不可用时，服务仍然可以正常运行。</p><p></p><p>最后是动态扩容与缩容的平滑处理。扩容时，新服务实例上线后，注册中心会自动注册并同步给消费者，负载均衡组件会逐步增加新实例的权重，实现流量的平滑过渡。缩容时，在下线实例之前，会逐步减少它的权重，等它处理完已有请求后再注销，避免流量损失。</p><p></p><p>总而言之，在微服务架构中，服务实例的动态变化是不可避免的，而应对这些变化的关键就在于服务注册中心和服务消费者的协同配合。</p>`,
					Guidance:  `心跳检测；健康检查；负载均衡；一致性哈希；重试；熔断；限流；降级；扩容；缩容；平滑处理；`,
					Shorthand: `注册中心实时监测，客户端做好容错，扩容要平滑；`,
				},
				Intermediate: web.AnswerElement{
					Id:        5113,
					Guidance:  "客户端容错；failover",
					Highlight: "客户端处理注册信息异常的容错机制；",
					Content:   `<p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">在服务实例节点动态变化的时候，最经常遇到的问题就是注册中心不能及时地将最新的状态同步给客户端，因此客户端容错就非常重要。</span></p><p></p><p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">而容错的做法其实也不难。举个例子来说，客户端在发现调用不通服务端的时候，可以考虑换一个节点重试。在这种最简单的做法之上，还可以考虑引入一些高级的做法，例如说当一个节点频繁的调用不通的时候，客户端可以考虑将该节点标记为不可用。而后客户端尝试向服务端发送心跳，如果心跳恢复了，则认为服务端节点已经恢复了，可以继续发送请求。</span></p><p></p><p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">我在 gRPC 里面就使用过类似的策略，有效提高了我们系统的可用性和稳定性。</span></p><p></p>`,
				},
				Advanced: web.AnswerElement{
					Id:        5114,
					Content:   `<p>而在大规模分布式系统下，如果服务节点动态变化频繁，那么会给注册中心带来庞大的压力。</p><p></p><p>一方面如果变化是服务节点主动上线下线引起的，那么它们就会触发写操作，更新注册中心中注册的信息。</p><p></p><p>另外一方面来说，如果注册中心的设计是实时同步，那么每一次变动注册中心都要通知客户端，这会导致注册中心和客户端之间频繁通信。</p><p></p><p>所以，现在部署大规模分布式微服务架构的时候，通常都是在 CAP 中选择 AP 模型来保证注册中心的高可用，同时确保注册中心能够撑住频繁的节点变化。</p>`,
					Guidance:  "CAP；",
					Highlight: "节点变化在大规模集群下对注册中心的影响",
					Shorthand: "注册中心选AP不选CP；",
				},
				Interactive: web.Interactive{
					CollectCnt: 1045,
					LikeCnt:    1044,
					ViewCnt:    1043,
					Collected:  true,
				},
				ExamineResult: 2,
				Permitted:     true,
			},
		},
		{
			name: "未命中缓存，刷新缓存",
			req: web.Qid{
				Qid: 22,
			},
			before: func(req *http.Request) {
				err := s.db.Create(&dao.PublishQuestion{
					Id:  22,
					Uid: uid,
					Labels: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val:   []string{"MySQL"},
					},
					BizId:   32,
					Biz:     "baguwen",
					Status:  domain.PublishedStatus.ToUint8(),
					Title:   "缓存测试问题标题",
					Content: "缓存测试问题内容",
					Utime:   1739678267424,
					Ctime:   1739678267424,
				}).Error
				require.NoError(s.T(), err)
				analysis := s.buildDAOAnswerEle(22, 1, dao.AnswerElementTypeAnalysis)
				analysis.Id = 101
				basic := s.buildDAOAnswerEle(22, 2, dao.AnswerElementTypeBasic)
				basic.Id = 102
				intermedia := s.buildDAOAnswerEle(22, 3, dao.AnswerElementTypeIntermedia)
				intermedia.Id = 103
				advanced := s.buildDAOAnswerEle(22, 4, dao.AnswerElementTypeAdvanced)
				advanced.Id = 104

				eles := []dao.PublishAnswerElement{
					dao.PublishAnswerElement(analysis),
					dao.PublishAnswerElement(basic),
					dao.PublishAnswerElement(advanced),
					dao.PublishAnswerElement(intermedia),
				}
				err = s.db.WithContext(context.Background()).Create(&eles).Error
				require.NoError(s.T(), err)

			},
			after: func() {
				analysis := s.buildDomainAnswerEle(1, 101)
				basic := s.buildDomainAnswerEle(2, 102)
				intermedia := s.buildDomainAnswerEle(3, 103)
				advanced := s.buildDomainAnswerEle(4, 104)

				// 校验缓存中有没有写入数据
				s.cacheAssertQuestion(domain.Question{
					Id:      22,
					Uid:     uid,
					Labels:  []string{"MySQL"},
					BizId:   32,
					Biz:     "baguwen",
					Status:  domain.PublishedStatus,
					Title:   "缓存测试问题标题",
					Content: "缓存测试问题内容",
					Answer: domain.Answer{
						Analysis:     analysis,
						Basic:        basic,
						Intermediate: intermedia,
						Advanced:     advanced,
					},
				})
			},
			wantData: web.Question{
				Id:      22,
				Labels:  []string{"MySQL"},
				BizId:   32,
				Biz:     "baguwen",
				Status:  domain.PublishedStatus.ToUint8(),
				Title:   "缓存测试问题标题",
				Content: "缓存测试问题内容",
				Utime:   1739678267424,
				Analysis: web.AnswerElement{
					Id:        101,
					Content:   fmt.Sprintf("这是解析 %d", 1),
					Keywords:  fmt.Sprintf("关键字 %d", 1),
					Shorthand: fmt.Sprintf("快速记忆法 %d", 1),
					Highlight: fmt.Sprintf("亮点 %d", 1),
					Guidance:  fmt.Sprintf("引导点 %d", 1),
				},
				Basic: web.AnswerElement{
					Id:        102,
					Content:   fmt.Sprintf("这是解析 %d", 2),
					Keywords:  fmt.Sprintf("关键字 %d", 2),
					Shorthand: fmt.Sprintf("快速记忆法 %d", 2),
					Highlight: fmt.Sprintf("亮点 %d", 2),
					Guidance:  fmt.Sprintf("引导点 %d", 2),
				},
				Intermediate: web.AnswerElement{
					Id:        103,
					Content:   fmt.Sprintf("这是解析 %d", 3),
					Keywords:  fmt.Sprintf("关键字 %d", 3),
					Shorthand: fmt.Sprintf("快速记忆法 %d", 3),
					Highlight: fmt.Sprintf("亮点 %d", 3),
					Guidance:  fmt.Sprintf("引导点 %d", 3),
				},
				Advanced: web.AnswerElement{
					Id:        104,
					Content:   fmt.Sprintf("这是解析 %d", 4),
					Keywords:  fmt.Sprintf("关键字 %d", 4),
					Shorthand: fmt.Sprintf("快速记忆法 %d", 4),
					Highlight: fmt.Sprintf("亮点 %d", 4),
					Guidance:  fmt.Sprintf("引导点 %d", 4),
				},
				Interactive: web.Interactive{
					CollectCnt: 25,
					LikeCnt:    24,
					ViewCnt:    23,
					Collected:  true,
				},
				ExamineResult: 0,
				Permitted:     true,
			},
		},
		{
			name: "命中缓存,直接返回",
			req: web.Qid{
				Qid: 23,
			},
			before: func(req *http.Request) {
				analysis := s.buildDomainAnswerEle(1, 105)
				basic := s.buildDomainAnswerEle(2, 106)
				intermedia := s.buildDomainAnswerEle(3, 107)
				advanced := s.buildDomainAnswerEle(4, 108)
				que := domain.Question{
					Id:      23,
					Uid:     uid,
					Labels:  []string{"MySQL"},
					BizId:   32,
					Biz:     "baguwen",
					Status:  domain.PublishedStatus,
					Title:   "缓存测试问题标题",
					Content: "缓存测试问题内容",
					Utime:   time.UnixMilli(1739678267424),
					Answer: domain.Answer{
						Analysis:     analysis,
						Basic:        basic,
						Intermediate: intermedia,
						Advanced:     advanced,
					},
				}
				queByte, err := json.Marshal(que)
				require.NoError(s.T(), err)
				err = s.rdb.Set(context.Background(), "question:publish:23", string(queByte), 24*time.Hour)
				require.NoError(s.T(), err)

			},
			after: func() {
				analysis := s.buildDomainAnswerEle(1, 105)
				basic := s.buildDomainAnswerEle(2, 106)
				intermedia := s.buildDomainAnswerEle(3, 107)
				advanced := s.buildDomainAnswerEle(4, 108)

				// 校验缓存中有没有写入数据
				s.cacheAssertQuestion(domain.Question{
					Id:      23,
					Uid:     uid,
					Labels:  []string{"MySQL"},
					BizId:   32,
					Biz:     "baguwen",
					Status:  domain.PublishedStatus,
					Title:   "缓存测试问题标题",
					Content: "缓存测试问题内容",
					Answer: domain.Answer{
						Analysis:     analysis,
						Basic:        basic,
						Intermediate: intermedia,
						Advanced:     advanced,
					},
				})
			},
			wantData: web.Question{
				Id:      23,
				Labels:  []string{"MySQL"},
				BizId:   32,
				Biz:     "baguwen",
				Status:  domain.PublishedStatus.ToUint8(),
				Title:   "缓存测试问题标题",
				Content: "缓存测试问题内容",
				Utime:   1739678267424,
				Analysis: web.AnswerElement{
					Id:        105,
					Content:   fmt.Sprintf("这是解析 %d", 1),
					Keywords:  fmt.Sprintf("关键字 %d", 1),
					Shorthand: fmt.Sprintf("快速记忆法 %d", 1),
					Highlight: fmt.Sprintf("亮点 %d", 1),
					Guidance:  fmt.Sprintf("引导点 %d", 1),
				},
				Basic: web.AnswerElement{
					Id:        106,
					Content:   fmt.Sprintf("这是解析 %d", 2),
					Keywords:  fmt.Sprintf("关键字 %d", 2),
					Shorthand: fmt.Sprintf("快速记忆法 %d", 2),
					Highlight: fmt.Sprintf("亮点 %d", 2),
					Guidance:  fmt.Sprintf("引导点 %d", 2),
				},
				Intermediate: web.AnswerElement{
					Id:        107,
					Content:   fmt.Sprintf("这是解析 %d", 3),
					Keywords:  fmt.Sprintf("关键字 %d", 3),
					Shorthand: fmt.Sprintf("快速记忆法 %d", 3),
					Highlight: fmt.Sprintf("亮点 %d", 3),
					Guidance:  fmt.Sprintf("引导点 %d", 3),
				},
				Advanced: web.AnswerElement{
					Id:        108,
					Content:   fmt.Sprintf("这是解析 %d", 4),
					Keywords:  fmt.Sprintf("关键字 %d", 4),
					Shorthand: fmt.Sprintf("快速记忆法 %d", 4),
					Highlight: fmt.Sprintf("亮点 %d", 4),
					Guidance:  fmt.Sprintf("引导点 %d", 4),
				},
				Interactive: web.Interactive{
					CollectCnt: 26,
					LikeCnt:    25,
					ViewCnt:    24,
					Liked:      true,
				},
				Permitted: true,
			},
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/question/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			tc.before(req)
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Question]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, 200, recorder.Code)
			data := recorder.MustScan().Data
			assert.Equal(t, tc.wantData, data)
			tc.after()
		})
	}
}

func (s *HandlerTestSuite) initData() {
	t := s.T()
	res := dao.QuestionResult{
		Id:     1,
		Uid:    uid,
		Qid:    1041,
		Result: domain.ResultIntermediate.ToUint8(),
	}
	que := dao.PublishQuestion{
		Id:      1041,
		Uid:     uid,
		BizId:   0,
		Biz:     "baguwen",
		Status:  domain.PublishedStatus.ToUint8(),
		Ctime:   123,
		Utime:   321,
		Title:   `在微服务架构中，如何处理服务实例的动态变化（如上线、下线、故障）？`,
		Content: `<p>略难的题，一般只会出现在社招中。</p><p></p><p>其实这种问法会让你觉得摸不着头脑，但是如果你把问题换成如果服务实例动态变化了，注册中心和客户端会怎样，就清晰多了。要在这个问题之下刷亮点，赢得竞争优势，你可以讨论客户端容错策略，以及高并发场景下服务实例频繁变化会给注册中心带来庞大的压力这两个点。</p>`,
	}
	analysis := dao.PublishAnswerElement{
		Id:      6110,
		Qid:     1041,
		Type:    dao.AnswerElementTypeAnalysis,
		Content: `<p>前置知识：</p><ul><li><a href="https://wsn.com/question/detail" rel="noopener noreferrer" target="_blank">你知道注册中心吗？</a></li></ul><p></p><p>在服务注册与发现中，服务健康检查是确保服务实例可用性的重要机制。通过健康检查，注册中心可以动态感知服务实例的状态变化（如健康、故障、下线等），从而保障消费者调用的服务始终可用。常见的健康检查方式主要有以下两类：</p><p></p><ol><li>主动健康检查：主动健康检查由注册中心或消费者主动发起探测请求，定期检测服务实例的健康状态。常见实现方式包括：<ul><li>HTTP 检查：注册中心向服务实例的健康检查端点（如 /health）发送 HTTP 请求，根据返回状态码（如 2xx）判断健康状态。<ul><li>优点：简单易用，适合 HTTP 服务。</li><li>缺点：只能检测服务的基本可达性，无法深入检测内部状态。</li><li>示例：Spring Boot Actuator 提供了 /actuator/health 端点，Nacos 和 Consul 支持通过 HTTP 检查服务健康。</li></ul></li><li>TCP 检查：注册中心尝试连接服务实例的指定端口，判断端口是否可用。<ul><li>优点：适合非 HTTP 服务（如数据库、消息队列）。</li><li>缺点：仅能检测端口连通性，无法反映业务逻辑状态。</li><li>示例：Consul 支持通过 TCP 检查服务端口。</li></ul></li><li>gRPC 检查：注册中心调用服务实例的 gRPC 健康检查接口（如 grpc.health.v1.Health/Check），判断服务是否健康。<ul><li>优点：适用于 gRPC 服务，通信高效。</li><li>缺点：需要服务实例实现 gRPC 健康检查接口。</li><li>示例：gRPC 官方提供了健康检查协议，适用于 gRPC 服务。</li></ul></li><li>自定义脚本检查：注册中心通过运行自定义脚本或命令检测服务状态。<ul><li>优点：灵活性高，可根据业务需求定制。</li><li>缺点：实现复杂，可能增加系统开销。</li><li>示例：Consul 支持通过 Shell 脚本实现自定义健康检查。</li></ul></li></ul></li><li>被动健康检查：被动健康检查通过监控服务实例的运行状态或调用结果，间接判断健康状况。常见实现方式包括：<ul><li>心跳检测：服务实例定期向注册中心发送心跳信号。如果在规定时间内未收到心跳，则认为实例不可用。<ul><li>优点：实现简单，适合大规模服务实例监控。</li><li>缺点：无法检测服务内部的业务逻辑状态。</li><li>示例：Eureka 和 Nacos 使用心跳机制维持服务健康状态。</li></ul></li><li>请求失败率监控：注册中心或消费者监控服务实例的请求失败率（如超时、错误响应等），当失败率超过阈值时，将实例标记为不可用。<ul><li>优点：能反映服务的实际运行状态。</li><li>缺点：需要额外的监控逻辑，可能存在延迟。</li><li>示例：Hystrix 和 Sentinel 可基于失败率隔离故障实例。</li></ul></li><li>日志监控：通过分析服务实例的运行日志，检测是否存在异常（如错误日志、超时日志）。<ul><li>优点：能深入了解服务运行状态。</li><li>缺点：实现复杂，实时性较差。</li><li>示例：使用 ELK（Elasticsearch、Logstash、Kibana）分析服务日志。</li></ul></li></ul></li></ol><p></p><p>在实际场景中，单一健康检查方式往往不足以全面反映服务状态，因此通常结合多种方式使用，并通过优化策略提升效率和准确性。例如：</p><ul><li>组合检查：<ul><li>主动 + 被动检查：通过 HTTP 检查服务的基本可达性，同时结合心跳检测判断服务是否仍然活跃。</li><li>多级检查：先通过 TCP 检查端口连通性，再通过 HTTP 检查服务业务逻辑状态。</li></ul></li><li>优化策略：<ul><li>调整检查频率：根据服务的重要性和负载情况，合理设置检查频率，避免过于频繁导致性能开销。</li><li>健康状态缓存：对健康检查结果进行短时间缓存，减少重复检查的开销。</li><li>多次失败判定：避免因短暂网络波动或服务抖动导致误判，可设置连续多次失败后才标记为不可用。</li><li>分布式健康检查：在大规模分布式系统中，将健康检查任务分散到多个节点，降低注册中心的压力。</li></ul></li></ul><p>在复杂场景中，还可以基于以下方式提升健康检查的深度和智能化：</p><ul><li><ul><li>依赖检查：检测服务依赖的资源（如数据库、缓存）是否正常。</li><li>业务指标检查：通过关键业务指标（如订单处理速度）判断服务健康状态。</li><li>AI大模型预测：利用AI大模型分析历史数据，提前预测潜在故障。</li></ul></li></ul><p></p><p>服务健康检查是服务注册与发现的关键环节，常见方式包括主动健康检查（如 HTTP、TCP、gRPC、自定义脚本）和被动健康检查（如心跳检测、失败率监控、日志分析）。主动检查适合检测服务的基本可达性，被动检查更能反映服务的实际运行状态。在实际应用中，通常结合多种方式，并通过优化策略提升健康检查的效率和准确性，从而保障微服务架构的稳定性和可用性。</p>`,
	}
	basic := dao.PublishAnswerElement{
		Id:        6111,
		Qid:       1041,
		Type:      dao.AnswerElementTypeBasic,
		Content:   `<p>在微服务架构中，服务实例的动态变化很常见，比如服务实例因为扩容、缩容或者故障而上下线。为了保障系统的稳定性和高可用性，我们需要一套完善的机制来处理这些变化，主要从以下几个方面入手：</p><p></p><p>首先是服务注册中心的动态管理。注册中心是处理服务实例动态变化的核心，它主要通过心跳检测和实例状态同步来实现。注册中心会定期检测实例的健康状态，如果超时未响应，就会移除实例。注册中心要把实例状态的变化实时同步给消费者。同步方式有推送和拉取两种，在大规模分布式系统场景下，可以采用增量同步、注册中心分区或分层架构等优化策略。</p><p></p><p>其次是服务消费者侧的容错策略。服务消费者这边需要一些容错策略来应对服务实例的动态变化。比如：服务消费者可以通过缓存和动态更新来应对实例的变化。还可以采用负载均衡策略，比如轮询、加权轮询、一致性哈希等，来分发流量。以及一些容错机制，比如重试机制、熔断机制、降级处理和限流保护等，确保在部分实例不可用时，服务仍然可以正常运行。</p><p></p><p>最后是动态扩容与缩容的平滑处理。扩容时，新服务实例上线后，注册中心会自动注册并同步给消费者，负载均衡组件会逐步增加新实例的权重，实现流量的平滑过渡。缩容时，在下线实例之前，会逐步减少它的权重，等它处理完已有请求后再注销，避免流量损失。</p><p></p><p>总而言之，在微服务架构中，服务实例的动态变化是不可避免的，而应对这些变化的关键就在于服务注册中心和服务消费者的协同配合。</p>`,
		Guidance:  `心跳检测；健康检查；负载均衡；一致性哈希；重试；熔断；限流；降级；扩容；缩容；平滑处理；`,
		Shorthand: `注册中心实时监测，客户端做好容错，扩容要平滑；`,
	}
	intermediate := dao.PublishAnswerElement{
		Id:        6112,
		Qid:       1041,
		Type:      dao.AnswerElementTypeIntermedia,
		Guidance:  "客户端容错；failover",
		Highlight: "客户端处理注册信息异常的容错机制；",
		Content:   `<p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">在服务实例节点动态变化的时候，最经常遇到的问题就是注册中心不能及时地将最新的状态同步给客户端，因此客户端容错就非常重要。</span></p><p></p><p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">而容错的做法其实也不难。举个例子来说，客户端在发现调用不通服务端的时候，可以考虑换一个节点重试。在这种最简单的做法之上，还可以考虑引入一些高级的做法，例如说当一个节点频繁的调用不通的时候，客户端可以考虑将该节点标记为不可用。而后客户端尝试向服务端发送心跳，如果心跳恢复了，则认为服务端节点已经恢复了，可以继续发送请求。</span></p><p></p><p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">我在 gRPC 里面就使用过类似的策略，有效提高了我们系统的可用性和稳定性。</span></p><p></p>`,
	}
	advanced := dao.PublishAnswerElement{
		Id:        6113,
		Qid:       1041,
		Type:      dao.AnswerElementTypeAdvanced,
		Content:   `<p>而在大规模分布式系统下，如果服务节点动态变化频繁，那么会给注册中心带来庞大的压力。</p><p></p><p>一方面如果变化是服务节点主动上线下线引起的，那么它们就会触发写操作，更新注册中心中注册的信息。</p><p></p><p>另外一方面来说，如果注册中心的设计是实时同步，那么每一次变动注册中心都要通知客户端，这会导致注册中心和客户端之间频繁通信。</p><p></p><p>所以，现在部署大规模分布式微服务架构的时候，通常都是在 CAP 中选择 AP 模型来保证注册中心的高可用，同时确保注册中心能够撑住频繁的节点变化。</p>`,
		Guidance:  "CAP；",
		Highlight: "节点变化在大规模集群下对注册中心的影响",
		Shorthand: "注册中心选AP不选CP；",
	}

	prores := dao.QuestionResult{
		Id:     2,
		Uid:    uid,
		Qid:    1042,
		Result: domain.ResultIntermediate.ToUint8(),
	}
	proQue := dao.PublishQuestion{
		Id:      1042,
		Uid:     uid,
		BizId:   32,
		Biz:     "project",
		Status:  domain.PublishedStatus.ToUint8(),
		Ctime:   123,
		Utime:   321,
		Title:   `在微服务架构中，如何处理服务实例的动态变化（如上线、下线、故障）？`,
		Content: `<p>略难的题，一般只会出现在社招中。</p><p></p><p>其实这种问法会让你觉得摸不着头脑，但是如果你把问题换成如果服务实例动态变化了，注册中心和客户端会怎样，就清晰多了。要在这个问题之下刷亮点，赢得竞争优势，你可以讨论客户端容错策略，以及高并发场景下服务实例频繁变化会给注册中心带来庞大的压力这两个点。</p>`,
	}
	proAnalysis := dao.PublishAnswerElement{
		Id:      5110,
		Qid:     1042,
		Type:    dao.AnswerElementTypeAnalysis,
		Content: `<p>前置知识：</p><ul><li><a href="https://wsn.com/question/detail" rel="noopener noreferrer" target="_blank">你知道注册中心吗？</a></li></ul><p></p><p>在服务注册与发现中，服务健康检查是确保服务实例可用性的重要机制。通过健康检查，注册中心可以动态感知服务实例的状态变化（如健康、故障、下线等），从而保障消费者调用的服务始终可用。常见的健康检查方式主要有以下两类：</p><p></p><ol><li>主动健康检查：主动健康检查由注册中心或消费者主动发起探测请求，定期检测服务实例的健康状态。常见实现方式包括：<ul><li>HTTP 检查：注册中心向服务实例的健康检查端点（如 /health）发送 HTTP 请求，根据返回状态码（如 2xx）判断健康状态。<ul><li>优点：简单易用，适合 HTTP 服务。</li><li>缺点：只能检测服务的基本可达性，无法深入检测内部状态。</li><li>示例：Spring Boot Actuator 提供了 /actuator/health 端点，Nacos 和 Consul 支持通过 HTTP 检查服务健康。</li></ul></li><li>TCP 检查：注册中心尝试连接服务实例的指定端口，判断端口是否可用。<ul><li>优点：适合非 HTTP 服务（如数据库、消息队列）。</li><li>缺点：仅能检测端口连通性，无法反映业务逻辑状态。</li><li>示例：Consul 支持通过 TCP 检查服务端口。</li></ul></li><li>gRPC 检查：注册中心调用服务实例的 gRPC 健康检查接口（如 grpc.health.v1.Health/Check），判断服务是否健康。<ul><li>优点：适用于 gRPC 服务，通信高效。</li><li>缺点：需要服务实例实现 gRPC 健康检查接口。</li><li>示例：gRPC 官方提供了健康检查协议，适用于 gRPC 服务。</li></ul></li><li>自定义脚本检查：注册中心通过运行自定义脚本或命令检测服务状态。<ul><li>优点：灵活性高，可根据业务需求定制。</li><li>缺点：实现复杂，可能增加系统开销。</li><li>示例：Consul 支持通过 Shell 脚本实现自定义健康检查。</li></ul></li></ul></li><li>被动健康检查：被动健康检查通过监控服务实例的运行状态或调用结果，间接判断健康状况。常见实现方式包括：<ul><li>心跳检测：服务实例定期向注册中心发送心跳信号。如果在规定时间内未收到心跳，则认为实例不可用。<ul><li>优点：实现简单，适合大规模服务实例监控。</li><li>缺点：无法检测服务内部的业务逻辑状态。</li><li>示例：Eureka 和 Nacos 使用心跳机制维持服务健康状态。</li></ul></li><li>请求失败率监控：注册中心或消费者监控服务实例的请求失败率（如超时、错误响应等），当失败率超过阈值时，将实例标记为不可用。<ul><li>优点：能反映服务的实际运行状态。</li><li>缺点：需要额外的监控逻辑，可能存在延迟。</li><li>示例：Hystrix 和 Sentinel 可基于失败率隔离故障实例。</li></ul></li><li>日志监控：通过分析服务实例的运行日志，检测是否存在异常（如错误日志、超时日志）。<ul><li>优点：能深入了解服务运行状态。</li><li>缺点：实现复杂，实时性较差。</li><li>示例：使用 ELK（Elasticsearch、Logstash、Kibana）分析服务日志。</li></ul></li></ul></li></ol><p></p><p>在实际场景中，单一健康检查方式往往不足以全面反映服务状态，因此通常结合多种方式使用，并通过优化策略提升效率和准确性。例如：</p><ul><li>组合检查：<ul><li>主动 + 被动检查：通过 HTTP 检查服务的基本可达性，同时结合心跳检测判断服务是否仍然活跃。</li><li>多级检查：先通过 TCP 检查端口连通性，再通过 HTTP 检查服务业务逻辑状态。</li></ul></li><li>优化策略：<ul><li>调整检查频率：根据服务的重要性和负载情况，合理设置检查频率，避免过于频繁导致性能开销。</li><li>健康状态缓存：对健康检查结果进行短时间缓存，减少重复检查的开销。</li><li>多次失败判定：避免因短暂网络波动或服务抖动导致误判，可设置连续多次失败后才标记为不可用。</li><li>分布式健康检查：在大规模分布式系统中，将健康检查任务分散到多个节点，降低注册中心的压力。</li></ul></li></ul><p>在复杂场景中，还可以基于以下方式提升健康检查的深度和智能化：</p><ul><li><ul><li>依赖检查：检测服务依赖的资源（如数据库、缓存）是否正常。</li><li>业务指标检查：通过关键业务指标（如订单处理速度）判断服务健康状态。</li><li>AI大模型预测：利用AI大模型分析历史数据，提前预测潜在故障。</li></ul></li></ul><p></p><p>服务健康检查是服务注册与发现的关键环节，常见方式包括主动健康检查（如 HTTP、TCP、gRPC、自定义脚本）和被动健康检查（如心跳检测、失败率监控、日志分析）。主动检查适合检测服务的基本可达性，被动检查更能反映服务的实际运行状态。在实际应用中，通常结合多种方式，并通过优化策略提升健康检查的效率和准确性，从而保障微服务架构的稳定性和可用性。</p>`,
	}
	proBasic := dao.PublishAnswerElement{
		Id:        5111,
		Qid:       1042,
		Type:      dao.AnswerElementTypeBasic,
		Content:   `<p>在微服务架构中，服务实例的动态变化很常见，比如服务实例因为扩容、缩容或者故障而上下线。为了保障系统的稳定性和高可用性，我们需要一套完善的机制来处理这些变化，主要从以下几个方面入手：</p><p></p><p>首先是服务注册中心的动态管理。注册中心是处理服务实例动态变化的核心，它主要通过心跳检测和实例状态同步来实现。注册中心会定期检测实例的健康状态，如果超时未响应，就会移除实例。注册中心要把实例状态的变化实时同步给消费者。同步方式有推送和拉取两种，在大规模分布式系统场景下，可以采用增量同步、注册中心分区或分层架构等优化策略。</p><p></p><p>其次是服务消费者侧的容错策略。服务消费者这边需要一些容错策略来应对服务实例的动态变化。比如：服务消费者可以通过缓存和动态更新来应对实例的变化。还可以采用负载均衡策略，比如轮询、加权轮询、一致性哈希等，来分发流量。以及一些容错机制，比如重试机制、熔断机制、降级处理和限流保护等，确保在部分实例不可用时，服务仍然可以正常运行。</p><p></p><p>最后是动态扩容与缩容的平滑处理。扩容时，新服务实例上线后，注册中心会自动注册并同步给消费者，负载均衡组件会逐步增加新实例的权重，实现流量的平滑过渡。缩容时，在下线实例之前，会逐步减少它的权重，等它处理完已有请求后再注销，避免流量损失。</p><p></p><p>总而言之，在微服务架构中，服务实例的动态变化是不可避免的，而应对这些变化的关键就在于服务注册中心和服务消费者的协同配合。</p>`,
		Guidance:  `心跳检测；健康检查；负载均衡；一致性哈希；重试；熔断；限流；降级；扩容；缩容；平滑处理；`,
		Shorthand: `注册中心实时监测，客户端做好容错，扩容要平滑；`,
	}
	proIntermediate := dao.PublishAnswerElement{
		Id:        5113,
		Qid:       1042,
		Type:      dao.AnswerElementTypeIntermedia,
		Guidance:  "客户端容错；failover",
		Highlight: "客户端处理注册信息异常的容错机制；",
		Content:   `<p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">在服务实例节点动态变化的时候，最经常遇到的问题就是注册中心不能及时地将最新的状态同步给客户端，因此客户端容错就非常重要。</span></p><p></p><p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">而容错的做法其实也不难。举个例子来说，客户端在发现调用不通服务端的时候，可以考虑换一个节点重试。在这种最简单的做法之上，还可以考虑引入一些高级的做法，例如说当一个节点频繁的调用不通的时候，客户端可以考虑将该节点标记为不可用。而后客户端尝试向服务端发送心跳，如果心跳恢复了，则认为服务端节点已经恢复了，可以继续发送请求。</span></p><p></p><p><span style="background-color: rgb(255, 255, 255); color: rgba(0, 0, 0, 0.88);">我在 gRPC 里面就使用过类似的策略，有效提高了我们系统的可用性和稳定性。</span></p><p></p>`,
	}
	proAdvanced := dao.PublishAnswerElement{
		Id:        5114,
		Qid:       1042,
		Type:      dao.AnswerElementTypeAdvanced,
		Content:   `<p>而在大规模分布式系统下，如果服务节点动态变化频繁，那么会给注册中心带来庞大的压力。</p><p></p><p>一方面如果变化是服务节点主动上线下线引起的，那么它们就会触发写操作，更新注册中心中注册的信息。</p><p></p><p>另外一方面来说，如果注册中心的设计是实时同步，那么每一次变动注册中心都要通知客户端，这会导致注册中心和客户端之间频繁通信。</p><p></p><p>所以，现在部署大规模分布式微服务架构的时候，通常都是在 CAP 中选择 AP 模型来保证注册中心的高可用，同时确保注册中心能够撑住频繁的节点变化。</p>`,
		Guidance:  "CAP；",
		Highlight: "节点变化在大规模集群下对注册中心的影响",
		Shorthand: "注册中心选AP不选CP；",
	}

	err := s.db.WithContext(context.Background()).Create([]dao.QuestionResult{
		res,
		prores,
	}).Error
	require.NoError(t, err)
	err = s.db.WithContext(context.Background()).Create([]dao.PublishQuestion{
		que,
		proQue,
	}).Error
	require.NoError(t, err)
	err = s.db.WithContext(context.Background()).Create([]dao.PublishAnswerElement{
		analysis,
		basic,
		intermediate,
		advanced,
		proAnalysis,
		proBasic,
		proIntermediate,
		proAdvanced,
	}).Error
	require.NoError(t, err)
}

// 校验缓存中的数据
func (s *HandlerTestSuite) cacheAssertQuestion(q domain.Question) {
	t := s.T()
	key := fmt.Sprintf("question:publish:%d", q.Id)
	val := s.rdb.Get(context.Background(), key)
	require.NoError(t, val.Err)

	var actual domain.Question
	err := json.Unmarshal([]byte(val.Val.(string)), &actual)
	require.NoError(t, err)

	// 处理时间字段
	require.True(t, actual.Utime.Unix() > 0)
	q.Utime = actual.Utime
	assert.Equal(t, q, actual)
	// 清理缓存
	_, err = s.rdb.Delete(context.Background(), key)
	require.NoError(t, err)
}

func (s *HandlerTestSuite) cacheAssertQuestionList(biz string, questions []domain.Question) {
	key := fmt.Sprintf("question:list:%s", biz)
	val := s.rdb.Get(context.Background(), key)
	require.NoError(s.T(), val.Err)

	qs := []domain.Question{}
	err := json.Unmarshal([]byte(val.Val.(string)), &qs)
	require.NoError(s.T(), err)
	require.Equal(s.T(), len(questions), len(qs))
	for idx, q := range qs {
		require.True(s.T(), q.Utime.UnixMilli() > 0)
		qs[idx].Utime = questions[idx].Utime
		qs[idx].Answer.Utime = questions[idx].Answer.Utime
	}
	assert.Equal(s.T(), questions, qs)
	_, err = s.rdb.Delete(context.Background(), key)
	require.NoError(s.T(), err)
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
