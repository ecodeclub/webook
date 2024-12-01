package credit

import (
	"context"
	"errors"
	"fmt"

	"github.com/gotomicro/ego/core/elog"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler"
	"github.com/ecodeclub/webook/internal/credit"
	uuid "github.com/lithammer/shortuuid/v4"
)

type HandlerBuilder struct {
	creditSvc credit.Service
	logRepo   repository.LLMCreditLogRepo
	logger    *elog.Component
}

func (h *HandlerBuilder) Name() string {
	return "credit"
}

var (
	ErrInsufficientCredit = errors.New("积分不足")
)

func NewHandlerBuilder(creSvc credit.Service, repo repository.LLMCreditLogRepo) *HandlerBuilder {
	return &HandlerBuilder{
		creditSvc: creSvc,
		logRepo:   repo,
		logger:    elog.DefaultLogger,
	}
}

func (h *HandlerBuilder) Next(next handler.Handler) handler.Handler {
	return handler.HandleFunc(func(ctx context.Context, req domain.LLMRequest) (domain.LLMResponse, error) {
		// 不需要扣除积分
		if req.Config.Price == 0 {
			return next.Handle(ctx, req)
		}
		cre, err := h.creditSvc.GetCreditsByUID(ctx, req.Uid)
		if err != nil {
			return domain.LLMResponse{}, err
		}
		// 如果剩余的积分不足就返回积分不足
		ok := h.checkCredit(cre)
		if !ok {
			return domain.LLMResponse{}, fmt.Errorf("%w, 余额非正数，无法继续调用，用户 %d",
				ErrInsufficientCredit, req.Uid)
		}

		// 调用下层服务
		resp, err := next.Handle(ctx, req)
		if err != nil {
			return resp, err
		}

		// 扣款
		id, err := h.logRepo.SaveCredit(ctx, h.newLog(req, resp))
		if err != nil {
			return domain.LLMResponse{}, err
		}
		err = h.deductCredit(ctx, credit.Credit{
			Uid: req.Uid,
			Logs: []credit.CreditLog{
				{
					Key:          uuid.New(),
					ChangeAmount: resp.Amount,
					Uid:          req.Uid,
					Biz:          "ai-llm",
					BizId:        id,
					Desc:         "ai-llm服务",
				},
			},
		})
		if err != nil {
			_, _ = h.logRepo.SaveCredit(ctx, domain.LLMCredit{
				Id:     id,
				Status: domain.CreditStatusFailed,
			})
			return domain.LLMResponse{}, err
		}

		_, err = h.logRepo.SaveCredit(ctx, domain.LLMCredit{
			Id:     id,
			Status: domain.CreditStatusSuccess,
		})

		return resp, err
	})
}

// TODO deductCredit 后面要求 credit 那边提供一个一次性接口，绕开 try-confirm 流程
func (h *HandlerBuilder) deductCredit(ctx context.Context, c credit.Credit) error {
	id, err := h.creditSvc.TryDeductCredits(ctx, c)
	if err != nil {
		return err
	}
	err = h.creditSvc.ConfirmDeductCredits(ctx, c.Uid, id)
	if err != nil {
		err1 := h.creditSvc.CancelDeductCredits(ctx, c.Uid, id)
		if err1 != nil {
			h.logger.Error("确认扣减积分失败之后试图回滚，也失败了", elog.FieldErr(err1))
		}
		err = fmt.Errorf("确认扣减积分失败 %w", err)
	}
	return err
}

func (h *HandlerBuilder) newLog(req domain.LLMRequest, resp domain.LLMResponse) domain.LLMCredit {
	return domain.LLMCredit{
		Tid:    req.Tid,
		Uid:    req.Uid,
		Biz:    req.Biz,
		Tokens: resp.Tokens,
		Amount: resp.Amount,
		Status: domain.CreditStatusProcessing,
	}
}

func (h *HandlerBuilder) checkCredit(cre credit.Credit) bool {
	// 判断积分是否满足
	// 并不能用一次调用的最大 token 数量来算，因为要考虑用户可能最后只剩下一点点钱了，
	// 这点钱不够最大数量，但是够一次普通调用
	return cre.TotalAmount > 0
}
