package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm"
	"github.com/lithammer/shortuuid/v4"
)

type GeneralService interface {
	// LLMAsk 通用询问ai的接口
	LLMAsk(ctx context.Context, uid int64, biz string, input []string) (domain.LLMResponse, error)
	Stream(ctx context.Context, uid int64, biz string, input []string) (chan domain.StreamEvent, error)
}

func NewGeneralService(aiSvc llm.Service) GeneralService {
	return &generalSvc{
		aiSvc: aiSvc,
	}
}

type generalSvc struct {
	aiSvc llm.Service
}

func (g *generalSvc) Stream(ctx context.Context, uid int64, biz string, input []string) (chan domain.StreamEvent, error) {
	tid := shortuuid.New()
	aiReq := domain.LLMRequest{
		Uid:   uid,
		Tid:   tid,
		Biz:   biz,
		Input: input,
	}
	return g.aiSvc.Stream(ctx, aiReq)
}

func (g *generalSvc) LLMAsk(ctx context.Context, uid int64, biz string, input []string) (domain.LLMResponse, error) {
	tid := shortuuid.New()
	aiReq := domain.LLMRequest{
		Uid:   uid,
		Tid:   tid,
		Biz:   biz,
		Input: input,
	}
	return g.aiSvc.Invoke(ctx, aiReq)
}
