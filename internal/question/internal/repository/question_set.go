package repository

import (
	"context"
	"time"

	"github.com/ecodeclub/ekit/bean/copier"
	"github.com/ecodeclub/ekit/bean/copier/converter"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/repository/dao"
	"github.com/gotomicro/ego/core/elog"
)

var (
	ErrDuplicatedQuestionID = dao.ErrDuplicatedQuestionID
)

type QuestionSetRepository interface {
	Create(ctx context.Context, set domain.QuestionSet) (int64, error)

	UpdateQuestions(ctx context.Context, set domain.QuestionSet) error
	AddQuestions(ctx context.Context, set domain.QuestionSet) error
	DeleteQuestions(ctx context.Context, set domain.QuestionSet) error
}

type questionSetRepository struct {
	dao            dao.QuestionSetDAO
	questionSetD2E copier.Copier[domain.QuestionSet, dao.QuestionSet]
	questionD2E    copier.Copier[domain.Question, dao.Question]
	logger         *elog.Component
}

func NewQuestionSetRepository(d dao.QuestionSetDAO) QuestionSetRepository {
	fieldConverter := converter.ConverterFunc[time.Time, int64](func(src time.Time) (int64, error) {
		return src.UnixMilli(), nil
	})
	questionSetD2E, err := copier.NewReflectCopier[domain.QuestionSet, dao.QuestionSet](
		copier.IgnoreFields("Questions"),
		copier.ConvertField[time.Time, int64]("Utime", fieldConverter),
	)
	if err != nil {
		panic(err)
	}
	questionD2E, err := copier.NewReflectCopier[domain.Question, dao.Question](
		copier.ConvertField[time.Time, int64]("Utime", fieldConverter),
	)

	if err != nil {
		panic(err)
	}
	return &questionSetRepository{
		dao:            d,
		questionSetD2E: questionSetD2E,
		questionD2E:    questionD2E,
		logger:         elog.DefaultLogger}
}

func (q *questionSetRepository) Create(ctx context.Context, set domain.QuestionSet) (int64, error) {
	d, err := q.questionSetD2E.Copy(&set)
	if err != nil {
		return 0, err
	}
	return q.dao.Create(ctx, *d)
}

func (q *questionSetRepository) UpdateQuestions(ctx context.Context, set domain.QuestionSet) error {
	questions := make([]dao.Question, len(set.Questions))
	for i := range set.Questions {
		d, _ := q.questionD2E.Copy(&set.Questions[i])
		questions[i] = *d
	}
	return q.dao.UpdateQuestionsByID(ctx, set.Id, questions)
}

func (q *questionSetRepository) AddQuestions(ctx context.Context, set domain.QuestionSet) error {
	questions := make([]dao.Question, len(set.Questions))
	for i := range set.Questions {
		d, _ := q.questionD2E.Copy(&set.Questions[i])
		questions[i] = *d
	}
	return q.dao.AddQuestionsByID(ctx, set.Id, questions)
}

func (q *questionSetRepository) DeleteQuestions(ctx context.Context, set domain.QuestionSet) error {
	questions := make([]dao.Question, len(set.Questions))
	for i := range set.Questions {
		d, _ := q.questionD2E.Copy(&set.Questions[i])
		questions[i] = *d
	}
	return q.dao.DeleteQuestionsByID(ctx, set.Id, questions)
}
