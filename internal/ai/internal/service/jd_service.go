package service

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm"
	"github.com/lithammer/shortuuid/v4"
	"golang.org/x/sync/errgroup"
)

type JDService interface {
	// Evaluate 测评
	Evaluate(ctx context.Context, uid int64, jd string) (domain.JD, error)
}

type jdSvc struct {
	aiSvc llm.Service
}

func NewJDService(aiSvc llm.Service) JDService {
	return &jdSvc{
		aiSvc: aiSvc,
	}
}

func (j *jdSvc) Evaluate(ctx context.Context, uid int64, jd string) (domain.JD, error) {
	var techJD, bizJD, positionJD *domain.JDEvaluation
	var amount int64
	var eg errgroup.Group
	eg.Go(func() error {
		var err error
		var techAmount int64
		techAmount, techJD, err = j.analysisJd(ctx, uid, domain.AnalysisJDTech, jd)
		if err != nil {
			return err
		}
		atomic.AddInt64(&amount, techAmount)
		return nil
	})
	eg.Go(func() error {
		var err error
		var bizAmount int64
		bizAmount, bizJD, err = j.analysisJd(ctx, uid, domain.AnalysisJDBiz, jd)
		if err != nil {
			return err
		}
		atomic.AddInt64(&amount, bizAmount)
		return nil
	})
	eg.Go(func() error {
		var err error
		var positionAmount int64
		positionAmount, positionJD, err = j.analysisJd(ctx, uid, domain.AnalysisJDPosition, jd)
		if err != nil {
			return err
		}
		atomic.AddInt64(&amount, positionAmount)
		return nil
	})
	if err := eg.Wait(); err != nil {
		return domain.JD{}, err
	}
	return domain.JD{
		Amount:    amount,
		TechScore: techJD,
		BizScore:  bizJD,
		PosScore:  positionJD,
	}, nil
}

func (j *jdSvc) analysisJd(ctx context.Context, uid int64, biz string, jd string) (int64, *domain.JDEvaluation, error) {
	tid := shortuuid.New()
	aiReq := domain.LLMRequest{
		Uid:   uid,
		Tid:   tid,
		Biz:   biz,
		Input: []string{jd},
	}
	resp, err := j.aiSvc.Invoke(ctx, aiReq)
	if err != nil {
		return 0, nil, err
	}
	var jdEva domain.JDEvaluation
	err = json.Unmarshal([]byte(resp.Answer), &jdEva)
	if err != nil {
		return 0, nil, err
	}
	return resp.Amount, &jdEva, nil
}
