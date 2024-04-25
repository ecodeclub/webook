package repository

import (
	"context"
	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/ecodeclub/webook/internal/search/internal/repository/dao"
	"time"
)

type questionRepository struct {
	questionDao dao.QuestionDAO
}

func NewQuestionRepo(questionDao dao.QuestionDAO) QuestionRepo {
	return &questionRepository{
		questionDao: questionDao,
	}
}

func (q *questionRepository) InputQuestion(ctx context.Context, msg domain.Question) error {
	return q.questionDao.InputQuestion(ctx, q.questionToEntity(msg))
}

func (q *questionRepository) SearchQuestion(ctx context.Context, keywords []string) ([]domain.Question, error) {
	ques, err := q.questionDao.SearchQuestion(ctx, keywords)
	if err != nil {
		return nil, err
	}
	ans := make([]domain.Question, 0, len(ques))
	for _, qu := range ques {
		ans = append(ans, q.questionToDomain(qu))
	}
	return ans, nil
}

func (que *questionRepository) questionToEntity(q domain.Question) dao.Question {
	return dao.Question{
		ID:      q.ID,
		UID:     q.UID,
		Title:   q.Title,
		Labels:  q.Labels,
		Content: q.Content,
		Status:  q.Status,
		Answer: dao.Answer{
			Analysis:     que.ansToEntity(q.Answer.Analysis),
			Basic:        que.ansToEntity(q.Answer.Basic),
			Intermediate: que.ansToEntity(q.Answer.Intermediate),
			Advanced:     que.ansToEntity(q.Answer.Advanced),
		},
	}
}

func (*questionRepository) ansToEntity(q domain.AnswerElement) dao.AnswerElement {
	return dao.AnswerElement{
		ID:        q.ID,
		Content:   q.Content,
		Keywords:  q.Keywords,
		Shorthand: q.Shorthand,
		Highlight: q.Highlight,
		Guidance:  q.Guidance,
		Utime:     q.Utime.UnixMilli(),
	}
}

func (q *questionRepository) questionToDomain(que dao.Question) domain.Question {
	return domain.Question{
		ID:      que.ID,
		UID:     que.UID,
		Title:   que.Title,
		Labels:  que.Labels,
		Content: que.Content,
		Status:  que.Status,
		Answer: domain.Answer{
			Analysis:     q.ansToDomain(que.Answer.Analysis),
			Basic:        q.ansToDomain(que.Answer.Basic),
			Intermediate: q.ansToDomain(que.Answer.Intermediate),
			Advanced:     q.ansToDomain(que.Answer.Advanced),
		},
	}
}

func (*questionRepository) ansToDomain(ans dao.AnswerElement) domain.AnswerElement {
	return domain.AnswerElement{
		ID:        ans.ID,
		Content:   ans.Content,
		Keywords:  ans.Keywords,
		Shorthand: ans.Shorthand,
		Highlight: ans.Highlight,
		Guidance:  ans.Guidance,
		Utime:     time.UnixMilli(ans.Utime),
	}
}
