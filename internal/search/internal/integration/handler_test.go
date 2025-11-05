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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/cases"
	casemocks "github.com/ecodeclub/webook/internal/cases/mocks"
	"github.com/ecodeclub/webook/internal/pkg/middleware"
	"github.com/ecodeclub/webook/internal/search/internal/event"
	"github.com/ecodeclub/webook/internal/search/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/search/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/search/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type HandlerTestSuite struct {
	suite.Suite
	server   *egin.Component
	es       *elasticsearch.TypedClient
	producer mq.Producer
}

func (s *HandlerTestSuite) TearDownSuite() {
	// 创建范围查询，匹配 5000< id < 9000 的文档
	gt := types.Float64(5000)
	lt := types.Float64(9000)
	rangeQuery := types.NewNumberRangeQuery()
	rangeQuery.Gt = &gt
	rangeQuery.Lt = &lt

	query := &types.Query{
		Range: map[string]types.RangeQuery{
			"id": rangeQuery.RangeQueryCaster(),
		},
	}

	deleteReq := map[string]interface{}{
		"query": query,
	}
	deleteBytes, _ := json.Marshal(deleteReq)

	_, err := s.es.DeleteByQuery("pub_case_index").
		Raw(bytes.NewReader(deleteBytes)).
		Do(context.Background())
	require.NoError(s.T(), err)
	_, err = s.es.DeleteByQuery("pub_question_index").
		Raw(bytes.NewReader(deleteBytes)).
		Do(context.Background())
	require.NoError(s.T(), err)
	_, err = s.es.DeleteByQuery("skill_index").
		Raw(bytes.NewReader(deleteBytes)).
		Do(context.Background())
	require.NoError(s.T(), err)
	_, err = s.es.DeleteByQuery(dao.QuestionSetIndexName).
		Raw(bytes.NewReader(deleteBytes)).
		Do(context.Background())
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) SetupSuite() {
	ctrl := gomock.NewController(s.T())
	examSvc := casemocks.NewMockExamineService(ctrl)
	examSvc.EXPECT().GetResults(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, uid int64, ids []int64) (map[int64]cases.ExamineResult, error) {
		res := slice.Map(ids, func(idx int, src int64) cases.ExamineResult {
			return cases.ExamineResult{
				Cid:    src,
				Result: cases.ExamineResultEnum(src % 2),
			}
		})
		resMap := make(map[int64]cases.ExamineResult, len(res))
		for _, examRes := range res {
			resMap[examRes.Cid] = examRes
		}
		return resMap, nil
	}).AnyTimes()

	intrSvc := intrmocks.NewMockService(ctrl)
	intrSvc.EXPECT().GetByIds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context,
		biz string, uid int64, ids []int64) (map[int64]interactive.Interactive, error) {
		res := make(map[int64]interactive.Interactive, len(ids))
		for _, id := range ids {
			intr := s.mockInteractive(biz, id)
			res[id] = intr
		}
		return res, nil
	}).AnyTimes()

	handler, err := startup.InitHandler(&cases.Module{
		ExamineSvc: examSvc,
	}, &interactive.Module{
		Svc: intrSvc,
	})
	require.NoError(s.T(), err)
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
	handler.PrivateRoutes(server.Engine)
	server.Use(middleware.NewCheckMembershipMiddlewareBuilder(nil).Build())
	s.server = server
	s.es = testioc.InitES()
	testmq := testioc.InitMQ()
	p, err := testmq.Producer(event.SyncTopic)
	if err != nil {
		panic(err)
	}
	time.Sleep(1 * time.Second)
	s.producer = p
}

