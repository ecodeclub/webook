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

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/search/internal/event"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/pkg/middleware"
	"github.com/ecodeclub/webook/internal/search/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/search/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/search/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const uid = 123

type AdminHandlerTestSuite struct {
	suite.Suite
	server   *egin.Component
	es       *elastic.Client
	producer mq.Producer
}

func (s *AdminHandlerTestSuite) SetupSuite() {
	adminHdl, err := startup.InitAdminHandler(&cases.Module{
		Svc: nil,
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
	adminHdl.PrivateRoutes(server.Engine)
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

func (s *AdminHandlerTestSuite) TearDownSuite() {
	// 创建范围查询，匹配 id > 10000 的文档
	query := elastic.NewRangeQuery("id").Gt(10000)
	_, err := s.es.DeleteByQuery("case_index").Query(query).Do(context.Background())
	require.NoError(s.T(), err)
	_, err = s.es.DeleteByQuery("question_index").Query(query).Do(context.Background())
	require.NoError(s.T(), err)
	_, err = s.es.DeleteByQuery("skill_index").Query(query).Do(context.Background())
	require.NoError(s.T(), err)
	_, err = s.es.DeleteByQuery(dao.QuestionSetIndexName).Query(query).Do(context.Background())
	require.NoError(s.T(), err)
}

func (s *AdminHandlerTestSuite) TestBizAdminSearch() {
	testCases := []struct {
		name    string
		before  func(t *testing.T)
		after   func(t *testing.T, wantRes web.SearchResult, actual web.SearchResult)
		wantAns web.SearchResult
		req     web.SearchReq
	}{
		{
			name: "搜索cases",
			before: func(t *testing.T) {
				s.initCases()
			},
			after: func(t *testing.T, wantRes web.SearchResult, actual web.SearchResult) {
				for idx := range actual.Cases {
					require.True(t, actual.Cases[idx].Utime != "")
					require.True(t, actual.Cases[idx].Ctime != "")
					actual.Cases[idx].Ctime = ""
					actual.Cases[idx].Utime = ""
				}
				assert.Equal(t, wantRes, actual)
			},
			wantAns: web.SearchResult{
				Cases: []web.Case{
					{
						Id:     10006,
						Uid:    1,
						Biz:    "test",
						BizID:  10006,
						Labels: []string{"label1"},
						Title:  "test_title",
						Content: web.EsVal{
							Val: "Elasticsearch内容",
						},

						GithubRepo: "Elasticsearch github代码库",
						GiteeRepo:  "Elasticsearch gitee代码库",
						Keywords:   "Elasticsearch关键词",
						Shorthand:  "Elasticsearch速记",
						Highlight:  "Elasticsearch亮点",
						Guidance:   "Elasticsearch引导",
						Status:     2,
						Result:     0,
					},
					{
						Id:     10005,
						Uid:    1,
						Biz:    "test",
						BizID:  10005,
						Labels: []string{"test_label"},
						Title:  "Elasticsearch标题",
						Content: web.EsVal{
							Val: "Elasticsearch内容",
						},
						GithubRepo: "Elasticsearch github代码库",
						GiteeRepo:  "Elasticsearch gitee代码库",
						Keywords:   "Elasticsearch关键词",
						Shorthand:  "Elasticsearch速记",
						Highlight:  "Elasticsearch亮点",
						Guidance:   "Elasticsearch引导",
						Status:     2,
						Result:     0,
					},
					{
						Id:     10002,
						Uid:    1,
						Biz:    "test",
						BizID:  10002,
						Labels: []string{"label1"},
						Title:  "Elasticsearch标题",
						Content: web.EsVal{
							Val: "Elasticsearch内容",
						},
						GithubRepo: "Elasticsearch github代码库",
						GiteeRepo:  "Elasticsearch gitee代码库",
						Keywords:   "test_keywords",
						Shorthand:  "Elasticsearch速记",
						Highlight:  "Elasticsearch亮点",
						Guidance:   "Elasticsearch引导",
						Status:     2,
						Result:     0,
					},
					{
						Id:     10003,
						Uid:    1,
						Biz:    "test",
						BizID:  10003,
						Labels: []string{"label1", "label2"},
						Title:  "Elasticsearch标题",
						Content: web.EsVal{
							Val: "Elasticsearch内容",
						},
						GithubRepo: "Elasticsearch github代码库",
						GiteeRepo:  "Elasticsearch gitee代码库",
						Keywords:   "Elasticsearch关键词",
						Shorthand:  "test_shorthands",
						Highlight:  "Elasticsearch亮点",
						Guidance:   "Elasticsearch引导",
						Status:     2,
						Result:     0,
					},
					{
						Id:     10001,
						Uid:    1,
						Biz:    "test",
						BizID:  10001,
						Labels: []string{"label1", "label2"},
						Title:  "Elasticsearch标题",
						Content: web.EsVal{
							Val: "test_content",
						},
						GithubRepo: "Elasticsearch github代码库",
						GiteeRepo:  "Elasticsearch gitee代码库",
						Keywords:   "Elasticsearch关键词",
						Shorthand:  "Elasticsearch速记",
						Highlight:  "Elasticsearch亮点",
						Guidance:   "Elasticsearch引导",
						Status:     2,
						Result:     0,
					},
					{
						Id:     10004,
						Uid:    1,
						Biz:    "test",
						BizID:  10004,
						Labels: []string{"label1", "label2"},
						Title:  "Elasticsearch标题",
						Content: web.EsVal{
							Val: "Elasticsearch内容",
						},
						GithubRepo: "Elasticsearch github代码库",
						GiteeRepo:  "Elasticsearch gitee代码库",
						Keywords:   "Elasticsearch关键词",
						Shorthand:  "Elasticsearch速记",
						Highlight:  "Elasticsearch亮点",
						Guidance:   "test_guidance",
						Status:     2,
						Result:     0,
					},
					{
						Id:     10007,
						Uid:    1,
						BizID:  10007,
						Biz:    "kkkk",
						Labels: []string{"xxxx"},
						Title:  "00000",
						Content: web.EsVal{
							Val: "Elasticsearch内容",
						},
						GithubRepo: "Elasticsearch github代码库",
						GiteeRepo:  "Elasticsearch gitee代码库",
						Keywords:   "Elasticsearch关键词",
						Shorthand:  "Elasticsearch速记",
						Highlight:  "Elasticsearch亮点",
						Guidance:   "Elasticsearch引导",
						Status:     2,
						Result:     0,
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
			after: func(t *testing.T, wantRes web.SearchResult, actual web.SearchResult) {
				for idx := range actual.Questions {
					require.True(t, actual.Questions[idx].Utime != "")
					actual.Questions[idx].Utime = ""
					if idx < 3 {
						assert.Equal(t, wantRes.Questions[idx], actual.Questions[idx])
					}
				}
				assert.ElementsMatch(t, wantRes.Questions, actual.Questions)

			},
			wantAns: web.SearchResult{
				Questions: []web.Question{
					{
						ID:    10002,
						Biz:   "test",
						BizID: 10002,

						UID:    101,
						Title:  "test_title",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
					},
					{
						ID:     10001,
						UID:    101,
						Biz:    "test",
						BizID:  10001,
						Title:  "dasdsa",
						Labels: []string{"test_label"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
					},
					{
						ID:     10004,
						Biz:    "test",
						BizID:  10004,
						UID:    101,
						Title:  "Elasticsearch",
						Labels: []string{"tElasticsearch"},
						Content: web.EsVal{
							Val: "test_content",
						},
						Status: 2,
					},
					{
						ID:     10003,
						UID:    101,
						Biz:    "test",
						BizID:  10003,
						Title:  "How to use Elasticsearch?",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
						Answer: web.Answer{
							Analysis: web.AnswerElement{
								ID: 1,
								Content: web.EsVal{
									Val: "Elasticsearch is a distributed search and analytics engine.",
								},
								Keywords:  "test_analysis_keywords",
								Shorthand: "ES",
								Highlight: "distributed search and analytics engine",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:     10005,
						UID:    101,
						Biz:    "test",
						BizID:  10005,
						Title:  "How to use Elasticsearch?",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
						Answer: web.Answer{
							Analysis: web.AnswerElement{
								ID: 1,
								Content: web.EsVal{
									Val: "Elasticsearch is a distributed search and analytics engine.",
								},
								Keywords:  "",
								Shorthand: "test_analysis_shorthand",
								Highlight: "distributed search and analytics engine",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:     10006,
						UID:    101,
						Biz:    "test",
						BizID:  10006,
						Title:  "How to use Elasticsearch?",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
						Answer: web.Answer{
							Analysis: web.AnswerElement{
								ID: 1,
								Content: web.EsVal{
									Val: "Elasticsearch is a distributed search and analytics engine.",
								},
								Keywords:  "",
								Shorthand: "",
								Highlight: "test_analysis_highlight",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:     10007,
						UID:    101,
						Biz:    "test",
						BizID:  10007,
						Title:  "How to use Elasticsearch?",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
						Answer: web.Answer{
							Analysis: web.AnswerElement{
								ID: 1,
								Content: web.EsVal{
									Val: "Elasticsearch is a distributed search and analytics engine.",
								},
								Keywords:  "",
								Shorthand: "",
								Highlight: "",
								Guidance:  "test_analysis_guidance",
							},
						},
					},
					{
						ID:     10008,
						UID:    101,
						Biz:    "test",
						BizID:  10008,
						Title:  "How to use Elasticsearch?",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
						Answer: web.Answer{
							Basic: web.AnswerElement{
								ID: 1,
								Content: web.EsVal{
									Val: "Elasticsearch is a distributed search and analytics engine.",
								},
								Keywords:  "test_basic_keywords",
								Shorthand: "",
								Highlight: "distributed search and analytics engine",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:     10009,
						UID:    101,
						Biz:    "test",
						BizID:  10009,
						Title:  "How to use Elasticsearch?",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
						Answer: web.Answer{
							Basic: web.AnswerElement{
								ID: 1,
								Content: web.EsVal{
									Val: "Elasticsearch is a distributed search and analytics engine.",
								},
								Keywords:  "",
								Shorthand: "test_basic_shorthand",
								Highlight: "",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:     10010,
						UID:    101,
						Biz:    "test",
						BizID:  10010,
						Title:  "How to use Elasticsearch?",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
						Answer: web.Answer{
							Basic: web.AnswerElement{
								ID: 1,
								Content: web.EsVal{
									Val: "Elasticsearch is a distributed search and analytics engine.",
								},
								Keywords:  "",
								Shorthand: "",
								Highlight: "test_basic_highlight",
								Guidance:  "",
							},
						},
					},
					{
						ID:     10011,
						UID:    101,
						Biz:    "test",
						BizID:  10011,
						Title:  "How to use Elasticsearch?",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
						Answer: web.Answer{
							Basic: web.AnswerElement{
								ID: 1,
								Content: web.EsVal{
									Val: "Elasticsearch is a distributed search and analytics engine.",
								},
								Keywords:  "",
								Shorthand: "",
								Highlight: "",
								Guidance:  "test_basic_guidance",
							},
						},
					},
					{
						ID:     10012,
						UID:    101,
						Biz:    "test",
						BizID:  10012,
						Title:  "How to use Elasticsearch?",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
						Answer: web.Answer{
							Intermediate: web.AnswerElement{
								ID: 1,
								Content: web.EsVal{
									Val: "Elasticsearch is a distributed search and analytics engine.",
								},
								Keywords:  "test_intermediate_keywords",
								Shorthand: "",
								Highlight: "distributed search and analytics engine",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:     10013,
						UID:    101,
						Biz:    "test",
						BizID:  10013,
						Title:  "How to use Elasticsearch?",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
						Answer: web.Answer{
							Intermediate: web.AnswerElement{
								ID: 1,
								Content: web.EsVal{
									Val: "Elasticsearch is a distributed search and analytics engine.",
								},
								Keywords:  "",
								Shorthand: "test_intermediate_shorthand",
								Highlight: "",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:     10014,
						UID:    101,
						Biz:    "test",
						BizID:  10014,
						Title:  "How to use Elasticsearch?",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
						Answer: web.Answer{
							Intermediate: web.AnswerElement{
								ID: 1,
								Content: web.EsVal{
									Val: "Elasticsearch is a distributed search and analytics engine.",
								},
								Keywords:  "",
								Shorthand: "",
								Highlight: "test_intermediate_highlight",
								Guidance:  "",
							},
						},
					},
					{
						ID:     10015,
						UID:    101,
						Biz:    "test",
						BizID:  10015,
						Title:  "How to use Elasticsearch?",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
						Answer: web.Answer{
							Intermediate: web.AnswerElement{
								ID: 1,
								Content: web.EsVal{
									Val: "Elasticsearch is a distributed search and analytics engine.",
								},
								Keywords:  "",
								Shorthand: "",
								Highlight: "",
								Guidance:  "test_intermediate_guidance",
							},
						},
					},
					{
						ID:     10016,
						UID:    101,
						Biz:    "test",
						BizID:  10016,
						Title:  "How to use Elasticsearch?",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
						Answer: web.Answer{
							Advanced: web.AnswerElement{
								ID: 1,
								Content: web.EsVal{
									Val: "Elasticsearch is a distributed search and analytics engine.",
								},
								Keywords:  "test_advanced_keywords",
								Shorthand: "",
								Highlight: "distributed search and analytics engine",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:     10017,
						UID:    101,
						Biz:    "test",
						BizID:  10017,
						Title:  "How to use Elasticsearch?",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
						Answer: web.Answer{
							Advanced: web.AnswerElement{
								ID: 1,
								Content: web.EsVal{
									Val: "Elasticsearch is a distributed search and analytics engine.",
								},
								Keywords:  "",
								Shorthand: "test_advanced_shorthand",
								Highlight: "",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:     10018,
						UID:    101,
						Biz:    "test",
						BizID:  10018,
						Title:  "How to use Elasticsearch?",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
						Answer: web.Answer{
							Advanced: web.AnswerElement{
								ID: 1,
								Content: web.EsVal{
									Val: "Elasticsearch is a distributed search and analytics engine.",
								},
								Keywords:  "",
								Shorthand: "",
								Highlight: "test_advanced_highlight",
								Guidance:  "",
							},
						},
					},
					{
						ID:     10019,
						UID:    101,
						Biz:    "test",
						BizID:  10019,
						Title:  "How to use Elasticsearch?",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
						Answer: web.Answer{
							Advanced: web.AnswerElement{
								ID: 1,
								Content: web.EsVal{
									Val: "Elasticsearch is a distributed search and analytics engine.",
								},
								Keywords:  "",
								Shorthand: "",
								Highlight: "",
								Guidance:  "test_advanced_guidance",
							},
						},
					},
				},
			},
			req: web.SearchReq{
				Keywords: "biz:question:test_content test_title test_label test_analysis_keywords test_analysis_shorthand test_analysis_highlight test_analysis_guidance test_basic_keywords test_basic_shorthand test_basic_highlight test_basic_guidance test_intermediate_keywords test_intermediate_shorthand test_intermediate_highlight test_intermediate_guidance test_advanced_keywords test_advanced_shorthand test_advanced_highlight test_advanced_guidance",
				Offset:   0,
				Limit:    20,
			},
		},
		{
			name: "搜索skills",
			before: func(t *testing.T) {
				s.initSkills()
			},
			after: func(t *testing.T, wantRes web.SearchResult, actual web.SearchResult) {
				for idx := range actual.Skills {
					require.True(t, actual.Skills[idx].Utime != "")
					actual.Skills[idx].Utime = ""
					actual.Skills[idx].Ctime = ""
					actual.Skills[idx].Basic = handlerSkillLevel(s.T(), actual.Skills[idx].Basic)
					actual.Skills[idx].Intermediate = handlerSkillLevel(s.T(), actual.Skills[idx].Intermediate)
					actual.Skills[idx].Advanced = handlerSkillLevel(s.T(), actual.Skills[idx].Advanced)
				}
				assert.ElementsMatch(t, wantRes.Skills, actual.Skills)

			},
			wantAns: web.SearchResult{
				Skills: []web.Skill{
					{
						ID:     10001,
						Labels: []string{"programming", "golang"},
						Name:   "test_name",
						Desc: web.EsVal{
							Val: "Learn Golang programming language",
						},
					},
					{
						ID:     10002,
						Labels: []string{"programming", "test_label"},
						Name:   "",
						Desc: web.EsVal{
							Val: "Learn Golang programming language",
						},
					},
					{
						ID:     10003,
						Labels: []string{"programming"},
						Name:   "",
						Desc: web.EsVal{
							Val: "test_desc",
						},
					},
					{
						ID:     10004,
						Labels: []string{"programming"},
						Name:   "",
						Basic: web.SkillLevel{
							ID: 1,
							Desc: web.EsVal{
								Val: "test_basic",
							},
							Questions: []int64{1},
							Cases:     []int64{1},
						},
					},
					{
						ID:     10005,
						Labels: []string{"programming"},
						Name:   "",
						Desc: web.EsVal{
							Val: "test_desc",
						},
						Intermediate: web.SkillLevel{
							ID: 2,
							Desc: web.EsVal{
								Val: "test_intermediate",
							},
							Questions: []int64{1},
							Cases:     []int64{1},
						},
					},
					{
						ID:     10006,
						Labels: []string{"programming"},
						Name:   "",
						Advanced: web.SkillLevel{
							ID: 2,
							Desc: web.EsVal{
								Val: "test_advanced",
							},
							Questions: []int64{1},
							Cases:     []int64{1},
						},
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
			after: func(t *testing.T, wantRes web.SearchResult, actual web.SearchResult) {
				for idx := range actual.QuestionSet {
					require.True(t, actual.QuestionSet[idx].Utime != "")
					actual.QuestionSet[idx].Utime = ""
				}
			},
			wantAns: web.SearchResult{
				QuestionSet: []web.QuestionSet{
					{
						Id:    10002,
						Uid:   123,
						Biz:   "test",
						BizID: 10002,
						Title: "test_title",
						Description: web.EsVal{
							Val: "This is a test question set",
						},
					},
					{
						Id:    10001,
						Uid:   123,
						Biz:   "test",
						BizID: 10001,
						Title: "jjjkjk",
						Description: web.EsVal{
							Val: "test_desc",
						},
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
			recorder := test.NewJSONResponseRecorder[web.SearchResult]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, 200, recorder.Code)
			data := recorder.MustScan().Data
			tc.after(t, tc.wantAns, data)
		})
	}

}

//func (s *AdminHandlerTestSuite) TestSearch() {
//	t := s.T()
//	s.initSearchData()
//	time.Sleep(1 * time.Second)
//	req, err := http.NewRequest(http.MethodPost,
//		"/search/list", iox.NewJSONReader(web.SearchReq{
//			Keywords: "biz:all:test_title",
//			Offset:   0,
//			Limit:    20,
//		}))
//	req.Header.Set("content-type", "application/json")
//	require.NoError(t, err)
//	recorder := test.NewJSONResponseRecorder[web.SearchResult]()
//	s.server.ServeHTTP(recorder, req)
//	require.Equal(t, 200, recorder.Code)
//	want := web.SearchResult{
//		Cases: []web.Case{
//			{
//				Id:         2,
//				Uid:        1,
//				Labels:     []string{"label1"},
//				Title:      "test_title",
//				Content:    "Elasticsearch内容",
//				GithubRepo: "Elasticsearch github代码库",
//				GiteeRepo:  "Elasticsearch gitee代码库",
//				Keywords:   "test_keywords",
//				Shorthand:  "Elasticsearch速记",
//				Highlight:  "Elasticsearch亮点",
//				Guidance:   "Elasticsearch引导",
//				Status:     2,
//				Result:     0,
//			},
//		},
//		Questions: []web.Question{
//			{
//				ID:      2,
//				UID:     101,
//				Title:   "test_title",
//				Labels:  []string{"elasticsearch", "search"},
//				Content: "I want to know how to use Elasticsearch for searching.",
//				Status:  2,
//			},
//		},
//		Skills: []web.Skill{
//			{
//				ID:     1,
//				Labels: []string{"programming", "golang"},
//				Name:   "test_title",
//				Desc:   "Learn Golang programming language",
//			},
//		},
//		QuestionSet: []web.QuestionSet{
//			{
//				Id:          2,
//				Uid:         123,
//				Title:       "test_title",
//				Description: "This is a test question set",
//			},
//		},
//	}
//	ans := recorder.MustScan().Data
//	for idx := range ans.Cases {
//		ans.Cases[idx].Utime = ""
//		ans.Cases[idx].Ctime = ""
//	}
//	for idx := range ans.Questions {
//		ans.Questions[idx].Utime = ""
//	}
//	for idx := range ans.QuestionSet {
//		ans.QuestionSet[idx].Utime = ""
//	}
//	for idx := range ans.Skills {
//		ans.Skills[idx].Ctime = ""
//		ans.Skills[idx].Utime = ""
//		ans.Skills[idx].Basic = handlerSkillLevel(t, ans.Skills[idx].Basic)
//		ans.Skills[idx].Intermediate = handlerSkillLevel(t, ans.Skills[idx].Intermediate)
//		ans.Skills[idx].Advanced = handlerSkillLevel(t, ans.Skills[idx].Advanced)
//	}
//	assert.Equal(t, want, ans)
//}
//
//func (s *AdminHandlerTestSuite) TestSearchWithCol() {
//	testCases := []struct {
//		name    string
//		before  func(t *testing.T)
//		after   func(t *testing.T, wantRes web.SearchResult, actual web.SearchResult)
//		wantAns web.SearchResult
//		req     web.SearchReq
//	}{
//		{
//			name: "搜索cases",
//			before: func(t *testing.T) {
//				s.initCases()
//			},
//			after: func(t *testing.T, wantRes web.SearchResult, actual web.SearchResult) {
//				for idx := range actual.Cases {
//					require.True(t, actual.Cases[idx].Utime != "")
//					require.True(t, actual.Cases[idx].Ctime != "")
//					actual.Cases[idx].Ctime = ""
//					actual.Cases[idx].Utime = ""
//				}
//				assert.Equal(t, wantRes, actual)
//			},
//			wantAns: web.SearchResult{
//				Cases: []web.Case{
//					{
//						Id:         6,
//						Uid:        1,
//						Labels:     []string{"label1"},
//						Title:      "test_title",
//						Content:    "Elasticsearch内容",
//						GithubRepo: "Elasticsearch github代码库",
//						GiteeRepo:  "Elasticsearch gitee代码库",
//						Keywords:   "Elasticsearch关键词",
//						Shorthand:  "Elasticsearch速记",
//						Highlight:  "Elasticsearch亮点",
//						Guidance:   "Elasticsearch引导",
//						Status:     2,
//						Result:     0,
//					},
//					{
//						Id:         5,
//						Uid:        1,
//						Labels:     []string{"test_label"},
//						Title:      "Elasticsearch标题",
//						Content:    "Elasticsearch内容",
//						GithubRepo: "Elasticsearch github代码库",
//						GiteeRepo:  "Elasticsearch gitee代码库",
//						Keywords:   "Elasticsearch关键词",
//						Shorthand:  "Elasticsearch速记",
//						Highlight:  "Elasticsearch亮点",
//						Guidance:   "Elasticsearch引导",
//						Status:     2,
//						Result:     1,
//					},
//				},
//			},
//			req: web.SearchReq{
//				Keywords: "biz:case:labels:test_label title:test_title",
//				Offset:   0,
//				Limit:    20,
//			},
//		},
//		{
//			name: "搜索questions",
//			before: func(t *testing.T) {
//				s.initQuestions()
//			},
//			after: func(t *testing.T, wantRes web.SearchResult, actual web.SearchResult) {
//				for idx := range actual.Questions {
//					require.True(t, actual.Questions[idx].Utime != "")
//					actual.Questions[idx].Utime = ""
//					if idx < 3 {
//						assert.Equal(t, wantRes.Questions[idx], actual.Questions[idx])
//					}
//				}
//				assert.ElementsMatch(t, wantRes.Questions, actual.Questions)
//
//			},
//			wantAns: web.SearchResult{
//				Questions: []web.Question{
//					{
//						ID:      2,
//						UID:     101,
//						Title:   "test_title",
//						Labels:  []string{"elasticsearch", "search"},
//						Content: "I want to know how to use Elasticsearch for searching.",
//						Status:  2,
//					},
//					{
//						ID:      1,
//						UID:     101,
//						Title:   "dasdsa",
//						Labels:  []string{"test_label"},
//						Content: "I want to know how to use Elasticsearch for searching.",
//						Status:  2,
//					},
//					{
//						ID:      12,
//						UID:     101,
//						Title:   "How to use Elasticsearch?",
//						Labels:  []string{"elasticsearch", "search"},
//						Content: "I want to know how to use Elasticsearch for searching.",
//						Status:  2,
//						Answer: web.Answer{
//							Intermediate: web.AnswerElement{
//								ID:        1,
//								Content:   "Elasticsearch is a distributed search and analytics engine.",
//								Keywords:  "test_intermediate_keywords",
//								Shorthand: "",
//								Highlight: "distributed search and analytics engine",
//								Guidance:  "Learn more about Elasticsearch documentation.",
//							},
//						},
//					},
//				},
//			},
//			req: web.SearchReq{
//				Keywords: "biz:question:title:test_title labels:test_label answer.intermediate.keywords:test_intermediate_keywords",
//				Offset:   0,
//				Limit:    20,
//			},
//		},
//		{
//			name: "搜索skills",
//			before: func(t *testing.T) {
//				s.initSkills()
//			},
//			after: func(t *testing.T, wantRes web.SearchResult, actual web.SearchResult) {
//				for idx := range actual.Skills {
//					require.True(t, actual.Skills[idx].Utime != "")
//					actual.Skills[idx].Utime = ""
//					actual.Skills[idx].Ctime = ""
//					actual.Skills[idx].Basic = handlerSkillLevel(s.T(), actual.Skills[idx].Basic)
//					actual.Skills[idx].Intermediate = handlerSkillLevel(s.T(), actual.Skills[idx].Intermediate)
//					actual.Skills[idx].Advanced = handlerSkillLevel(s.T(), actual.Skills[idx].Advanced)
//					if idx < 3 {
//						assert.Equal(t, wantRes.Skills[idx], actual.Skills[idx])
//					}
//				}
//				assert.ElementsMatch(t, wantRes.Skills, actual.Skills)
//
//			},
//			wantAns: web.SearchResult{
//				Skills: []web.Skill{
//					{
//						ID:     1,
//						Labels: []string{"programming", "golang"},
//						Name:   "test_name",
//						Desc:   "Learn Golang programming language",
//					},
//					{
//						ID:     2,
//						Labels: []string{"programming", "test_label"},
//						Name:   "",
//						Desc:   "Learn Golang programming language",
//					},
//					{
//						ID:     4,
//						Labels: []string{"programming"},
//						Name:   "",
//						Desc:   "",
//						Basic: web.SkillLevel{
//							ID:        1,
//							Desc:      "test_basic",
//							Questions: []int64{1},
//							Cases:     []int64{1},
//						},
//					},
//				},
//			},
//			req: web.SearchReq{
//				Keywords: "biz:skill:labels:golang name:test_name labels:test_label basic.desc:test_basic",
//				Offset:   0,
//				Limit:    20,
//			},
//		},
//		{
//			name: "搜索questionSets",
//			before: func(t *testing.T) {
//				s.initQuestionSets()
//			},
//			after: func(t *testing.T, wantRes web.SearchResult, actual web.SearchResult) {
//				for idx := range actual.QuestionSet {
//					require.True(t, actual.QuestionSet[idx].Utime != "")
//					actual.QuestionSet[idx].Utime = ""
//				}
//			},
//			wantAns: web.SearchResult{
//				QuestionSet: []web.QuestionSet{
//					{
//						Id:          2,
//						Uid:         123,
//						Title:       "test_title",
//						Description: "This is a test question set",
//					},
//					{
//						Id:          1,
//						Uid:         123,
//						Title:       "jjjkjk",
//						Description: "test_desc",
//					},
//				},
//			},
//			req: web.SearchReq{
//				Keywords: "biz:questionSet:title:test_title",
//				Offset:   0,
//				Limit:    20,
//			},
//		},
//	}
//	for _, tc := range testCases {
//		tc := tc
//		s.T().Run(tc.name, func(t *testing.T) {
//			tc.before(t)
//			time.Sleep(3 * time.Second)
//			req, err := http.NewRequest(http.MethodPost,
//				"/search/list", iox.NewJSONReader(tc.req))
//			req.Header.Set("content-type", "application/json")
//			require.NoError(t, err)
//			recorder := test.NewJSONResponseRecorder[web.SearchResult]()
//			s.server.ServeHTTP(recorder, req)
//			require.Equal(t, 200, recorder.Code)
//			tc.after(t, tc.wantAns, recorder.MustScan().Data)
//		})
//	}
//
//}

func (s *AdminHandlerTestSuite) TestSync() {
	testcases := []struct {
		name   string
		msg    event.SyncEvent
		before func(t *testing.T)
		after  func(t *testing.T)
	}{
		{
			name: "同步case",
			before: func(t *testing.T) {

			},
			msg: getCase(s.T()),
			after: func(t *testing.T) {
				res := s.getDataFromEs(t, "pub_case_index", "1")
				var ans dao.Case
				err := json.Unmarshal(res.Source, &ans)
				require.NoError(t, err)
				assert.Equal(t, dao.Case{
					Id:         1,
					Uid:        1001,
					Labels:     []string{"label1", "label2"},
					Title:      "Test Case",
					Content:    "Test Content",
					GithubRepo: "github.com/test",
					GiteeRepo:  "gitee.com/test",
					Keywords:   "test keywords",
					Shorthand:  "test shorthand",
					Highlight:  "test highlight",
					Guidance:   "test guidance",
					Status:     1,
					Ctime:      1619430000,
					Utime:      1619430000,
				}, ans)
			},
		},
		{
			name: "同步question",
			before: func(t *testing.T) {
			},
			msg: getQuestion(s.T()),
			after: func(t *testing.T) {
				res := s.getDataFromEs(t, dao.QuestionIndexName, "1")
				var ans dao.Question
				err := json.Unmarshal(res.Source, &ans)
				require.NoError(t, err)
				q := dao.Question{
					ID:      1,
					UID:     1001,
					Title:   "Example Question",
					Labels:  []string{"label1", "label2"},
					Content: "Example content",
					Status:  1,
					Answer: dao.Answer{
						Analysis: dao.AnswerElement{
							ID:        1,
							Content:   "Analysis content",
							Keywords:  "Analysis keywords",
							Shorthand: "Analysis shorthand",
							Highlight: "Analysis highlight",
							Guidance:  "Analysis guidance",
						},
						Basic: dao.AnswerElement{
							ID:        2,
							Content:   "Basic content",
							Keywords:  "Basic keywords",
							Shorthand: "Basic shorthand",
							Highlight: "Basic highlight",
							Guidance:  "Basic guidance",
						},
						Intermediate: dao.AnswerElement{
							ID:        3,
							Content:   "Intermediate content",
							Keywords:  "Intermediate keywords",
							Shorthand: "Intermediate shorthand",
							Highlight: "Intermediate highlight",
							Guidance:  "Intermediate guidance",
						},
						Advanced: dao.AnswerElement{
							ID:        4,
							Content:   "Advanced content",
							Keywords:  "Advanced keywords",
							Shorthand: "Advanced shorthand",
							Highlight: "Advanced highlight",
							Guidance:  "Advanced guidance",
						},
					},
					Utime: 1619430000,
				}
				assert.Equal(t, q, ans)
			},
		},
		{
			name: "如果文档存在就更新",
			msg:  getSkill(s.T()),
			before: func(t *testing.T) {
				skill := dao.Skill{
					ID:     99,
					Labels: []string{"old_label1", "label2"},
					Name:   "old Skill",
					Desc:   "old skill description",
					Ctime:  1619430000,
					Utime:  1619430000,
				}
				s.insertSkills([]dao.Skill{skill})
			},
			after: func(t *testing.T) {
				res := s.getDataFromEs(t, dao.SkillIndexName, "99")
				var ans dao.Skill
				err := json.Unmarshal(res.Source, &ans)
				require.NoError(t, err)
				skill := dao.Skill{
					ID:     99,
					Labels: []string{"label1", "label2"},
					Name:   "Example Skill",
					Desc:   "Example skill description",
					Basic: dao.SkillLevel{
						ID:        1,
						Desc:      "Basic",
						Ctime:     1619430000,
						Utime:     1619430000,
						Questions: []int64{1, 2, 3},
						Cases:     []int64{4, 5, 6},
					},
					Intermediate: dao.SkillLevel{
						ID:        2,
						Desc:      "Intermediate",
						Ctime:     1619430000,
						Utime:     1619430000,
						Questions: []int64{4, 5, 6},
						Cases:     []int64{7, 8, 9},
					},
					Advanced: dao.SkillLevel{
						ID:        3,
						Desc:      "Advanced",
						Ctime:     1619430000,
						Utime:     1619430000,
						Questions: []int64{7, 8, 9},
						Cases:     []int64{10, 11, 12},
					},
					Ctime: 1619430000,
					Utime: 1619430000,
				}
				assert.Equal(t, skill, ans)
			},
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			val, err := json.Marshal(tc.msg)
			require.NoError(t, err)
			msg := &mq.Message{
				Value: val,
			}
			_, err = s.producer.Produce(context.Background(), msg)
			require.NoError(t, err)
			time.Sleep(10 * time.Second)
			tc.after(t)
		})
	}

}

func (s *AdminHandlerTestSuite) TestSearchLimit() {
	testCases := []struct {
		name    string
		before  func(t *testing.T)
		after   func(t *testing.T, wantRes web.SearchResult, actual web.SearchResult)
		wantAns web.SearchResult
		req     web.SearchReq
	}{
		{
			name: "搜索cases分页",
			before: func(t *testing.T) {
				s.initCases()
			},
			after: func(t *testing.T, wantRes web.SearchResult, actual web.SearchResult) {
				for idx := range actual.Cases {
					require.True(t, actual.Cases[idx].Utime != "")
					require.True(t, actual.Cases[idx].Ctime != "")
					actual.Cases[idx].Ctime = ""
					actual.Cases[idx].Utime = ""
				}
				assert.Equal(t, wantRes, actual)
			},
			wantAns: web.SearchResult{
				Cases: []web.Case{
					{
						Id:     10006,
						BizID:  10006,
						Biz:    "test",
						Uid:    1,
						Labels: []string{"label1"},
						Title:  "test_title",
						Content: web.EsVal{
							Val: "Elasticsearch内容",
						},
						GithubRepo: "Elasticsearch github代码库",
						GiteeRepo:  "Elasticsearch gitee代码库",
						Keywords:   "Elasticsearch关键词",
						Shorthand:  "Elasticsearch速记",
						Highlight:  "Elasticsearch亮点",
						Guidance:   "Elasticsearch引导",
						Status:     2,
						Result:     0,
					},
					{
						Id:     10005,
						Uid:    1,
						BizID:  10005,
						Biz:    "test",
						Labels: []string{"test_label"},
						Title:  "Elasticsearch标题",
						Content: web.EsVal{
							Val: "Elasticsearch内容",
						},
						GithubRepo: "Elasticsearch github代码库",
						GiteeRepo:  "Elasticsearch gitee代码库",
						Keywords:   "Elasticsearch关键词",
						Shorthand:  "Elasticsearch速记",
						Highlight:  "Elasticsearch亮点",
						Guidance:   "Elasticsearch引导",
						Status:     2,
						Result:     0,
					},
				},
			},
			req: web.SearchReq{
				Keywords: "biz:case:test_content test_keywords test_shorthands test_guidance test_title test_label",
				Offset:   0,
				Limit:    2,
			},
		},
		{
			name: "搜索questions分页",
			before: func(t *testing.T) {
				s.initQuestions()
			},
			after: func(t *testing.T, wantRes web.SearchResult, actual web.SearchResult) {
				for idx := range actual.Questions {
					require.True(t, actual.Questions[idx].Utime != "")
					actual.Questions[idx].Utime = ""
					if idx < 3 {
						assert.Equal(t, wantRes.Questions[idx], actual.Questions[idx])
					}
				}
				assert.ElementsMatch(t, wantRes.Questions, actual.Questions)

			},
			wantAns: web.SearchResult{
				Questions: []web.Question{
					{
						ID:     10002,
						BizID:  10002,
						Biz:    "test",
						UID:    101,
						Title:  "test_title",
						Labels: []string{"elasticsearch", "search"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
					},
					{
						ID:     10001,
						BizID:  10001,
						Biz:    "test",
						UID:    101,
						Title:  "dasdsa",
						Labels: []string{"test_label"},
						Content: web.EsVal{
							Val: "I want to know how to use Elasticsearch for searching.",
						},
						Status: 2,
					},
				},
			},
			req: web.SearchReq{
				Keywords: "biz:question:test_content test_title test_label test_analysis_keywords test_analysis_shorthand test_analysis_highlight test_analysis_guidance test_basic_keywords test_basic_shorthand test_basic_highlight test_basic_guidance test_intermediate_keywords test_intermediate_shorthand test_intermediate_highlight test_intermediate_guidance test_advanced_keywords test_advanced_shorthand test_advanced_highlight test_advanced_guidance",
				Offset:   0,
				Limit:    2,
			},
		},
		{
			name: "搜索skills分页",
			before: func(t *testing.T) {
				s.initSkills()
			},
			after: func(t *testing.T, wantRes web.SearchResult, actual web.SearchResult) {
				for idx := range actual.Skills {
					require.True(t, actual.Skills[idx].Utime != "")
					actual.Skills[idx].Utime = ""
					actual.Skills[idx].Ctime = ""
					actual.Skills[idx].Basic = handlerSkillLevel(s.T(), actual.Skills[idx].Basic)
					actual.Skills[idx].Intermediate = handlerSkillLevel(s.T(), actual.Skills[idx].Intermediate)
					actual.Skills[idx].Advanced = handlerSkillLevel(s.T(), actual.Skills[idx].Advanced)
				}
				assert.ElementsMatch(t, wantRes.Skills, actual.Skills)

			},
			wantAns: web.SearchResult{
				Skills: []web.Skill{
					{
						ID:     10001,
						Labels: []string{"programming", "golang"},
						Name:   "test_name",
						Desc: web.EsVal{
							Val: "Learn Golang programming language",
						},
					},
					{
						ID:     10002,
						Labels: []string{"programming", "test_label"},
						Name:   "",
						Desc: web.EsVal{
							Val: "Learn Golang programming language",
						},
					},
					{
						ID:     10003,
						Labels: []string{"programming"},
						Name:   "",
						Desc: web.EsVal{
							Val: "test_desc",
						},
					},
				},
			},
			req: web.SearchReq{
				Keywords: "biz:skill:test_name test_label test_desc test_advanced test_basic test_intermediate",
				Offset:   0,
				Limit:    3,
			},
		},
		{
			name: "搜索questionSets分页",
			before: func(t *testing.T) {
				s.initQuestionSets()
			},
			after: func(t *testing.T, wantRes web.SearchResult, actual web.SearchResult) {
				for idx := range actual.QuestionSet {
					require.True(t, actual.QuestionSet[idx].Utime != "")
					actual.QuestionSet[idx].Utime = ""
				}
			},
			wantAns: web.SearchResult{
				QuestionSet: []web.QuestionSet{
					{
						Id:    10002,
						Uid:   123,
						Title: "test_title",
						Description: web.EsVal{
							Val: "This is a test question set",
						},
					},
				},
			},
			req: web.SearchReq{
				Keywords: "biz:questionSet:test_title test_desc",
				Offset:   0,
				Limit:    1,
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
			recorder := test.NewJSONResponseRecorder[web.SearchResult]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, 200, recorder.Code)
			tc.after(t, tc.wantAns, recorder.MustScan().Data)
		})
	}
}

func (s *AdminHandlerTestSuite) getDataFromEs(t *testing.T, index, docID string) *elastic.GetResult {
	doc, err := s.es.Get().
		Index(index).
		Id(docID).
		Do(context.Background())
	require.NoError(t, err)
	return doc
}

func getCase(t *testing.T) event.SyncEvent {
	evt := event.SyncEvent{
		Biz:   "case",
		BizID: 1,
		Live:  true,
	}
	val := dao.Case{
		Id:         1,
		Uid:        1001,
		Labels:     []string{"label1", "label2"},
		Title:      "Test Case",
		Content:    "Test Content",
		GithubRepo: "github.com/test",
		GiteeRepo:  "gitee.com/test",
		Keywords:   "test keywords",
		Shorthand:  "test shorthand",
		Highlight:  "test highlight",
		Guidance:   "test guidance",
		Status:     1,
		Ctime:      1619430000,
		Utime:      1619430000,
	}
	caseByte, err := json.Marshal(val)
	require.NoError(t, err)
	evt.Data = string(caseByte)
	return evt
}

func getQuestion(t *testing.T) event.SyncEvent {
	eve := event.SyncEvent{
		Biz:   "question",
		BizID: 1,
	}
	question := dao.Question{
		ID:      1,
		UID:     1001,
		Title:   "Example Question",
		Labels:  []string{"label1", "label2"},
		Content: "Example content",
		Status:  1,
		Answer: dao.Answer{
			Analysis: dao.AnswerElement{
				ID:        1,
				Content:   "Analysis content",
				Keywords:  "Analysis keywords",
				Shorthand: "Analysis shorthand",
				Highlight: "Analysis highlight",
				Guidance:  "Analysis guidance",
			},
			Basic: dao.AnswerElement{
				ID:        2,
				Content:   "Basic content",
				Keywords:  "Basic keywords",
				Shorthand: "Basic shorthand",
				Highlight: "Basic highlight",
				Guidance:  "Basic guidance",
			},
			Intermediate: dao.AnswerElement{
				ID:        3,
				Content:   "Intermediate content",
				Keywords:  "Intermediate keywords",
				Shorthand: "Intermediate shorthand",
				Highlight: "Intermediate highlight",
				Guidance:  "Intermediate guidance",
			},
			Advanced: dao.AnswerElement{
				ID:        4,
				Content:   "Advanced content",
				Keywords:  "Advanced keywords",
				Shorthand: "Advanced shorthand",
				Highlight: "Advanced highlight",
				Guidance:  "Advanced guidance",
			},
		},
		Utime: 1619430000,
	}
	questionByte, err := json.Marshal(question)
	require.NoError(t, err)
	eve.Data = string(questionByte)
	return eve
}

func getSkill(t *testing.T) event.SyncEvent {
	eve := event.SyncEvent{
		Biz:   "skill",
		BizID: 99,
	}
	skill := dao.Skill{
		ID:     99,
		Labels: []string{"label1", "label2"},
		Name:   "Example Skill",
		Desc:   "Example skill description",
		Basic: dao.SkillLevel{
			ID:        1,
			Desc:      "Basic",
			Ctime:     1619430000,
			Utime:     1619430000,
			Questions: []int64{1, 2, 3},
			Cases:     []int64{4, 5, 6},
		},
		Intermediate: dao.SkillLevel{
			ID:        2,
			Desc:      "Intermediate",
			Ctime:     1619430000,
			Utime:     1619430000,
			Questions: []int64{4, 5, 6},
			Cases:     []int64{7, 8, 9},
		},
		Advanced: dao.SkillLevel{
			ID:        3,
			Desc:      "Advanced",
			Ctime:     1619430000,
			Utime:     1619430000,
			Questions: []int64{7, 8, 9},
			Cases:     []int64{10, 11, 12},
		},
		Ctime: 1619430000,
		Utime: 1619430000,
	}
	questionByte, err := json.Marshal(skill)
	require.NoError(t, err)
	eve.Data = string(questionByte)
	return eve
}

func (s *AdminHandlerTestSuite) initCases() {
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

func (s *AdminHandlerTestSuite) initQuestions() {
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
					Content:   "Elasticsearch is a distributed search and analytics engine.",
					Keywords:  "test_analysis_keywords",
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
					Content:   "Elasticsearch is a distributed search and analytics engine.",
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
					Content:   "Elasticsearch is a distributed search and analytics engine.",
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
					Content:   "Elasticsearch is a distributed search and analytics engine.",
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

func (s *AdminHandlerTestSuite) initSkills() {
	skills := []dao.Skill{
		{
			ID:     10001,
			Labels: []string{"programming", "golang"},
			Name:   "test_name",
			Desc:   "Learn Golang programming language",
		},
		{
			ID:     10002,
			Labels: []string{"programming", "test_label"},
			Name:   "",
			Desc:   "Learn Golang programming language",
		},
		{
			ID:     10003,
			Labels: []string{"programming"},
			Name:   "",
			Desc:   "test_desc",
		},
		{
			ID:     10004,
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
		},
		{
			ID:     10005,
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
		},
		{
			ID:     10006,
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
		},
	}
	for _, skill := range skills {
		by, err := json.Marshal(skill)
		require.NoError(s.T(), err)
		_, err = s.es.Index().
			Index(dao.SkillIndexName).
			Id(strconv.FormatInt(skill.ID, 10)).
			BodyJson(string(by)).Do(context.Background())
		require.NoError(s.T(), err)
	}
}

func (s *AdminHandlerTestSuite) initQuestionSets() {
	questionSets := []dao.QuestionSet{
		{
			Id:          10002,
			Uid:         123,
			Biz:         "test",
			BizID:       10002,
			Title:       "test_title",
			Description: "This is a test question set",
			Utime:       1713856231,
		},
		{
			Id:          10001,
			Uid:         123,
			Biz:         "test",
			BizID:       10001,
			Title:       "jjjkjk",
			Description: "test_desc",
			Utime:       1713856231,
		},
	}
	s.insertQuestionSet(questionSets)
}

func (s *AdminHandlerTestSuite) insertQuestion(ques []dao.Question) {
	for _, que := range ques {
		by, err := json.Marshal(que)
		require.NoError(s.T(), err)
		_, err = s.es.Index().
			Index(dao.QuestionIndexName).
			Id(strconv.FormatInt(que.ID, 10)).
			BodyJson(string(by)).Do(context.Background())
		require.NoError(s.T(), err)
	}
}

func (s *AdminHandlerTestSuite) insertCase(cas []dao.Case) {
	for _, ca := range cas {
		by, err := json.Marshal(ca)
		require.NoError(s.T(), err)
		_, err = s.es.Index().
			Index(dao.CaseIndexName).
			Id(strconv.FormatInt(ca.Id, 10)).
			BodyJson(string(by)).Do(context.Background())
		require.NoError(s.T(), err)
	}
}

func (s *AdminHandlerTestSuite) insertQuestionSet(qs []dao.QuestionSet) {
	for _, q := range qs {
		by, err := json.Marshal(q)
		require.NoError(s.T(), err)
		_, err = s.es.Index().
			Index(dao.QuestionSetIndexName).
			Id(strconv.FormatInt(q.Id, 10)).
			BodyJson(string(by)).Do(context.Background())
		require.NoError(s.T(), err)
	}
}

func (s *AdminHandlerTestSuite) insertSkills(sks []dao.Skill) {
	for _, sk := range sks {
		by, err := json.Marshal(sk)
		require.NoError(s.T(), err)
		_, err = s.es.Index().
			Index(dao.SkillIndexName).
			Id(strconv.FormatInt(sk.ID, 10)).
			BodyJson(string(by)).Do(context.Background())
		require.NoError(s.T(), err)
	}
}

func (s *AdminHandlerTestSuite) initSearchData() {
	cas := []dao.Case{
		{
			Id:         2,
			Uid:        1,
			Labels:     []string{"label1"},
			Title:      "test_title",
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
	}
	s.insertCase(cas)
	ques := []dao.Question{
		{
			ID:      2,
			UID:     101,
			Title:   "test_title",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Utime:   1619708855,
		},
	}
	s.insertQuestion(ques)
	skills := []dao.Skill{
		{
			ID:     1,
			Labels: []string{"programming", "golang"},
			Name:   "test_title",
			Desc:   "Learn Golang programming language",
		},
	}
	s.insertSkills(skills)
	qs := []dao.QuestionSet{
		{
			Id:          2,
			Uid:         123,
			Title:       "test_title",
			Description: "This is a test question set",
			Utime:       1713856231,
		},
	}
	s.insertQuestionSet(qs)
}

//func handlerSkillLevel(t *testing.T, sk web.SkillLevel) web.SkillLevel {
//	assert.True(t, sk.Utime != "")
//	assert.True(t, sk.Ctime != "")
//	sk.Utime = ""
//	sk.Ctime = ""
//	return sk
//}

func TestAdminHandler(t *testing.T) {
	suite.Run(t, new(AdminHandlerTestSuite))
}
