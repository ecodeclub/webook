package biz

import (
	"context"

	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/roadmap/internal/domain"
)

type QuestionSetStrategy struct {
	queSetSvc baguwen.QuestionSetService
}

func NewQuestionSetStrategy(queSvc baguwen.QuestionSetService) Strategy {
	return &QuestionSetStrategy{
		queSetSvc: queSvc,
	}
}
func (q *QuestionSetStrategy) GetBizsByIds(ctx context.Context, ids []int64) (map[int64]domain.Biz, error) {
	qs, err := q.queSetSvc.GetByIds(ctx, ids)
	if err != nil {
		return nil, err
	}
	res := make(map[int64]domain.Biz, len(qs))
	for _, q := range qs {
		res[q.Id] = domain.Biz{
			Biz:   domain.BizQuestionSet,
			BizId: q.Id,
			Title: q.Title,
		}
	}
	return res, nil
}
