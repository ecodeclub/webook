package response

import (
	"context"
	"encoding/json"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler"
)

type Handler struct {
	repo repository.GPTLogRepo
}

func NewHandler(repo repository.GPTLogRepo) *Handler {
	return &Handler{
		repo: repo,
	}
}
func (h *Handler) Name() string {
	return "response"
}

func (h *Handler) Next(next handler.HandleFunc) handler.HandleFunc {
	return func(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error) {
		resp, err := next(ctx, req)
		msgByte, _ := json.Marshal(req.Input)
		msg := string(msgByte)
		if err != nil {
			_, _ = h.repo.SaveLog(ctx, domain.GPTLog{
				Tid:    req.Tid,
				Biz:    req.Biz,
				Uid:    req.Uid,
				Prompt: msg,
				Status: domain.FailLogStatus,
			})
			return domain.GPTResponse{}, err
		}
		_, err = h.repo.SaveLog(ctx, domain.GPTLog{
			Tid:    req.Tid,
			Uid:    req.Uid,
			Biz:    req.Biz,
			Tokens: int64(resp.Tokens),
			Amount: resp.Amount,
			Status: domain.ProcessingStatus,
			Prompt: msg,
			Answer: resp.Answer,
		})
		return resp, err
	}
}
