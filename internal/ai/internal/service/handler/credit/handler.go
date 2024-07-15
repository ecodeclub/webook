package credit

import (
	"context"
	"encoding/json"
	"errors"
	"math"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler"
	"github.com/ecodeclub/webook/internal/credit"
	uuid "github.com/lithammer/shortuuid/v4"
)

type Handler struct {
	creditSvc credit.Service
	logRepo   repository.GPTLogRepo
}

func (h *Handler) Name() string {
	return "credit"
}

var (
	ErrInsufficientCredit = errors.New("积分不足")
)

func NewHandler(creSvc credit.Service, repo repository.GPTLogRepo) *Handler {
	return &Handler{
		creditSvc: creSvc,
		logRepo:   repo,
	}
}

func (h *Handler) Next(next handler.HandleFunc) handler.HandleFunc {
	return func(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error) {
		bizConfig := req.BizConfig
		cre, err := h.creditSvc.GetCreditsByUID(ctx, req.Uid)
		if err != nil {
			return domain.GPTResponse{}, err
		}
		// 如果剩余的积分不足就返回积分不足
		err = h.checkCredit(cre, bizConfig)
		if err != nil {
			return domain.GPTResponse{}, err
		}

		// 调用下层服务
		resp, err := next(ctx, req)
		if err != nil {
			return resp, err
		}

		// 扣款
		needCredit := h.roundUp(float64(resp.Tokens) * bizConfig.CreditPerToken)
		needAmount := h.roundUp(float64(resp.Tokens) * bizConfig.AmountPerToken)
		resp.Amount = int64(needAmount)
		id, err := h.logRepo.SaveCreditLog(ctx, h.convertToDomain(needCredit, needAmount, req, resp))
		if err != nil {
			return domain.GPTResponse{}, err
		}
		err = h.creditSvc.AddCredits(context.Background(), credit.Credit{
			Uid: req.Uid,
			Logs: []credit.CreditLog{
				{
					Key:          uuid.New(),
					Uid:          req.Uid,
					ChangeAmount: int64(-1 * needCredit),
					Biz:          "ai-gpt",
					BizId:        id,
					Desc:         "ai-gpt服务",
				},
			},
		})
		if err != nil {
			_, _ = h.logRepo.SaveCreditLog(ctx, domain.GPTCreditLog{
				Id:     id,
				Status: domain.FailLogStatus,
			})
			return domain.GPTResponse{}, err
		} else {
			_, err = h.logRepo.SaveCreditLog(ctx, domain.GPTCreditLog{
				Id:     id,
				Status: domain.SuccessStatus,
			})
		}
		return resp, err
	}
}

func (h *Handler) convertToDomain(needCredit, needAmount int, req domain.GPTRequest, resp domain.GPTResponse) domain.GPTCreditLog {
	prompt, _ := json.Marshal(req.Input)
	return domain.GPTCreditLog{
		Tid:    req.Tid,
		Uid:    req.Uid,
		Biz:    req.Biz,
		Tokens: int64(resp.Tokens),
		Amount: int64(needAmount),
		Credit: int64(needCredit),
		Status: domain.ProcessingStatus,
		Prompt: string(prompt),
		Answer: resp.Answer,
	}
}

func (h *Handler) checkCredit(cre credit.Credit, bizConfig domain.GPTBiz) error {
	// 判断积分是否满足
	wantCre := h.roundUp(float64(bizConfig.MaxTokensPerTime) * bizConfig.CreditPerToken)
	if wantCre > int(cre.TotalAmount) {
		return ErrInsufficientCredit
	}
	return nil
}

// 向上取整
func (h *Handler) roundUp(val float64) int {
	return int(math.Ceil(val))
}
