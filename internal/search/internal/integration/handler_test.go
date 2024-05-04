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

type HandlerTestSuite struct {
	suite.Suite
	server   *egin.Component
	es       *elastic.Client
	producer mq.Producer
}

func (s *HandlerTestSuite) SetupSuite() {
	handler, err := startup.InitHandler()
	require.NoError(s.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	handler.PublicRoutes(server.Engine)
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: uid,
			Data: map[string]string{
				"memberDDL": strconv.FormatInt(time.Now().Add(time.Hour).UnixMilli(), 10),
			},
		}))
	})
	server.Use(middleware.NewCheckMembershipMiddlewareBuilder(nil).Build())
	s.server = server
	s.es = testioc.InitES()
	testmq := testioc.InitMQ()
	p, err := testmq.Producer(event.SyncTopic)
	if err != nil {
		panic(err)
	}
	s.producer = p
}

func (s *HandlerTestSuite) TearDownSuite() {
	_, err := s.es.DeleteIndex(dao.SkillIndexName).Do(context.Background())
	require.NoError(s.T(), err)
	_, err = s.es.DeleteIndex(dao.CaseIndexName).Do(context.Background())
	require.NoError(s.T(), err)
	_, err = s.es.DeleteIndex(dao.QuestionIndexName).Do(context.Background())
	require.NoError(s.T(), err)
	_, err = s.es.DeleteIndex(dao.QuestionSetIndexName).Do(context.Background())
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) TearDownTest() {
	var err error
	query := elastic.NewMatchAllQuery()
	_, err = s.es.DeleteByQuery(dao.CaseIndexName).Query(query).Do(context.Background())
	require.NoError(s.T(), err)
	_, err = s.es.DeleteByQuery(dao.SkillIndexName).Query(query).Do(context.Background())
	require.NoError(s.T(), err)
	_, err = s.es.DeleteByQuery(dao.QuestionIndexName).Query(query).Do(context.Background())
	require.NoError(s.T(), err)
	_, err = s.es.DeleteByQuery(dao.QuestionSetIndexName).Query(query).Do(context.Background())
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) TestBizSearch() {
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
						Id:        6,
						Uid:       1,
						Labels:    []string{"label1"},
						Title:     "test_title",
						Content:   "Elasticsearch内容",
						CodeRepo:  "Elasticsearch代码库",
						Keywords:  "Elasticsearch关键词",
						Shorthand: "Elasticsearch速记",
						Highlight: "Elasticsearch亮点",
						Guidance:  "Elasticsearch引导",
						Status:    2,
					},
					{
						Id:        5,
						Uid:       1,
						Labels:    []string{"test_label"},
						Title:     "Elasticsearch标题",
						Content:   "Elasticsearch内容",
						CodeRepo:  "Elasticsearch代码库",
						Keywords:  "Elasticsearch关键词",
						Shorthand: "Elasticsearch速记",
						Highlight: "Elasticsearch亮点",
						Guidance:  "Elasticsearch引导",
						Status:    2,
					},
					{
						Id:        2,
						Uid:       1,
						Labels:    []string{"label1"},
						Title:     "Elasticsearch标题",
						Content:   "Elasticsearch内容",
						CodeRepo:  "Elasticsearch代码库",
						Keywords:  "test_keywords",
						Shorthand: "Elasticsearch速记",
						Highlight: "Elasticsearch亮点",
						Guidance:  "Elasticsearch引导",
						Status:    2,
					},
					{
						Id:        3,
						Uid:       1,
						Labels:    []string{"label1", "label2"},
						Title:     "Elasticsearch标题",
						Content:   "Elasticsearch内容",
						CodeRepo:  "Elasticsearch代码库",
						Keywords:  "Elasticsearch关键词",
						Shorthand: "test_shorthands",
						Highlight: "Elasticsearch亮点",
						Guidance:  "Elasticsearch引导",
						Status:    2,
					},
					{
						Id:        1,
						Uid:       1,
						Labels:    []string{"label1", "label2"},
						Title:     "Elasticsearch标题",
						Content:   "test_content",
						CodeRepo:  "Elasticsearch代码库",
						Keywords:  "Elasticsearch关键词",
						Shorthand: "Elasticsearch速记",
						Highlight: "Elasticsearch亮点",
						Guidance:  "Elasticsearch引导",
						Status:    2,
					},
					{
						Id:        4,
						Uid:       1,
						Labels:    []string{"label1", "label2"},
						Title:     "Elasticsearch标题",
						Content:   "Elasticsearch内容",
						CodeRepo:  "Elasticsearch代码库",
						Keywords:  "Elasticsearch关键词",
						Shorthand: "Elasticsearch速记",
						Highlight: "Elasticsearch亮点",
						Guidance:  "test_guidance",
						Status:    2,
					},
				},
			},
			req: web.SearchReq{
				KeyWords: "biz:case:test_content test_keywords test_shorthands test_guidance test_title test_label",
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
						ID:      2,
						UID:     101,
						Title:   "test_title",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
					},
					{
						ID:      1,
						UID:     101,
						Title:   "dasdsa",
						Labels:  []string{"test_label"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
					},
					{
						ID:      4,
						UID:     101,
						Title:   "Elasticsearch",
						Labels:  []string{"tElasticsearch"},
						Content: "test_content",
						Status:  2,
					},
					{
						ID:      3,
						UID:     101,
						Title:   "How to use Elasticsearch?",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
						Answer: web.Answer{
							Analysis: web.AnswerElement{
								ID:        1,
								Content:   "Elasticsearch is a distributed search and analytics engine.",
								Keywords:  "test_analysis_keywords",
								Shorthand: "ES",
								Highlight: "distributed search and analytics engine",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:      5,
						UID:     101,
						Title:   "How to use Elasticsearch?",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
						Answer: web.Answer{
							Analysis: web.AnswerElement{
								ID:        1,
								Content:   "Elasticsearch is a distributed search and analytics engine.",
								Keywords:  "",
								Shorthand: "test_analysis_shorthand",
								Highlight: "distributed search and analytics engine",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:      6,
						UID:     101,
						Title:   "How to use Elasticsearch?",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
						Answer: web.Answer{
							Analysis: web.AnswerElement{
								ID:        1,
								Content:   "Elasticsearch is a distributed search and analytics engine.",
								Keywords:  "",
								Shorthand: "",
								Highlight: "test_analysis_highlight",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:      7,
						UID:     101,
						Title:   "How to use Elasticsearch?",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
						Answer: web.Answer{
							Analysis: web.AnswerElement{
								ID:        1,
								Content:   "Elasticsearch is a distributed search and analytics engine.",
								Keywords:  "",
								Shorthand: "",
								Highlight: "",
								Guidance:  "test_analysis_guidance",
							},
						},
					},
					{
						ID:      8,
						UID:     101,
						Title:   "How to use Elasticsearch?",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
						Answer: web.Answer{
							Basic: web.AnswerElement{
								ID:        1,
								Content:   "Elasticsearch is a distributed search and analytics engine.",
								Keywords:  "test_basic_keywords",
								Shorthand: "",
								Highlight: "distributed search and analytics engine",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:      9,
						UID:     101,
						Title:   "How to use Elasticsearch?",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
						Answer: web.Answer{
							Basic: web.AnswerElement{
								ID:        1,
								Content:   "Elasticsearch is a distributed search and analytics engine.",
								Keywords:  "",
								Shorthand: "test_basic_shorthand",
								Highlight: "",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:      10,
						UID:     101,
						Title:   "How to use Elasticsearch?",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
						Answer: web.Answer{
							Basic: web.AnswerElement{
								ID:        1,
								Content:   "Elasticsearch is a distributed search and analytics engine.",
								Keywords:  "",
								Shorthand: "",
								Highlight: "test_basic_highlight",
								Guidance:  "",
							},
						},
					},
					{
						ID:      11,
						UID:     101,
						Title:   "How to use Elasticsearch?",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
						Answer: web.Answer{
							Basic: web.AnswerElement{
								ID:        1,
								Content:   "Elasticsearch is a distributed search and analytics engine.",
								Keywords:  "",
								Shorthand: "",
								Highlight: "",
								Guidance:  "test_basic_guidance",
							},
						},
					},
					{
						ID:      12,
						UID:     101,
						Title:   "How to use Elasticsearch?",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
						Answer: web.Answer{
							Intermediate: web.AnswerElement{
								ID:        1,
								Content:   "Elasticsearch is a distributed search and analytics engine.",
								Keywords:  "test_intermediate_keywords",
								Shorthand: "",
								Highlight: "distributed search and analytics engine",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:      13,
						UID:     101,
						Title:   "How to use Elasticsearch?",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
						Answer: web.Answer{
							Intermediate: web.AnswerElement{
								ID:        1,
								Content:   "Elasticsearch is a distributed search and analytics engine.",
								Keywords:  "",
								Shorthand: "test_intermediate_shorthand",
								Highlight: "",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:      14,
						UID:     101,
						Title:   "How to use Elasticsearch?",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
						Answer: web.Answer{
							Intermediate: web.AnswerElement{
								ID:        1,
								Content:   "Elasticsearch is a distributed search and analytics engine.",
								Keywords:  "",
								Shorthand: "",
								Highlight: "test_intermediate_highlight",
								Guidance:  "",
							},
						},
					},
					{
						ID:      15,
						UID:     101,
						Title:   "How to use Elasticsearch?",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
						Answer: web.Answer{
							Intermediate: web.AnswerElement{
								ID:        1,
								Content:   "Elasticsearch is a distributed search and analytics engine.",
								Keywords:  "",
								Shorthand: "",
								Highlight: "",
								Guidance:  "test_intermediate_guidance",
							},
						},
					},
					{
						ID:      16,
						UID:     101,
						Title:   "How to use Elasticsearch?",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
						Answer: web.Answer{
							Advanced: web.AnswerElement{
								ID:        1,
								Content:   "Elasticsearch is a distributed search and analytics engine.",
								Keywords:  "test_advanced_keywords",
								Shorthand: "",
								Highlight: "distributed search and analytics engine",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:      17,
						UID:     101,
						Title:   "How to use Elasticsearch?",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
						Answer: web.Answer{
							Advanced: web.AnswerElement{
								ID:        1,
								Content:   "Elasticsearch is a distributed search and analytics engine.",
								Keywords:  "",
								Shorthand: "test_advanced_shorthand",
								Highlight: "",
								Guidance:  "Learn more about Elasticsearch documentation.",
							},
						},
					},
					{
						ID:      18,
						UID:     101,
						Title:   "How to use Elasticsearch?",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
						Answer: web.Answer{
							Advanced: web.AnswerElement{
								ID:        1,
								Content:   "Elasticsearch is a distributed search and analytics engine.",
								Keywords:  "",
								Shorthand: "",
								Highlight: "test_advanced_highlight",
								Guidance:  "",
							},
						},
					},
					{
						ID:      19,
						UID:     101,
						Title:   "How to use Elasticsearch?",
						Labels:  []string{"elasticsearch", "search"},
						Content: "I want to know how to use Elasticsearch for searching.",
						Status:  2,
						Answer: web.Answer{
							Advanced: web.AnswerElement{
								ID:        1,
								Content:   "Elasticsearch is a distributed search and analytics engine.",
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
				KeyWords: "biz:question:test_content test_title test_label test_analysis_keywords test_analysis_shorthand test_analysis_highlight test_analysis_guidance test_basic_keywords test_basic_shorthand test_basic_highlight test_basic_guidance test_intermediate_keywords test_intermediate_shorthand test_intermediate_highlight test_intermediate_guidance test_advanced_keywords test_advanced_shorthand test_advanced_highlight test_advanced_guidance",
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
					if idx < 3 {
						assert.Equal(t, wantRes.Skills[idx], actual.Skills[idx])
					}
				}
				assert.ElementsMatch(t, wantRes.Skills, actual.Skills)

			},
			wantAns: web.SearchResult{
				Skills: []web.Skill{
					{
						ID:     1,
						Labels: []string{"programming", "golang"},
						Name:   "test_name",
						Desc:   "Learn Golang programming language",
					},
					{
						ID:     2,
						Labels: []string{"programming", "test_label"},
						Name:   "",
						Desc:   "Learn Golang programming language",
					},
					{
						ID:     3,
						Labels: []string{"programming"},
						Name:   "",
						Desc:   "test_desc",
					},
					{
						ID:     4,
						Labels: []string{"programming"},
						Name:   "",
						Desc:   "",
						Basic: web.SkillLevel{
							ID:        1,
							Desc:      "test_basic",
							Questions: []int64{1},
							Cases:     []int64{1},
						},
					},
					{
						ID:     5,
						Labels: []string{"programming"},
						Name:   "",
						Desc:   "",
						Intermediate: web.SkillLevel{
							ID:        2,
							Desc:      "test_intermediate",
							Questions: []int64{1},
							Cases:     []int64{1},
						},
					},
					{
						ID:     6,
						Labels: []string{"programming"},
						Name:   "",
						Desc:   "",
						Advanced: web.SkillLevel{
							ID:        2,
							Desc:      "test_advanced",
							Questions: []int64{1},
							Cases:     []int64{1},
						},
					},
				},
			},
			req: web.SearchReq{
				KeyWords: "biz:skill:test_name test_label test_desc test_advanced test_basic test_intermediate",
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
						Id:          2,
						Uid:         123,
						Title:       "test_title",
						Description: "This is a test question set",
					},
					{
						Id:          1,
						Uid:         123,
						Title:       "jjjkjk",
						Description: "test_desc",
					},
				},
			},
			req: web.SearchReq{
				KeyWords: "biz:questionSet:test_title test_desc",
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			time.Sleep(1 * time.Second)
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

func (s *HandlerTestSuite) TestSearch() {
	t := s.T()
	s.initSearchData()
	time.Sleep(1 * time.Second)
	req, err := http.NewRequest(http.MethodPost,
		"/search/list", iox.NewJSONReader(web.SearchReq{
			KeyWords: "biz:all:test_title",
		}))
	req.Header.Set("content-type", "application/json")
	require.NoError(t, err)
	recorder := test.NewJSONResponseRecorder[web.SearchResult]()
	s.server.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
	want := web.SearchResult{
		Cases: []web.Case{
			{
				Id:        2,
				Uid:       1,
				Labels:    []string{"label1"},
				Title:     "test_title",
				Content:   "Elasticsearch内容",
				CodeRepo:  "Elasticsearch代码库",
				Keywords:  "test_keywords",
				Shorthand: "Elasticsearch速记",
				Highlight: "Elasticsearch亮点",
				Guidance:  "Elasticsearch引导",
				Status:    2,
			},
		},
		Questions: []web.Question{
			{
				ID:      2,
				UID:     101,
				Title:   "test_title",
				Labels:  []string{"elasticsearch", "search"},
				Content: "I want to know how to use Elasticsearch for searching.",
				Status:  2,
			},
		},
		Skills: []web.Skill{
			{
				ID:     1,
				Labels: []string{"programming", "golang"},
				Name:   "test_title",
				Desc:   "Learn Golang programming language",
			},
		},
		QuestionSet: []web.QuestionSet{
			{
				Id:          2,
				Uid:         123,
				Title:       "test_title",
				Description: "This is a test question set",
			},
		},
	}
	ans := recorder.MustScan().Data
	for idx := range ans.Cases {
		ans.Cases[idx].Utime = ""
		ans.Cases[idx].Ctime = ""
	}
	for idx := range ans.Questions {
		ans.Questions[idx].Utime = ""
	}
	for idx := range ans.QuestionSet {
		ans.QuestionSet[idx].Utime = ""
	}
	for idx := range ans.Skills {
		ans.Skills[idx].Ctime = ""
		ans.Skills[idx].Utime = ""
		ans.Skills[idx].Basic = handlerSkillLevel(t, ans.Skills[idx].Basic)
		ans.Skills[idx].Intermediate = handlerSkillLevel(t, ans.Skills[idx].Intermediate)
		ans.Skills[idx].Advanced = handlerSkillLevel(t, ans.Skills[idx].Advanced)
	}
	assert.Equal(t, want, ans)
}

func (s *HandlerTestSuite) TestSync() {
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
				res := s.getDataFromEs(t, dao.CaseIndexName, "1")
				var ans dao.Case
				err := json.Unmarshal(res.Source, &ans)
				require.NoError(t, err)
				assert.Equal(t, dao.Case{
					Id:        1,
					Uid:       1001,
					Labels:    []string{"label1", "label2"},
					Title:     "Test Case",
					Content:   "Test Content",
					CodeRepo:  "github.com/test",
					Keywords:  "test keywords",
					Shorthand: "test shorthand",
					Highlight: "test highlight",
					Guidance:  "test guidance",
					Status:    1,
					Ctime:     1619430000,
					Utime:     1619430000,
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

func (s *HandlerTestSuite) getDataFromEs(t *testing.T, index, docID string) *elastic.GetResult {
	doc, err := s.es.Get().
		Index(index).
		Id(docID).
		Do(context.Background())
	require.NoError(t, err)
	return doc
}

func getCase(t *testing.T) event.SyncEvent {
	event := event.SyncEvent{
		Biz:   "case",
		BizID: 1,
	}
	val := dao.Case{
		Id:        1,
		Uid:       1001,
		Labels:    []string{"label1", "label2"},
		Title:     "Test Case",
		Content:   "Test Content",
		CodeRepo:  "github.com/test",
		Keywords:  "test keywords",
		Shorthand: "test shorthand",
		Highlight: "test highlight",
		Guidance:  "test guidance",
		Status:    1,
		Ctime:     1619430000,
		Utime:     1619430000,
	}
	caseByte, err := json.Marshal(val)
	require.NoError(t, err)
	event.Data = string(caseByte)
	return event
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

func (s *HandlerTestSuite) initCases() {
	testcases := []dao.Case{
		{
			Id:        1,
			Uid:       1,
			Labels:    []string{"label1", "label2"},
			Title:     "Elasticsearch标题",
			Content:   "test_content",
			CodeRepo:  "Elasticsearch代码库",
			Keywords:  "Elasticsearch关键词",
			Shorthand: "Elasticsearch速记",
			Highlight: "Elasticsearch亮点",
			Guidance:  "Elasticsearch引导",
			Status:    2,
			Ctime:     1619708855,
			Utime:     1619708855,
		},
		{
			Id:        2,
			Uid:       1,
			Labels:    []string{"label1"},
			Title:     "Elasticsearch标题",
			Content:   "Elasticsearch内容",
			CodeRepo:  "Elasticsearch代码库",
			Keywords:  "test_keywords",
			Shorthand: "Elasticsearch速记",
			Highlight: "Elasticsearch亮点",
			Guidance:  "Elasticsearch引导",
			Status:    2,
			Ctime:     1619708855,
			Utime:     1619708855,
		},
		{
			Id:        3,
			Uid:       1,
			Labels:    []string{"label1", "label2"},
			Title:     "Elasticsearch标题",
			Content:   "Elasticsearch内容",
			CodeRepo:  "Elasticsearch代码库",
			Keywords:  "Elasticsearch关键词",
			Shorthand: "test_shorthands",
			Highlight: "Elasticsearch亮点",
			Guidance:  "Elasticsearch引导",
			Status:    2,
			Ctime:     1619708855,
			Utime:     1619708855,
		},
		{
			Id:        4,
			Uid:       1,
			Labels:    []string{"label1", "label2"},
			Title:     "Elasticsearch标题",
			Content:   "Elasticsearch内容",
			CodeRepo:  "Elasticsearch代码库",
			Keywords:  "Elasticsearch关键词",
			Shorthand: "Elasticsearch速记",
			Highlight: "Elasticsearch亮点",
			Guidance:  "test_guidance",
			Status:    2,
			Ctime:     1619708855,
			Utime:     1619708855,
		},
		{
			Id:        5,
			Uid:       1,
			Labels:    []string{"test_label"},
			Title:     "Elasticsearch标题",
			Content:   "Elasticsearch内容",
			CodeRepo:  "Elasticsearch代码库",
			Keywords:  "Elasticsearch关键词",
			Shorthand: "Elasticsearch速记",
			Highlight: "Elasticsearch亮点",
			Guidance:  "Elasticsearch引导",
			Status:    2,
			Ctime:     1619708855,
			Utime:     1619708855,
		},
		{
			Id:        6,
			Uid:       1,
			Labels:    []string{"label1"},
			Title:     "test_title",
			Content:   "Elasticsearch内容",
			CodeRepo:  "Elasticsearch代码库",
			Keywords:  "Elasticsearch关键词",
			Shorthand: "Elasticsearch速记",
			Highlight: "Elasticsearch亮点",
			Guidance:  "Elasticsearch引导",
			Status:    2,
			Ctime:     1619708855,
			Utime:     1619708855,
		},
		{
			Id:        7,
			Uid:       1,
			Labels:    []string{"label1", "test_label"},
			Title:     "test_title未发布",
			Content:   "Elasticsearch内容",
			CodeRepo:  "Elasticsearch代码库",
			Keywords:  "Elasticsearch关键词",
			Shorthand: "Elasticsearch速记",
			Highlight: "Elasticsearch亮点",
			Guidance:  "Elasticsearch引导",
			Status:    1,
			Ctime:     1619708855,
			Utime:     1619708855,
		},
	}
	s.insertCase(testcases)
}

func (s *HandlerTestSuite) initQuestions() {
	questions := []dao.Question{
		{
			ID:      2,
			UID:     101,
			Title:   "test_title",
			Labels:  []string{"elasticsearch", "search"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Utime:   1619708855,
		},
		{
			ID:      1,
			UID:     101,
			Title:   "dasdsa",
			Labels:  []string{"test_label"},
			Content: "I want to know how to use Elasticsearch for searching.",
			Status:  2,
			Utime:   1619708855,
		},
		{
			ID:      3,
			UID:     101,
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
			ID:      4,
			UID:     101,
			Title:   "Elasticsearch",
			Labels:  []string{"tElasticsearch"},
			Content: "test_content",
			Status:  2,
			Utime:   1619708855,
		},
		{
			ID:      5,
			UID:     101,
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
			ID:      6,
			UID:     101,
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
			ID:      7,
			UID:     101,
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
			ID:      8,
			UID:     101,
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
			ID:      9,
			UID:     101,
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
			ID:      10,
			UID:     101,
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
			ID:      11,
			UID:     101,
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
			ID:      12,
			UID:     101,
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
			ID:      13,
			UID:     101,
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
			ID:      14,
			UID:     101,
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
			ID:      15,
			UID:     101,
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
			ID:      16,
			UID:     101,
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
			ID:      17,
			UID:     101,
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
			ID:      18,
			UID:     101,
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
			ID:      19,
			UID:     101,
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

func (s *HandlerTestSuite) initSkills() {
	skills := []dao.Skill{
		{
			ID:     1,
			Labels: []string{"programming", "golang"},
			Name:   "test_name",
			Desc:   "Learn Golang programming language",
		},
		{
			ID:     2,
			Labels: []string{"programming", "test_label"},
			Name:   "",
			Desc:   "Learn Golang programming language",
		},
		{
			ID:     3,
			Labels: []string{"programming"},
			Name:   "",
			Desc:   "test_desc",
		},
		{
			ID:     4,
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
			ID:     5,
			Labels: []string{"programming"},
			Name:   "",
			Desc:   "",
			Intermediate: dao.SkillLevel{
				ID:        2,
				Desc:      "test_intermediate",
				Utime:     1619708855,
				Questions: []int64{1},
				Cases:     []int64{1},
			},
		},
		{
			ID:     6,
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

func (s *HandlerTestSuite) initQuestionSets() {
	questionSets := []dao.QuestionSet{
		{
			Id:          2,
			Uid:         123,
			Title:       "test_title",
			Description: "This is a test question set",
			Utime:       1713856231,
		},
		{
			Id:          1,
			Uid:         123,
			Title:       "jjjkjk",
			Description: "test_desc",
			Utime:       1713856231,
		},
	}
	s.insertQuestionSet(questionSets)
}

func (s *HandlerTestSuite) insertQuestion(ques []dao.Question) {
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

func (s *HandlerTestSuite) insertCase(cas []dao.Case) {
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

func (s *HandlerTestSuite) insertQuestionSet(qs []dao.QuestionSet) {
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

func (s *HandlerTestSuite) insertSkills(sks []dao.Skill) {
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

func (s *HandlerTestSuite) initSearchData() {
	cas := []dao.Case{
		{
			Id:        2,
			Uid:       1,
			Labels:    []string{"label1"},
			Title:     "test_title",
			Content:   "Elasticsearch内容",
			CodeRepo:  "Elasticsearch代码库",
			Keywords:  "test_keywords",
			Shorthand: "Elasticsearch速记",
			Highlight: "Elasticsearch亮点",
			Guidance:  "Elasticsearch引导",
			Status:    2,
			Ctime:     1619708855,
			Utime:     1619708855,
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

func handlerSkillLevel(t *testing.T, sk web.SkillLevel) web.SkillLevel {
	assert.True(t, sk.Utime != "")
	assert.True(t, sk.Ctime != "")
	sk.Utime = ""
	sk.Ctime = ""
	return sk
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
