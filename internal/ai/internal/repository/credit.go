package repository

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
)

type LLMCreditLogRepo interface {
	SaveCredit(ctx context.Context, l domain.LLMCredit) (int64, error)
}

type llmCreditLogRepo struct {
	logDao dao.LLMCreditDAO
}

func NewLLMCreditLogRepo(logDao dao.LLMCreditDAO) LLMCreditLogRepo {
	return &llmCreditLogRepo{
		logDao: logDao,
	}
}

func (g *llmCreditLogRepo) creditLogToEntity(l domain.LLMCredit) dao.LLMCredit {
	return dao.LLMCredit{
		Id:     l.Id,
		Tid:    l.Tid,
		Uid:    l.Uid,
		Biz:    l.Biz,
		Amount: l.Amount,
		Status: l.Status.ToUint8(),
	}
}

func (g *llmCreditLogRepo) SaveCredit(ctx context.Context, l domain.LLMCredit) (int64, error) {
	logEntity := g.creditLogToEntity(l)
	return g.logDao.SaveCredit(ctx, logEntity)
}
