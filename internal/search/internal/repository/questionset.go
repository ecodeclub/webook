package repository

import (
	"context"
	"time"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/ecodeclub/webook/internal/search/internal/repository/dao"
)

type questionSetRepo struct {
	qsDao dao.QuestionSetDAO
}

func NewQuestionSetRepo(questionSetDao dao.QuestionSetDAO) QuestionSetRepo {
	return &questionSetRepo{
		qsDao: questionSetDao,
	}
}

func (q *questionSetRepo) InputQuestionSet(ctx context.Context, msg domain.QuestionSet) error {
	return q.qsDao.InputQuestionSet(ctx, q.toEntity(msg))
}

func (q *questionSetRepo) SearchQuestionSet(ctx context.Context, ids []int64, keywords []string) ([]domain.QuestionSet, error) {
	sets, err := q.qsDao.SearchQuestionSet(ctx, ids, keywords)
	if err != nil {
		return nil, err
	}
	ans := make([]domain.QuestionSet, 0, len(sets))
	for _, set := range sets {
		ans = append(ans, q.toDomain(set))
	}

	return ans, nil
}

func (*questionSetRepo) toEntity(qs domain.QuestionSet) dao.QuestionSet {
	return dao.QuestionSet{
		Id:          qs.Id,
		Uid:         qs.Uid,
		Title:       qs.Title,
		Description: qs.Description,
		Questions:   qs.Questions,
		Utime:       qs.Utime.UnixMilli(),
	}
}

func (*questionSetRepo) toDomain(qs dao.QuestionSet) domain.QuestionSet {
	return domain.QuestionSet{
		Id:          qs.Id,
		Uid:         qs.Uid,
		Title:       qs.Title,
		Description: qs.Description,
		Questions:   qs.Questions,
		Utime:       time.UnixMilli(qs.Utime),
	}
}
