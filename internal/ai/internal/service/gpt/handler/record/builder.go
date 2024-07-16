package record

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/service/gpt/handler"
	"github.com/gotomicro/ego/core/elog"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository"
)

type HandlerBuilder struct {
	repo   repository.GPTLogRepo
	logger *elog.Component
}

func NewHandler(repo repository.GPTLogRepo) *HandlerBuilder {
	return &HandlerBuilder{
		repo:   repo,
		logger: elog.DefaultLogger,
	}
}
func (h *HandlerBuilder) Name() string {
	return "response"
}

func (h *HandlerBuilder) Next(next handler.Handler) handler.Handler {
	return handler.HandleFunc(func(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error) {
		log := domain.GPTRecord{
			Tid:            req.Tid,
			Biz:            req.Biz,
			Uid:            req.Uid,
			Input:          req.Input,
			KnowledgeId:    req.Config.KnowledgeId,
			PromptTemplate: req.Config.PromptTemplate,
		}
		defer func() {
			_, err1 := h.repo.SaveLog(ctx, log)
			if err1 != nil {
				h.logger.Error("保存 GPT 访问记录失败", elog.FieldErr(err1))
			}
		}()
		resp, err := next.Handle(ctx, req)
		if err != nil {
			log.Status = domain.RecordStatusFailed
			return domain.GPTResponse{}, err
		}
		log.Tokens = resp.Tokens
		log.Amount = resp.Amount
		log.Status = domain.RecordStatusProcessing
		log.Answer = resp.Answer
		return resp, err
	})
}
