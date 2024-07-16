package repository

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
)

type GPTCreditLogRepo interface {
	SaveCredit(ctx context.Context, GPTDeductLog domain.GPTCredit) (int64, error)
}

type gptCreditLogRepo struct {
	logDao dao.GPTCreditDAO
}

func NewGPTCreditLogRepo(logDao dao.GPTCreditDAO) GPTCreditLogRepo {
	return &gptCreditLogRepo{
		logDao: logDao,
	}
}

func (g *gptCreditLogRepo) creditLogToEntity(gptLog domain.GPTCredit) dao.GPTCredit {
	return dao.GPTCredit{
		Id:     gptLog.Id,
		Tid:    gptLog.Tid,
		Uid:    gptLog.Uid,
		Biz:    gptLog.Biz,
		Amount: gptLog.Amount,
		Status: gptLog.Status.ToUint8(),
	}
}

func (g *gptCreditLogRepo) SaveCredit(ctx context.Context, gptDeductLog domain.GPTCredit) (int64, error) {
	logEntity := g.creditLogToEntity(gptDeductLog)
	return g.logDao.SaveCredit(ctx, logEntity)
}
