package web

import (
	"encoding/json"
	"sync/atomic"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm"
	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid/v4"
	"golang.org/x/sync/errgroup"
)

type Handler struct {
	llmSvc llm.Service
}

func NewHandler(llmSvc llm.Service) *Handler {
	return &Handler{
		llmSvc: llmSvc,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	server.POST("/ai/ask", ginx.BS(h.LLMAsk))
	server.POST("/ai/analysis_jd", ginx.BS(h.AnalysisJd))
}

func (h *Handler) LLMAsk(ctx *ginx.Context, req LLMRequest, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	tid := shortuuid.New()
	aiReq := domain.LLMRequest{
		Uid:   uid,
		Tid:   tid,
		Biz:   req.Biz,
		Input: req.Input,
	}
	resp, err := h.llmSvc.Invoke(ctx, aiReq)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: LLMResponse{
			Amount:    resp.Amount,
			RawResult: resp.Answer,
		},
	}, nil
}

func (h *Handler) AnalysisJd(ctx *ginx.Context, req JDRequest, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	var techJD, bizJD, positionJD *JD
	var amount int64
	var eg errgroup.Group
	eg.Go(func() error {
		var err error
		var techAmount int64
		techAmount, techJD, err = h.analysisJd(ctx, uid, domain.AnalysisJDTech, req)
		if err != nil {
			return err
		}
		atomic.AddInt64(&amount, techAmount)
		return nil
	})
	eg.Go(func() error {
		var err error
		var bizAmount int64
		bizAmount, bizJD, err = h.analysisJd(ctx, uid, domain.AnalysisJDBiz, req)
		if err != nil {
			return err
		}
		atomic.AddInt64(&amount, bizAmount)
		return nil
	})
	eg.Go(func() error {
		var err error
		var positionAmount int64
		positionAmount, positionJD, err = h.analysisJd(ctx, uid, domain.AnalysisJDPosition, req)
		if err != nil {
			return err
		}
		atomic.AddInt64(&amount, positionAmount)
		return nil
	})
	if err := eg.Wait(); err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: JDResponse{
			Amount:    amount,
			TechScore: techJD,
			BizScore:  bizJD,
			PosScore:  positionJD,
		},
	}, nil
}

func (h *Handler) analysisJd(ctx *ginx.Context, uid int64, biz string, req JDRequest) (int64, *JD, error) {
	tid := shortuuid.New()
	aiReq := domain.LLMRequest{
		Uid:   uid,
		Tid:   tid,
		Biz:   biz,
		Input: req.Input,
	}
	resp, err := h.llmSvc.Invoke(ctx, aiReq)
	if err != nil {
		return 0, nil, err
	}
	var jd JD
	err = json.Unmarshal([]byte(resp.Answer), &jd)
	if err != nil {
		return 0, nil, err
	}
	return resp.Amount, &jd, nil
}
