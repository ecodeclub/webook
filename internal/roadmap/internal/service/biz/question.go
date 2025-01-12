package biz

import (
	"context"

	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/roadmap/internal/domain"
)

type QuestionStrategy struct {
	queSvc baguwen.Service
}

func NewQuestionStrategy(queSvc baguwen.Service) Strategy {
	return &QuestionStrategy{
		queSvc: queSvc,
	}
}

func (q *QuestionStrategy) GetBizsByIds(ctx context.Context, ids []int64) (map[int64]domain.Biz, error) {
	ques, err := q.queSvc.GetPubByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	res := make(map[int64]domain.Biz, len(ques))
	for _, que := range ques {
		res[que.Id] = domain.Biz{
			Biz:   domain.BizQuestion,
			BizId: que.Id,
			Title: que.Title,
		}
	}
	return res, nil
}
