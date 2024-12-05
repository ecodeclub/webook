package service

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/gotomicro/ego/core/elog"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm"
	"github.com/lithammer/shortuuid/v4"
	"golang.org/x/sync/errgroup"
)

// 最简单的提取方式
const jsonExpr = `\{(.|\n|\r)+\}`

type JDService interface {
	// Evaluate 测评
	Evaluate(ctx context.Context, uid int64, jd string) (domain.JD, error)
}

type jdSvc struct {
	aiSvc  llm.Service
	logger *elog.Component
	expr   *regexp.Regexp
}

func NewJDService(aiSvc llm.Service) JDService {
	return &jdSvc{
		aiSvc:  aiSvc,
		logger: elog.DefaultLogger,
		expr:   regexp.MustCompile(jsonExpr),
	}
}

func (j *jdSvc) Evaluate(ctx context.Context, uid int64, jd string) (domain.JD, error) {
	var techJD, bizJD, positionJD domain.JDEvaluation
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

func (j *jdSvc) analysisJd(ctx context.Context, uid int64, biz string, jd string) (int64, domain.JDEvaluation, error) {
	tid := shortuuid.New()
	aiReq := domain.LLMRequest{
		Uid:   uid,
		Tid:   tid,
		Biz:   biz,
		Input: []string{jd},
	}
	resp, err := j.aiSvc.Invoke(ctx, aiReq)
	if err != nil {
		return 0, domain.JDEvaluation{}, err
	}
	jsonStr := j.expr.FindString(resp.Answer)
	var (
		scoreResp ScoreResp
		analysis  string
	)
	err = json.Unmarshal([]byte(jsonStr), &scoreResp)
	if err != nil {
		j.logger.Error("不符合预期的大模型响应",
			elog.FieldErr(err),
			elog.String("resp", resp.Answer))
	} else {
		analysis = "- " + strings.Join(scoreResp.Summary, "\n- ")
	}
	return resp.Amount, domain.JDEvaluation{
		Score: scoreResp.Score,
		// 按照 Markdown 的写法，拼接起来
		Analysis: analysis,
	}, nil
}

type ScoreResp struct {
	Score   float64  `json:"score"`
	Summary []string `json:"summary"`
}
