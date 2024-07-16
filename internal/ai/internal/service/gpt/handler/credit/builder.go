package credit

import (
	"context"
	"errors"
	"fmt"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/ecodeclub/webook/internal/ai/internal/service/gpt/handler"
	"github.com/ecodeclub/webook/internal/credit"
	uuid "github.com/lithammer/shortuuid/v4"
)

type HandlerBuilder struct {
	creditSvc credit.Service
	logRepo   repository.GPTCreditLogRepo
}

func (h *HandlerBuilder) Name() string {
	return "credit"
}

var (
	ErrInsufficientCredit = errors.New("积分不足")
)

func NewHandlerBuilder(creSvc credit.Service, repo repository.GPTCreditLogRepo) *HandlerBuilder {
	return &HandlerBuilder{
		creditSvc: creSvc,
		logRepo:   repo,
	}
}

func (h *HandlerBuilder) Next(next handler.Handler) handler.Handler {
	return handler.HandleFunc(func(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error) {
		cre, err := h.creditSvc.GetCreditsByUID(ctx, req.Uid)
		if err != nil {
			return domain.GPTResponse{}, err
		}
		// 如果剩余的积分不足就返回积分不足
		ok := h.checkCredit(cre)
		if !ok {
			return domain.GPTResponse{}, fmt.Errorf("%w, 余额非正数，无法继续调用，用户 %d",
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
			return domain.GPTResponse{}, err
		}
		err = h.creditSvc.AddCredits(context.Background(), credit.Credit{
			Uid: req.Uid,
			Logs: []credit.CreditLog{
				{
					Key:   uuid.New(),
					Uid:   req.Uid,
					Biz:   "ai-gpt",
					BizId: id,
					Desc:  "ai-gpt服务",
				},
			},
		})
		if err != nil {
			_, _ = h.logRepo.SaveCredit(ctx, domain.GPTCredit{
				Id:     id,
				Status: domain.CreditStatusFailed,
			})
			return domain.GPTResponse{}, err
		} else {
			_, err = h.logRepo.SaveCredit(ctx, domain.GPTCredit{
				Id:     id,
				Status: domain.CreditStatusSuccess,
			})
		}
		return resp, err
	})
}

func (h *HandlerBuilder) newLog(req domain.GPTRequest, resp domain.GPTResponse) domain.GPTCredit {
	return domain.GPTCredit{
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