func (s *HandlerTestSuite) initSkills() {
	skills := []dao.Skill{
		{
			ID:     5001,
			Labels: []string{"programming", "golang"},
			Name:   "test_name",
			Desc:   "Learn Golang programming language",
			Utime:  1619708855,
		},
		{
			ID:     5002,
			Labels: []string{"programming", "test_label"},
			Name:   "",
			Desc:   "Learn Golang programming language",
			Utime:  1619708855,
		},
		{
			ID:     5003,
			Labels: []string{"programming"},
			Name:   "",
			Desc:   "test_desc",
			Utime:  1619708855,
		},
		{
			ID:     5004,
			Labels: []string{"programming"},
			Name:   "",
			Desc:   "",
			Basic: dao.SkillLevel{
				ID:        1,
				Desc:      "test_basic",
				Utime:     1619708855,
				Questions: []int64{1},
				Cases:     []int64{1},
			},
			Utime: 1619708855,
		},
		{
			ID:     5005,
			Labels: []string{"programming"},
			Name:   "",
			Desc:   "test_desc",
			Intermediate: dao.SkillLevel{
				ID:        2,
				Desc:      "test_intermediate",
				Utime:     1619708855,
				Questions: []int64{1},
				Cases:     []int64{1},
			},
			Utime: 1619708855,
		},
		{
			ID:     5006,
			Labels: []string{"programming"},
			Name:   "",
			Desc:   "",
			Advanced: dao.SkillLevel{
				ID:        2,
				Desc:      "test_advanced",
				Utime:     1619708855,
				Questions: []int64{1},
				Cases:     []int64{1},
			},
			Utime: 1619708855,
		},
	}
	for _, skill := range skills {
		by, err := json.Marshal(skill)
		require.NoError(s.T(), err)
		_, err = s.es.Index(dao.SkillIndexName).
			Id(strconv.FormatInt(skill.ID, 10)).
			Raw(bytes.NewReader(by)).
			Do(context.Background())
		require.NoError(s.T(), err)
	}
}

