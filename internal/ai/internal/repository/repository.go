package repository

import (
	"context"
	"database/sql"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
)

type GPTLogRepo interface {
	SaveCreditLog(ctx context.Context, GPTDeductLog domain.GPTCreditLog) (int64, error)
	SaveLog(ctx context.Context, gptLog domain.GPTLog) (int64, error)
}

type gptLogRepo struct {
	logDao dao.GPTLogDAO
}

func NewGPTLogRepo(logDao dao.GPTLogDAO) GPTLogRepo {
	return &gptLogRepo{
		logDao: logDao,
	}
}

func (g *gptLogRepo) creditLogToEntity(gptLog domain.GPTCreditLog) dao.GptCreditLog {
	return dao.GptCreditLog{
		Id:     gptLog.Id,
		Tid:    gptLog.Tid,
		Uid:    gptLog.Uid,
		Biz:    gptLog.Biz,
		Tokens: gptLog.Tokens,
		Amount: gptLog.Amount,
		Credit: gptLog.Credit,
		Status: gptLog.Status.ToUint8(),
		Prompt: sql.NullString{
			Valid:  true,
			String: gptLog.Prompt,
		},
		Answer: sql.NullString{
			Valid:  true,
			String: gptLog.Answer,
		},
	}
}

func (g *gptLogRepo) logToEntity(gptLog domain.GPTLog) dao.GptLog {
	return dao.GptLog{
		Id:     gptLog.Id,
		Tid:    gptLog.Tid,
		Uid:    gptLog.Uid,
		Biz:    gptLog.Biz,
		Tokens: gptLog.Tokens,
		Amount: gptLog.Amount,
		Status: gptLog.Status.ToUint8(),
		Prompt: sql.NullString{
			Valid:  true,
			String: gptLog.Prompt,
		},
		Answer: sql.NullString{
			Valid:  true,
			String: gptLog.Answer,
		},
	}
}

func (g *gptLogRepo) SaveCreditLog(ctx context.Context, gptDeductLog domain.GPTCreditLog) (int64, error) {
	logEntity := g.creditLogToEntity(gptDeductLog)
	return g.logDao.SaveCreditLog(ctx, logEntity)
}

func (g *gptLogRepo) SaveLog(ctx context.Context, gptLog domain.GPTLog) (int64, error) {
	logEntity := g.logToEntity(gptLog)
	return g.logDao.SaveLog(ctx, logEntity)
}
