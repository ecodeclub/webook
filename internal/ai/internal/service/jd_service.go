package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
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
	var subtext string
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

	eg.Go(func() error {
		tid := shortuuid.New()
		resp, err := j.aiSvc.Invoke(ctx, domain.LLMRequest{
			Uid:   uid,
			Tid:   tid,
			Biz:   domain.AnalysisJDSubtext,
			Input: []string{jd},
		})
		subtext = resp.Answer
		atomic.AddInt64(&amount, resp.Amount)
		return err
	})
	if err := eg.Wait(); err != nil {
		return domain.JD{}, err
	}
	return domain.JD{
		Amount:    amount,
		TechScore: techJD,
		BizScore:  bizJD,
		PosScore:  positionJD,
		Subtext:   subtext,
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
	answer := strings.SplitN(resp.Answer, "\n", 2)
	if len(answer) != 2 {
		return 0, nil, errors.New("不符合预期的大模型响应")
	}
	score := answer[0]
	scoreNum, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(score, "score:")), 64)
	if err != nil {
		return 0, nil, fmt.Errorf("分数返回的数据不对 %s", score)
	}

	return resp.Amount, &domain.JDEvaluation{
		Score:    scoreNum,
		Analysis: strings.TrimSpace(strings.TrimPrefix(answer[1], "analysis:")),
	}, nil
}
