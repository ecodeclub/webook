//go:build e2e

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ecodeclub/webook/internal/question/internal/repository/cache"

	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/event"
	"github.com/ecodeclub/webook/internal/question/internal/repository"
	"github.com/ecodeclub/webook/internal/question/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SearchSyncTestSuite struct {
	suite.Suite
	db     *egorm.Component
	repo   repository.Repository
	client *elastic.Client
	svc    service.SearchSyncService
}

func (s *SearchSyncTestSuite) SetupSuite() {
	s.db = testioc.InitDB()
	err := dao.InitTables(s.db)
	ca := testioc.InitCache()
	require.NoError(s.T(), err)

	// Initialize elastic client
	client, err := elastic.NewClient(
		elastic.SetURL("http://localhost:9200"),
		elastic.SetSniff(false),
	)
	require.NoError(s.T(), err)
	s.client = client
	queCache := cache.NewQuestionECache(ca)

	// Initialize repository
	s.repo = repository.NewCacheRepository(dao.NewGORMQuestionDAO(s.db), queCache)

	// Initialize service
	s.svc = service.NewSearchSyncService(s.repo, s.client)
}

func (s *SearchSyncTestSuite) TearDownSuite() {
	// Clean up elasticsearch indices
	_, err := s.client.DeleteIndex("question_index").Do(context.Background())
	require.NoError(s.T(), err)
	_, err = s.client.DeleteIndex("pub_question_index").Do(context.Background())
	require.NoError(s.T(), err)
	err = s.db.WithContext(s.T().Context()).Where("biz = ?", "test").Delete(&dao.Question{}).Error
	require.NoError(s.T(), err)
	err = s.db.WithContext(s.T().Context()).Where("biz = ?", "test").Delete(&dao.PublishQuestion{}).Error
	require.NoError(s.T(), err)
}