func (s *HandlerTestSuite) TestBizSearch() {
	testCases := []struct {
		name    string
		before  func(t *testing.T)
		after   func(t *testing.T, wantRes web.CSearchResp, actual web.CSearchResp)
		wantAns web.CSearchResp
		req     web.SearchReq
	}{
		{
			name: "搜索cases",
			before: func(t *testing.T) {
				s.initCases()
			},
			after: func(t *testing.T, wantRes web.CSearchResp, actual web.CSearchResp) {
				for idx := range actual.Cases {
					que := actual.Cases[idx]
					require.True(t, que.Utime > 0)
					actual.Cases[idx].Utime = 0
				}
				assert.ElementsMatch(t, wantRes.Cases, actual.Cases)
			},
			wantAns: web.CSearchResp{
				Cases: []web.CSearchRes{
					{
						Id:          10006,
						Title:       "test_title",
						Tags:        []string{"label1"},
						Description: "Elasticsearch内容",
						Result:      0,
						Interactive: web.Interactive{
							ViewCnt:    10007,
							LikeCnt:    10008,
							CollectCnt: 10009,
							Liked:      false,
							Collected:  true,
						},
					},
					{
						Id:          10005,
						Title:       "Elasticsearch标题",
						Tags:        []string{"test_label"},
						Description: "Elasticsearch内容",
						Result:      1,
						Interactive: web.Interactive{
							ViewCnt:    10006,
							LikeCnt:    10007,
							CollectCnt: 10008,
							Liked:      true,
							Collected:  false,
						},
					},
					{
						Id:          10002,
						Title:       "Elasticsearch标题",
						Tags:        []string{"label1"},
						Description: "Elasticsearch内容",
						Result:      0,
						Interactive: web.Interactive{
							ViewCnt:    10003,
							LikeCnt:    10004,
							CollectCnt: 10005,
							Liked:      false,
							Collected:  true,
						},
					},
					{
						Id:          10003,
						Title:       "Elasticsearch标题",
						Tags:        []string{"label1", "label2"},
						Description: "Elasticsearch内容",
						Result:      1,
						Interactive: web.Interactive{
							ViewCnt:    10004,
							LikeCnt:    10005,
							CollectCnt: 10006,
							Liked:      true,
							Collected:  false,
						},
					},
					{
						Id:          10001,
						Title:       "Elasticsearch标题",
						Tags:        []string{"label1", "label2"},
						Description: "<strong>test_content</strong>",
						Result:      1,
						Interactive: web.Interactive{
							ViewCnt:    10002,
							LikeCnt:    10003,
							CollectCnt: 10004,
							Liked:      true,
							Collected:  false,
						},
					},
					{
						Id:          10004,
						Title:       "Elasticsearch标题",
						Tags:        []string{"label1", "label2"},
						Description: "Elasticsearch内容",
						Result:      0,
						Interactive: web.Interactive{
							ViewCnt:    10005,
							LikeCnt:    10006,
							CollectCnt: 10007,
							Liked:      false,
							Collected:  true,
						},
					},
					{
						Id:          10007,
						Title:       "00000",
						Tags:        []string{"xxxx"},
						Description: "Elasticsearch内容",
						Result:      1,
						Interactive: web.Interactive{
							ViewCnt:    10008,
							LikeCnt:    10009,
							CollectCnt: 10010,
							Liked:      true,
							Collected:  false,
						},
					},
				},
			},
			req: web.SearchReq{
				Keywords: "biz:case:test_content test_keywords test_shorthands test_guidance test_title test_label kkkk",
				Offset:   0,
				Limit:    20,
			},
		},
		{
			name: "搜索questions",
			before: func(t *testing.T) {
				s.initQuestions()
			},
			after: func(t *testing.T, wantRes web.CSearchResp, actual web.CSearchResp) {
				for idx := range actual.Questions {
					actual.Questions[idx].Utime = 0
				}
				assert.ElementsMatch(t, wantRes.Questions, actual.Questions)
			},
			wantAns: web.CSearchResp{
				Questions: []web.CSearchRes{
					{
						Id:          10002,
						Title:       "test_title",
						Tags:        []string{"elasticsearch", "search"},
						Description: "I want to know how to use Elasticsearch for searching.",
						Interactive: web.Interactive{
							ViewCnt:    10003,
							LikeCnt:    10004,
							CollectCnt: 10005,
							Liked:      false,
							Collected:  true,
						},
					},
					{
						Id:          10001,
						Title:       "dasdsa",
						Tags:        []string{"test_label"},
						Description: "I want to know how to use Elasticsearch for searching.",
						Interactive: web.Interactive{
							ViewCnt:    10002,
							LikeCnt:    10003,
							CollectCnt: 10004,
							Liked:      true,
							Collected:  false,
						},
					},
					{
						Id:          10004,
						Title:       "Elasticsearch",
						Tags:        []string{"tElasticsearch"},
						Description: "描述：<strong>test_content</strong><br/>",
						Interactive: web.Interactive{
							ViewCnt:    10005,
							LikeCnt:    10006,
							CollectCnt: 10007,
							Liked:      false,
							Collected:  true,
						},
					},
					{
						Id:          10003,
						Title:       "How to use Elasticsearch?",
						Tags:        []string{"elasticsearch", "search"},
						Description: "题目分析：<strong>test_analysis_content</strong><br/>",
						Interactive: web.Interactive{
							ViewCnt:    10004,
							LikeCnt:    10005,
							CollectCnt: 10006,
							Liked:      true,
							Collected:  false,
						},
					},
					{
						Id:          10005,
						Title:       "How to use Elasticsearch?",
						Tags:        []string{"elasticsearch", "search"},
						Description: "I want to know how to use Elasticsearch for searching.",
						Interactive: web.Interactive{
							ViewCnt:    10006,
							LikeCnt:    10007,
							CollectCnt: 10008,
							Liked:      true,
							Collected:  false,
						},
					},
					{
						Id:          10006,
						Title:       "How to use Elasticsearch?",
						Tags:        []string{"elasticsearch", "search"},
						Description: "I want to know how to use Elasticsearch for searching.",
						Interactive: web.Interactive{
							ViewCnt:    10007,
							LikeCnt:    10008,
							CollectCnt: 10009,
							Liked:      false,
							Collected:  true,
						},
					},
					{
						Id:          10007,
						Title:       "How to use Elasticsearch?",
						Tags:        []string{"elasticsearch", "search"},
						Description: "I want to know how to use Elasticsearch for searching.",
						Interactive: web.Interactive{
							ViewCnt:    10008,
							LikeCnt:    10009,
							CollectCnt: 10010,
							Liked:      true,
							Collected:  false,
						},
					},
					{
						Id:          10008,
						Title:       "How to use Elasticsearch?",
						Tags:        []string{"elasticsearch", "search"},
						Description: "基础回答：<strong>test_basic_content</strong><br/>",
						Interactive: web.Interactive{
							ViewCnt:    10009,
							LikeCnt:    10010,
							CollectCnt: 10011,
							Liked:      false,
							Collected:  true,
						},
					},
					{
						Id:          10009,
						Title:       "How to use Elasticsearch?",
						Tags:        []string{"elasticsearch", "search"},
						Description: "I want to know how to use Elasticsearch for searching.",
						Interactive: web.Interactive{
							ViewCnt:    10010,
							LikeCnt:    10011,
							CollectCnt: 10012,
							Liked:      true,
							Collected:  false,
						},
					},
					{
						Id:          10010,
						Title:       "How to use Elasticsearch?",
						Tags:        []string{"elasticsearch", "search"},
						Description: "I want to know how to use Elasticsearch for searching.",
						Interactive: web.Interactive{
							ViewCnt:    10011,
							LikeCnt:    10012,
							CollectCnt: 10013,
							Liked:      false,
							Collected:  true,
						},
					},
					{
						Id:          10011,
						Title:       "How to use Elasticsearch?",
						Tags:        []string{"elasticsearch", "search"},
						Description: "I want to know how to use Elasticsearch for searching.",
						Interactive: web.Interactive{
							ViewCnt:    10012,
							LikeCnt:    10013,
							CollectCnt: 10014,
							Liked:      true,
							Collected:  false,
						},
					},
					{
						Id:          10012,
						Title:       "How to use Elasticsearch?",
						Tags:        []string{"elasticsearch", "search"},
						Description: "中级回答：<strong>test_intermediate_content</strong><br/>",
						Interactive: web.Interactive{
							ViewCnt:    10013,
							LikeCnt:    10014,
							CollectCnt: 10015,
							Liked:      false,
							Collected:  true,
						},
					},
					{
						Id:          10013,
						Title:       "How to use Elasticsearch?",
						Tags:        []string{"elasticsearch", "search"},
						Description: "I want to know how to use Elasticsearch for searching.",
						Interactive: web.Interactive{
							ViewCnt:    10014,
							LikeCnt:    10015,
							CollectCnt: 10016,
							Liked:      true,
							Collected:  false,
						},
					},
					{
						Id:    10014,
						Title: "How to use Elasticsearch?",

						Tags:        []string{"elasticsearch", "search"},
						Description: "I want to know how to use Elasticsearch for searching.",
						Interactive: web.Interactive{
							ViewCnt:    10015,
							LikeCnt:    10016,
							CollectCnt: 10017,
							Liked:      false,
							Collected:  true,
						},
					},
					{
						Id:    10015,
						Title: "How to use Elasticsearch?",

						Tags:        []string{"elasticsearch", "search"},
						Description: "I want to know how to use Elasticsearch for searching.",
						Interactive: web.Interactive{
							ViewCnt:    10016,
							LikeCnt:    10017,
							CollectCnt: 10018,
							Liked:      true,
							Collected:  false,
						},
					},
					{
						Id:    10016,
						Title: "How to use Elasticsearch?",

						Tags:        []string{"elasticsearch", "search"},
						Description: "I want to know how to use Elasticsearch for searching.",
						Interactive: web.Interactive{
							ViewCnt:    10017,
							LikeCnt:    10018,
							CollectCnt: 10019,
							Liked:      false,
							Collected:  true,
						},
					},
					{
						Id:    10017,
						Title: "How to use Elasticsearch?",

						Tags:        []string{"elasticsearch", "search"},
						Description: "I want to know how to use Elasticsearch for searching.",
						Interactive: web.Interactive{
							ViewCnt:    10018,
							LikeCnt:    10019,
							CollectCnt: 10020,
							Liked:      true,
							Collected:  false,
						},
					},
					{
						Id:    10018,
						Title: "How to use Elasticsearch?",

						Tags:        []string{"elasticsearch", "search"},
						Description: "I want to know how to use Elasticsearch for searching.",
						Interactive: web.Interactive{
							ViewCnt:    10019,
							LikeCnt:    10020,
							CollectCnt: 10021,
							Liked:      false,
							Collected:  true,
						},
					},
					{
						Id:    10019,
						Title: "How to use Elasticsearch?",

						Tags:        []string{"elasticsearch", "search"},
						Description: "高级回答：<strong>test_advanced_content</strong><br/>",
						Interactive: web.Interactive{
							ViewCnt:    10020,
							LikeCnt:    10021,
							CollectCnt: 10022,
							Liked:      true,
							Collected:  false,
						},
					},
				},
			},
			req: web.SearchReq{
				Keywords: "biz:question:test_content test_title test_label test_analysis_keywords test_analysis_shorthand test_analysis_highlight test_analysis_guidance test_analysis_content test_basic_keywords test_basic_shorthand test_basic_highlight test_basic_guidance test_basic_content  test_intermediate_keywords test_intermediate_shorthand test_intermediate_highlight test_intermediate_guidance test_intermediate_content test_advanced_keywords test_advanced_shorthand test_advanced_highlight test_advanced_guidance test_advanced_content",
				Offset:   0,
				Limit:    20,
			},
		},
		{
			name: "搜索skills",
			before: func(t *testing.T) {
				s.initSkills()
			},
			after: func(t *testing.T, wantRes web.CSearchResp, actual web.CSearchResp) {
				for idx := range actual.Skills {
					skill := actual.Skills[idx]
					require.True(t, skill.Utime > 0)
					actual.Skills[idx].Utime = 0
				}
				assert.ElementsMatch(t, wantRes.Skills, actual.Skills)
			},
			wantAns: web.CSearchResp{
				Skills: []web.CSearchRes{
					{
						Id:          5001,
						Title:       "test_name",
						Tags:        []string{"programming", "golang"},
						Description: "Learn Golang programming language",
					},
					{
						Id:          5002,
						Title:       "",
						Tags:        []string{"programming", "test_label"},
						Description: "Learn Golang programming language",
					},
					{
						Id:          5003,
						Title:       "",
						Tags:        []string{"programming"},
						Description: "描述：<strong>test_desc</strong><br/>",
					},
					{
						Id:          5004,
						Title:       "",
						Tags:        []string{"programming"},
						Description: "基础回答：<strong>test_basic</strong><br/>",
					},
					{
						Id:          5005,
						Title:       "",
						Tags:        []string{"programming"},
						Description: "描述：<strong>test_desc</strong><br/>中级回答：<strong>test_intermediate</strong><br/>",
					},
					{
						Id:          5006,
						Title:       "",
						Tags:        []string{"programming"},
						Description: "高级回答：<strong>test_advanced</strong><br/>",
					},
				},
			},
			req: web.SearchReq{
				Keywords: "biz:skill:test_name test_label test_desc test_advanced test_basic test_intermediate",
				Offset:   0,
				Limit:    20,
			},
		},
		{
			name: "搜索questionSets",
			before: func(t *testing.T) {
				s.initQuestionSets()
			},
			after: func(t *testing.T, wantRes web.CSearchResp, actual web.CSearchResp) {
				for idx := range actual.QuestionSet {
					que := actual.QuestionSet[idx]
					require.True(t, que.Utime > 0)
					actual.QuestionSet[idx].Utime = 0
				}
				assert.ElementsMatch(t, wantRes.QuestionSet, actual.QuestionSet)
			},
			wantAns: web.CSearchResp{
				QuestionSet: []web.CSearchRes{
					{
						Id:    5002,
						Title: "test_title",

						Description: "This is a test question set",
					},
					{
						Id:    5001,
						Title: "jjjkjk",

						Description: "<strong>test_desc</strong>",
					},
				},
			},
			req: web.SearchReq{
				Keywords: "biz:questionSet:test_title test_desc",
				Offset:   0,
				Limit:    20,
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			time.Sleep(3 * time.Second)
			req, err := http.NewRequest(http.MethodPost,
				"/search/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.CSearchResp]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, 200, recorder.Code)
			data := recorder.MustScan().Data
			tc.after(t, tc.wantAns, data)
		})
	}
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

func (s *HandlerTestSuite) initCases() {
	testcases := []dao.Case{
		{
			Id:         10001,
			Uid:        1,
			Biz:        "test",
			BizID:      10001,
			Labels:     []string{"label1", "label2"},
			Title:      "Elasticsearch标题",
			Content:    "test_content",
			GithubRepo: "Elasticsearch github代码库",
			GiteeRepo:  "Elasticsearch gitee代码库",
			Keywords:   "Elasticsearch关键词",
			Shorthand:  "Elasticsearch速记",
			Highlight:  "Elasticsearch亮点",
			Guidance:   "Elasticsearch引导",
			Status:     2,
			Ctime:      1619708855,
			Utime:      1619708855,
		},
		{
			Id:         10002,
			Uid:        1,
			BizID:      10002,
			Biz:        "test",
			Labels:     []string{"label1"},
			Title:      "Elasticsearch标题",
			Content:    "Elasticsearch内容",
			GithubRepo: "Elasticsearch github代码库",
			GiteeRepo:  "Elasticsearch gitee代码库",
			Keywords:   "test_keywords",
			Shorthand:  "Elasticsearch速记",
			Highlight:  "Elasticsearch亮点",
			Guidance:   "Elasticsearch引导",
			Status:     2,
			Ctime:      1619708855,
			Utime:      1619708855,
		},
		{
			Id:         10003,
			Uid:        1,
			BizID:      10003,
			Biz:        "test",
			Labels:     []string{"label1", "label2"},
			Title:      "Elasticsearch标题",
			Content:    "Elasticsearch内容",
			GithubRepo: "Elasticsearch github代码库",
			GiteeRepo:  "Elasticsearch gitee代码库",
			Keywords:   "Elasticsearch关键词",
			Shorthand:  "test_shorthands",
			Highlight:  "Elasticsearch亮点",
			Guidance:   "Elasticsearch引导",
			Status:     2,
			Ctime:      1619708855,
			Utime:      1619708855,
		},
		{
			Id:         10004,
			Uid:        1,
			BizID:      10004,
			Biz:        "test",
			Labels:     []string{"label1", "label2"},
			Title:      "Elasticsearch标题",
			Content:    "Elasticsearch内容",
			GithubRepo: "Elasticsearch github代码库",
			GiteeRepo:  "Elasticsearch gitee代码库",
			Keywords:   "Elasticsearch关键词",
			Shorthand:  "Elasticsearch速记",
			Highlight:  "Elasticsearch亮点",
			Guidance:   "test_guidance",
			Status:     2,
			Ctime:      1619708855,
			Utime:      1619708855,
		},
		{
			Id:         10005,
			Uid:        1,
			BizID:      10005,
			Biz:        "test",
			Labels:     []string{"test_label"},
			Title:      "Elasticsearch标题",
			Content:    "Elasticsearch内容",
			GithubRepo: "Elasticsearch github代码库",
			GiteeRepo:  "Elasticsearch gitee代码库",
			Keywords:   "Elasticsearch关键词",
			Shorthand:  "Elasticsearch速记",
			Highlight:  "Elasticsearch亮点",
			Guidance:   "Elasticsearch引导",
			Status:     2,
			Ctime:      1619708855,
			Utime:      1619708855,
		},
		{
			Id:         10006,
			Uid:        1,
			BizID:      10006,
			Biz:        "test",
			Labels:     []string{"label1"},
			Title:      "test_title",
			Content:    "Elasticsearch内容",
			GithubRepo: "Elasticsearch github代码库",
			GiteeRepo:  "Elasticsearch gitee代码库",
			Keywords:   "Elasticsearch关键词",
			Shorthand:  "Elasticsearch速记",
			Highlight:  "Elasticsearch亮点",
			Guidance:   "Elasticsearch引导",
			Status:     2,
			Ctime:      1619708855,
			Utime:      1619708855,
		},
		{
			Id:         10007,
			Uid:        1,
			BizID:      10007,
			Biz:        "kkkk",
			Labels:     []string{"xxxx"},
			Title:      "00000",
			Content:    "Elasticsearch内容",
			GithubRepo: "Elasticsearch github代码库",
			GiteeRepo:  "Elasticsearch gitee代码库",
			Keywords:   "Elasticsearch关键词",
			Shorthand:  "Elasticsearch速记",
			Highlight:  "Elasticsearch亮点",
			Guidance:   "Elasticsearch引导",
			Status:     2,
			Ctime:      1619708855,
			Utime:      1619708855,
		},
	}
	s.insertCase(testcases)
}

func (s *HandlerTestSuite) initQuestions() {
	questions := []dao.Question{
		{
			ID:      10002,
			UID:     101,
			Biz:     "test",
			BizID:   10002,
			Title:   "test_title",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Utime:   1619708855,
		},
		{
			ID:      10001,
			UID:     101,
			BizID:   10001,
			Biz:     "test",
			Title:   "dasdsa",
			Labels:  []string{"test_label"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Utime:   1619708855,
		},
		{
			ID:      10003,
			UID:     101,
			Biz:     "test",
			BizID:   10003,
			Title:   "How to use Elasticsearch?",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Answer: dao.Answer{
				Analysis: dao.AnswerElement{
					ID:        1,
					Content:   "test_analysis_content",
					Keywords:  "",
					Shorthand: "ES",
					Highlight: "distributed search and analytics engine",
					Guidance:  "Learn more about Elasticsearch documentation.",
				},
			},
			Utime: 1619708855,
		},
		{
			ID:      10004,
			UID:     101,
			Biz:     "test",
			BizID:   10004,
			Title:   "Elasticsearch",
			Labels:  []string{"tElasticsearch"},
			Content: "test_content",
			Status:  2,
			Utime:   1619708855,
		},
		{
			ID:      10005,
			UID:     101,
			Biz:     "test",
			BizID:   10005,
			Title:   "How to use Elasticsearch?",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Answer: dao.Answer{
				Analysis: dao.AnswerElement{
					ID:        1,
					Content:   "Elasticsearch is a distributed search and analytics engine.",
					Keywords:  "",
					Shorthand: "test_analysis_shorthand",
					Highlight: "distributed search and analytics engine",
					Guidance:  "Learn more about Elasticsearch documentation.",
				},
			},
			Utime: 1619708855,
		},
		{
			ID:      10006,
			UID:     101,
			Biz:     "test",
			BizID:   10006,
			Title:   "How to use Elasticsearch?",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Answer: dao.Answer{
				Analysis: dao.AnswerElement{
					ID:        1,
					Content:   "Elasticsearch is a distributed search and analytics engine.",
					Keywords:  "",
					Shorthand: "",
					Highlight: "test_analysis_highlight",
					Guidance:  "Learn more about Elasticsearch documentation.",
				},
			},
			Utime: 1619708855,
		},
		{
			ID:      10007,
			UID:     101,
			Biz:     "test",
			BizID:   10007,
			Title:   "How to use Elasticsearch?",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Answer: dao.Answer{
				Analysis: dao.AnswerElement{
					ID:        1,
					Content:   "Elasticsearch is a distributed search and analytics engine.",
					Keywords:  "",
					Shorthand: "",
					Highlight: "",
					Guidance:  "test_analysis_guidance",
				},
			},
			Utime: 1619708855,
		},
		{
			ID:      10008,
			UID:     101,
			Biz:     "test",
			BizID:   10008,
			Title:   "How to use Elasticsearch?",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Answer: dao.Answer{
				Basic: dao.AnswerElement{
					ID:        1,
					Content:   "test_basic_content",
					Keywords:  "test_basic_keywords",
					Shorthand: "",
					Highlight: "distributed search and analytics engine",
					Guidance:  "Learn more about Elasticsearch documentation.",
				},
			},
			Utime: 1619708855,
		},
		{
			ID:      10009,
			UID:     101,
			Biz:     "test",
			BizID:   10009,
			Title:   "How to use Elasticsearch?",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Answer: dao.Answer{
				Basic: dao.AnswerElement{
					ID:        1,
					Content:   "Elasticsearch is a distributed search and analytics engine.",
					Keywords:  "",
					Shorthand: "test_basic_shorthand",
					Highlight: "",
					Guidance:  "Learn more about Elasticsearch documentation.",
				},
			},
			Utime: 1619708855,
		},
		{
			ID:      10010,
			UID:     101,
			Biz:     "test",
			BizID:   10010,
			Title:   "How to use Elasticsearch?",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Answer: dao.Answer{
				Basic: dao.AnswerElement{
					ID:        1,
					Content:   "Elasticsearch is a distributed search and analytics engine.",
					Keywords:  "",
					Shorthand: "",
					Highlight: "test_basic_highlight",
					Guidance:  "",
				},
			},
			Utime: 1619708855,
		},
		{
			ID:      10011,
			UID:     101,
			Biz:     "test",
			BizID:   10011,
			Title:   "How to use Elasticsearch?",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Answer: dao.Answer{
				Basic: dao.AnswerElement{
					ID:        1,
					Content:   "Elasticsearch is a distributed search and analytics engine.",
					Keywords:  "",
					Shorthand: "",
					Highlight: "",
					Guidance:  "test_basic_guidance",
				},
			},
			Utime: 1619708855,
		},
		{
			ID:      10012,
			UID:     101,
			Biz:     "test",
			BizID:   10012,
			Title:   "How to use Elasticsearch?",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Answer: dao.Answer{
				Intermediate: dao.AnswerElement{
					ID:        1,
					Content:   "test_intermediate_content",
					Keywords:  "test_intermediate_keywords",
					Shorthand: "",
					Highlight: "distributed search and analytics engine",
					Guidance:  "Learn more about Elasticsearch documentation.",
				},
			},
			Utime: 1619708855,
		},
		{
			ID:      10013,
			UID:     101,
			Biz:     "test",
			BizID:   10013,
			Title:   "How to use Elasticsearch?",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Answer: dao.Answer{
				Intermediate: dao.AnswerElement{
					ID:        1,
					Content:   "Elasticsearch is a distributed search and analytics engine.",
					Keywords:  "",
					Shorthand: "test_intermediate_shorthand",
					Highlight: "",
					Guidance:  "Learn more about Elasticsearch documentation.",
				},
			},
			Utime: 1619708855,
		},
		{
			ID:      10014,
			UID:     101,
			Biz:     "test",
			BizID:   10014,
			Title:   "How to use Elasticsearch?",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Answer: dao.Answer{
				Intermediate: dao.AnswerElement{
					ID:        1,
					Content:   "Elasticsearch is a distributed search and analytics engine.",
					Keywords:  "",
					Shorthand: "",
					Highlight: "test_intermediate_highlight",
					Guidance:  "",
				},
			},
			Utime: 1619708855,
		},
		{
			ID:      10015,
			UID:     101,
			Biz:     "test",
			BizID:   10015,
			Title:   "How to use Elasticsearch?",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Answer: dao.Answer{
				Intermediate: dao.AnswerElement{
					ID:        1,
					Content:   "Elasticsearch is a distributed search and analytics engine.",
					Keywords:  "",
					Shorthand: "",
					Highlight: "",
					Guidance:  "test_intermediate_guidance",
				},
			},
			Utime: 1619708855,
		},
		{
			ID:      10016,
			UID:     101,
			Biz:     "test",
			BizID:   10016,
			Title:   "How to use Elasticsearch?",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Answer: dao.Answer{
				Advanced: dao.AnswerElement{
					ID:        1,
					Content:   "Elasticsearch is a distributed search and analytics engine.",
					Keywords:  "test_advanced_keywords",
					Shorthand: "",
					Highlight: "distributed search and analytics engine",
					Guidance:  "Learn more about Elasticsearch documentation.",
				},
			},
			Utime: 1619708855,
		},
		{
			ID:      10017,
			UID:     101,
			Biz:     "test",
			BizID:   10017,
			Title:   "How to use Elasticsearch?",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Answer: dao.Answer{
				Advanced: dao.AnswerElement{
					ID:        1,
					Content:   "Elasticsearch is a distributed search and analytics engine.",
					Keywords:  "",
					Shorthand: "test_advanced_shorthand",
					Highlight: "",
					Guidance:  "Learn more about Elasticsearch documentation.",
				},
			},
			Utime: 1619708855,
		},
		{
			ID:      10018,
			UID:     101,
			Biz:     "test",
			BizID:   10018,
			Title:   "How to use Elasticsearch?",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Answer: dao.Answer{
				Advanced: dao.AnswerElement{
					ID:        1,
					Content:   "Elasticsearch is a distributed search and analytics engine.",
					Keywords:  "",
					Shorthand: "",
					Highlight: "test_advanced_highlight",
					Guidance:  "",
				},
			},
			Utime: 1619708855,
		},
		{
			ID:      10019,
			UID:     101,
			Biz:     "test",
			BizID:   10019,
			Title:   "How to use Elasticsearch?",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Answer: dao.Answer{
				Advanced: dao.AnswerElement{
					ID:        1,
					Content:   "test_advanced_content",
					Keywords:  "",
					Shorthand: "",
					Highlight: "",
					Guidance:  "test_advanced_guidance",
				},
			},
			Utime: 1619708855,
		},
	}
	s.insertQuestion(questions)
}

func (s *HandlerTestSuite) insertQuestion(ques []dao.Question) {
	for _, que := range ques {
		by, err := json.Marshal(que)
		require.NoError(s.T(), err)
		_, err = s.es.Index(dao.PubQuestionIndexName).
			Id(strconv.FormatInt(que.ID, 10)).
			Raw(bytes.NewReader(by)).
			Do(context.Background())
		require.NoError(s.T(), err)
	}
}

func (s *HandlerTestSuite) insertCase(cas []dao.Case) {
	for _, ca := range cas {
		by, err := json.Marshal(ca)
		require.NoError(s.T(), err)
		_, err = s.es.Index(dao.PubCaseIndexName).
			Id(strconv.FormatInt(ca.Id, 10)).
			Raw(bytes.NewReader(by)).
			Do(context.Background())
		require.NoError(s.T(), err)
	}
}

func (s *HandlerTestSuite) initQuestionSets() {
	questionSets := []dao.QuestionSet{
		{
			Id:          5002,
			Uid:         123,
			Biz:         "test",
			BizID:       5002,
			Title:       "test_title",
			Description: "This is a test question set",
			Utime:       1713856231,
		},
		{
			Id:          5001,
			Uid:         123,
			Biz:         "test",
			BizID:       5001,
			Title:       "jjjkjk",
			Description: "test_desc",
			Utime:       1713856231,
		},
	}
	s.insertQuestionSet(questionSets)
}

func (s *HandlerTestSuite) insertQuestionSet(qs []dao.QuestionSet) {
	for _, q := range qs {
		by, err := json.Marshal(q)
		require.NoError(s.T(), err)
		_, err = s.es.Index(dao.QuestionSetIndexName).
			Id(strconv.FormatInt(q.Id, 10)).
			Raw(bytes.NewReader(by)).
			Do(context.Background())
		require.NoError(s.T(), err)
	}
}

func handlerSkillLevel(t *testing.T, sk web.SkillLevel) web.SkillLevel {
	assert.True(t, sk.Utime != "")
	assert.True(t, sk.Ctime != "")
	sk.Utime = ""
	sk.Ctime = ""
	return sk
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
