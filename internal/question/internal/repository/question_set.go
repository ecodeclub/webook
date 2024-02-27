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

type QuestionSetRepository interface {
	Create(ctx context.Context, set domain.QuestionSet) (int64, error)
	UpdateQuestions(ctx context.Context, set domain.QuestionSet) error
	GetByID(ctx context.Context, id int64) (domain.QuestionSet, error)
	GetByIDAndUID(ctx context.Context, id, uid int64) (domain.QuestionSet, error)
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
	qids := make([]int64, len(set.Questions))
	for i := range set.Questions {
		qids[i] = set.Questions[i].Id
	}
	return q.dao.UpdateQuestionsByIDAndUID(ctx, set.Id, set.Uid, qids)
}

func (q *questionSetRepository) GetByID(ctx context.Context, id int64) (domain.QuestionSet, error) {
	set, err := q.dao.GetByID(ctx, id)
	if err != nil {
		return domain.QuestionSet{}, err
	}
	questions, err := q.getDomainQuestions(ctx, id)
	if err != nil {
		return domain.QuestionSet{}, err
	}

	return domain.QuestionSet{
		Id:          set.Id,
		Uid:         set.Uid,
		Title:       set.Title,
		Description: set.Description,
		Questions:   questions,
		Utime:       time.Unix(set.Utime, 0),
	}, nil
}

func (q *questionSetRepository) getDomainQuestions(ctx context.Context, id int64) ([]domain.Question, error) {
	questionDAOs, err := q.dao.GetQuestionsByID(ctx, id)
	if err != nil {
		return nil, err
	}
	questionE2D, _ := copier.NewReflectCopier[dao.Question, domain.Question](
		copier.IgnoreFields("Ctime"),
		copier.ConvertField[int64, time.Time]("Utime", converter.ConverterFunc[int64, time.Time](func(src int64) (time.Time, error) {
			return time.Unix(0, src*int64(time.Millisecond)), nil
		})),
	)
	questions := make([]domain.Question, len(questionDAOs))
	for i, question := range questionDAOs {
		d, _ := questionE2D.Copy(&question)
		questions[i] = *d
	}
	return questions, nil
}

func (q *questionSetRepository) GetByIDAndUID(ctx context.Context, id, uid int64) (domain.QuestionSet, error) {
	set, err := q.dao.GetByIDAndUID(ctx, id, uid)
	if err != nil {
		return domain.QuestionSet{}, err
	}
	questions, err := q.getDomainQuestions(ctx, id)
	if err != nil {
		return domain.QuestionSet{}, err
	}

	return domain.QuestionSet{
		Id:          set.Id,
		Uid:         set.Uid,
		Title:       set.Title,
		Description: set.Description,
		Questions:   questions,
		Utime:       time.Unix(0, set.Utime*int64(time.Millisecond)),
	}, nil
}