func (s *SearchSyncTestSuite) TestSyncAll() {
	ctx := context.Background()

	// 创建3条制作库数据
	productionQuestions := []domain.Question{
		{
			Uid:     456,
			Title:   "Production Question 1",
			Content: "Production Content 1",
			Biz:     "test",
			BizId:   1,
			Status:  domain.UnPublishedStatus,
			Answer: domain.Answer{
				Analysis: domain.AnswerElement{
					Id:        1,
					Content:   "Analysis 1",
					Keywords:  "Keywords 1",
					Shorthand: "Shorthand 1",
					Highlight: "Highlight 1",
					Guidance:  "Guidance 1",
				},
				Basic: domain.AnswerElement{
					Id:        2,
					Content:   "Basic 1",
					Keywords:  "Keywords 2",
					Shorthand: "Shorthand 2",
					Highlight: "Highlight 2",
					Guidance:  "Guidance 2",
				},
				Intermediate: domain.AnswerElement{
					Id:        3,
					Content:   "Intermediate 1",
					Keywords:  "Keywords 3",
					Shorthand: "Shorthand 3",
					Highlight: "Highlight 3",
					Guidance:  "Guidance 3",
				},
				Advanced: domain.AnswerElement{
					Id:        4,
					Content:   "Advanced 1",
					Keywords:  "Keywords 4",
					Shorthand: "Shorthand 4",
					Highlight: "Highlight 4",
					Guidance:  "Guidance 4",
				},
			},
		},
		{
			Uid:     456,
			Title:   "Production Question 2",
			Content: "Production Content 2",
			Biz:     "test",
			BizId:   2,
			Status:  domain.UnPublishedStatus,
			Answer: domain.Answer{
				Analysis: domain.AnswerElement{
					Id:        5,
					Content:   "Analysis 2",
					Keywords:  "Keywords 5",
					Shorthand: "Shorthand 5",
					Highlight: "Highlight 5",
					Guidance:  "Guidance 5",
				},
				Basic: domain.AnswerElement{
					Id:        6,
					Content:   "Basic 2",
					Keywords:  "Keywords 6",
					Shorthand: "Shorthand 6",
					Highlight: "Highlight 6",
					Guidance:  "Guidance 6",
				},
				Intermediate: domain.AnswerElement{
					Id:        7,
					Content:   "Intermediate 2",
					Keywords:  "Keywords 7",
					Shorthand: "Shorthand 7",
					Highlight: "Highlight 7",
					Guidance:  "Guidance 7",
				},
				Advanced: domain.AnswerElement{
					Id:        8,
					Content:   "Advanced 2",
					Keywords:  "Keywords 8",
					Shorthand: "Shorthand 8",
					Highlight: "Highlight 8",
					Guidance:  "Guidance 8",
				},
			},
		},
		{
			Uid:     456,
			Title:   "Production Question 3",
			Content: "Production Content 3",
			Biz:     "test",
			BizId:   3,
			Status:  domain.UnPublishedStatus,
			Answer: domain.Answer{
				Analysis: domain.AnswerElement{
					Id:        9,
					Content:   "Analysis 3",
					Keywords:  "Keywords 9",
					Shorthand: "Shorthand 9",
					Highlight: "Highlight 9",
					Guidance:  "Guidance 9",
				},
				Basic: domain.AnswerElement{
					Id:        10,
					Content:   "Basic 3",
					Keywords:  "Keywords 10",
					Shorthand: "Shorthand 10",
					Highlight: "Highlight 10",
					Guidance:  "Guidance 10",
				},
				Intermediate: domain.AnswerElement{
					Id:        11,
					Content:   "Intermediate 3",
					Keywords:  "Keywords 11",
					Shorthand: "Shorthand 11",
					Highlight: "Highlight 11",
					Guidance:  "Guidance 11",
				},
				Advanced: domain.AnswerElement{
					Id:        12,
					Content:   "Advanced 3",
					Keywords:  "Keywords 12",
					Shorthand: "Shorthand 12",
					Highlight: "Highlight 12",
					Guidance:  "Guidance 12",
				},
			},
		},
	}

	// 创建2条线上库数据
	publishedQuestions := []domain.Question{
		{
			Uid:     456,
			Title:   "Published Question 1",
			Content: "Published Content 1",
			Biz:     "test",
			BizId:   4,
			Status:  domain.PublishedStatus,
			Answer: domain.Answer{
				Analysis: domain.AnswerElement{
					Id:        13,
					Content:   "Analysis 4",
					Keywords:  "Keywords 13",
					Shorthand: "Shorthand 13",
					Highlight: "Highlight 13",
					Guidance:  "Guidance 13",
				},
				Basic: domain.AnswerElement{
					Id:        14,
					Content:   "Basic 4",
					Keywords:  "Keywords 14",
					Shorthand: "Shorthand 14",
					Highlight: "Highlight 14",
					Guidance:  "Guidance 14",
				},
				Intermediate: domain.AnswerElement{
					Id:        15,
					Content:   "Intermediate 4",
					Keywords:  "Keywords 15",
					Shorthand: "Shorthand 15",
					Highlight: "Highlight 15",
					Guidance:  "Guidance 15",
				},
				Advanced: domain.AnswerElement{
					Id:        16,
					Content:   "Advanced 4",
					Keywords:  "Keywords 16",
					Shorthand: "Shorthand 16",
					Highlight: "Highlight 16",
					Guidance:  "Guidance 16",
				},
			},
		},
		{
			Uid:     456,
			Title:   "Published Question 2",
			Content: "Published Content 2",
			Biz:     "test",
			BizId:   5,
			Status:  domain.PublishedStatus,
			Answer: domain.Answer{
				Analysis: domain.AnswerElement{
					Id:        17,
					Content:   "Analysis 5",
					Keywords:  "Keywords 17",
					Shorthand: "Shorthand 17",
					Highlight: "Highlight 17",
					Guidance:  "Guidance 17",
				},
				Basic: domain.AnswerElement{
					Id:        18,
					Content:   "Basic 5",
					Keywords:  "Keywords 18",
					Shorthand: "Shorthand 18",
					Highlight: "Highlight 18",
					Guidance:  "Guidance 18",
				},
				Intermediate: domain.AnswerElement{
					Id:        19,
					Content:   "Intermediate 5",
					Keywords:  "Keywords 19",
					Shorthand: "Shorthand 19",
					Highlight: "Highlight 19",
					Guidance:  "Guidance 19",
				},
				Advanced: domain.AnswerElement{
					Id:        20,
					Content:   "Advanced 5",
					Keywords:  "Keywords 20",
					Shorthand: "Shorthand 20",
					Highlight: "Highlight 20",
					Guidance:  "Guidance 20",
				},
			},
		},
	}

	// 保存制作库数据
	for idx, q := range productionQuestions {
		id, err := s.repo.Create(ctx, &q)
		require.NoError(s.T(), err)
		productionQuestions[idx].Id = id
	}

	// 保存并同步线上库数据
	for idx, q := range publishedQuestions {
		id, err := s.repo.Sync(ctx, &q)
		require.NoError(s.T(), err)
		publishedQuestions[idx].Id = id
	}

	// 运行同步
	s.svc.SyncAll()
	productionQuestions = append(productionQuestions, publishedQuestions...)
	time.Sleep(3 * time.Second)
	for _, q := range productionQuestions {
		res, err := s.client.Get().
			Index("question_index").
			Id(fmt.Sprintf("%d", q.Id)).
			Do(ctx)
		require.NoError(s.T(), err)
		assert.True(s.T(), res.Found)

		// 解析ES返回的数据
		var esQuestion event.Question
		err = json.Unmarshal([]byte(res.Source), &esQuestion)
		require.NoError(s.T(), err)

		// 验证字段匹配
		assert.Equal(s.T(), q.Id, esQuestion.ID)
		assert.Equal(s.T(), q.Title, esQuestion.Title)
		assert.Equal(s.T(), q.Content, esQuestion.Content)
		assert.Equal(s.T(), q.Biz, esQuestion.Biz)
		assert.Equal(s.T(), q.BizId, esQuestion.BizId)
		assert.Equal(s.T(), q.Status.ToUint8(), esQuestion.Status)

		// 验证 Answer 字段
		assert.Equal(s.T(), q.Answer.Analysis.Content, esQuestion.Answer.Analysis.Content)
		assert.Equal(s.T(), q.Answer.Analysis.Keywords, esQuestion.Answer.Analysis.Keywords)
		assert.Equal(s.T(), q.Answer.Analysis.Shorthand, esQuestion.Answer.Analysis.Shorthand)
		assert.Equal(s.T(), q.Answer.Analysis.Highlight, esQuestion.Answer.Analysis.Highlight)
		assert.Equal(s.T(), q.Answer.Analysis.Guidance, esQuestion.Answer.Analysis.Guidance)

		assert.Equal(s.T(), q.Answer.Basic.Content, esQuestion.Answer.Basic.Content)
		assert.Equal(s.T(), q.Answer.Basic.Keywords, esQuestion.Answer.Basic.Keywords)
		assert.Equal(s.T(), q.Answer.Basic.Shorthand, esQuestion.Answer.Basic.Shorthand)
		assert.Equal(s.T(), q.Answer.Basic.Highlight, esQuestion.Answer.Basic.Highlight)
		assert.Equal(s.T(), q.Answer.Basic.Guidance, esQuestion.Answer.Basic.Guidance)

		assert.Equal(s.T(), q.Answer.Intermediate.Content, esQuestion.Answer.Intermediate.Content)
		assert.Equal(s.T(), q.Answer.Intermediate.Keywords, esQuestion.Answer.Intermediate.Keywords)
		assert.Equal(s.T(), q.Answer.Intermediate.Shorthand, esQuestion.Answer.Intermediate.Shorthand)
		assert.Equal(s.T(), q.Answer.Intermediate.Highlight, esQuestion.Answer.Intermediate.Highlight)
		assert.Equal(s.T(), q.Answer.Intermediate.Guidance, esQuestion.Answer.Intermediate.Guidance)

		assert.Equal(s.T(), q.Answer.Advanced.Content, esQuestion.Answer.Advanced.Content)
		assert.Equal(s.T(), q.Answer.Advanced.Keywords, esQuestion.Answer.Advanced.Keywords)
		assert.Equal(s.T(), q.Answer.Advanced.Shorthand, esQuestion.Answer.Advanced.Shorthand)
		assert.Equal(s.T(), q.Answer.Advanced.Highlight, esQuestion.Answer.Advanced.Highlight)
		assert.Equal(s.T(), q.Answer.Advanced.Guidance, esQuestion.Answer.Advanced.Guidance)
	}

	// 验证线上库数据同步到 pub_question_index
	for _, q := range publishedQuestions {
		res, err := s.client.Get().
			Index("pub_question_index").
			Id(fmt.Sprintf("%d", q.Id)).
			Do(ctx)
		require.NoError(s.T(), err)
		assert.True(s.T(), res.Found)

		// 解析ES返回的数据
		var esQuestion event.Question
		err = json.Unmarshal([]byte(res.Source), &esQuestion)
		require.NoError(s.T(), err)

		// 验证字段匹配
		assert.Equal(s.T(), q.Id, esQuestion.ID)
		assert.Equal(s.T(), q.Title, esQuestion.Title)
		assert.Equal(s.T(), q.Content, esQuestion.Content)
		assert.Equal(s.T(), q.Biz, esQuestion.Biz)
		assert.Equal(s.T(), q.BizId, esQuestion.BizId)
		assert.Equal(s.T(), q.Status.ToUint8(), esQuestion.Status)

		// 验证 Answer 字段
		assert.Equal(s.T(), q.Answer.Analysis.Content, esQuestion.Answer.Analysis.Content)
		assert.Equal(s.T(), q.Answer.Analysis.Keywords, esQuestion.Answer.Analysis.Keywords)
		assert.Equal(s.T(), q.Answer.Analysis.Shorthand, esQuestion.Answer.Analysis.Shorthand)
		assert.Equal(s.T(), q.Answer.Analysis.Highlight, esQuestion.Answer.Analysis.Highlight)
		assert.Equal(s.T(), q.Answer.Analysis.Guidance, esQuestion.Answer.Analysis.Guidance)

		assert.Equal(s.T(), q.Answer.Basic.Content, esQuestion.Answer.Basic.Content)
		assert.Equal(s.T(), q.Answer.Basic.Keywords, esQuestion.Answer.Basic.Keywords)
		assert.Equal(s.T(), q.Answer.Basic.Shorthand, esQuestion.Answer.Basic.Shorthand)
		assert.Equal(s.T(), q.Answer.Basic.Highlight, esQuestion.Answer.Basic.Highlight)
		assert.Equal(s.T(), q.Answer.Basic.Guidance, esQuestion.Answer.Basic.Guidance)

		assert.Equal(s.T(), q.Answer.Intermediate.Content, esQuestion.Answer.Intermediate.Content)
		assert.Equal(s.T(), q.Answer.Intermediate.Keywords, esQuestion.Answer.Intermediate.Keywords)
		assert.Equal(s.T(), q.Answer.Intermediate.Shorthand, esQuestion.Answer.Intermediate.Shorthand)
		assert.Equal(s.T(), q.Answer.Intermediate.Highlight, esQuestion.Answer.Intermediate.Highlight)
		assert.Equal(s.T(), q.Answer.Intermediate.Guidance, esQuestion.Answer.Intermediate.Guidance)

		assert.Equal(s.T(), q.Answer.Advanced.Content, esQuestion.Answer.Advanced.Content)
		assert.Equal(s.T(), q.Answer.Advanced.Keywords, esQuestion.Answer.Advanced.Keywords)
		assert.Equal(s.T(), q.Answer.Advanced.Shorthand, esQuestion.Answer.Advanced.Shorthand)
		assert.Equal(s.T(), q.Answer.Advanced.Highlight, esQuestion.Answer.Advanced.Highlight)
		assert.Equal(s.T(), q.Answer.Advanced.Guidance, esQuestion.Answer.Advanced.Guidance)
	}

}

func TestSearchSync(t *testing.T) {
	suite.Run(t, new(SearchSyncTestSuite))
}
