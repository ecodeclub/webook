package web

import (
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/interview/internal/service"
	"github.com/gin-gonic/gin"
)

type OfferHandler struct {
	svc service.OfferService
}

func NewOfferHandler(svc service.OfferService) *OfferHandler {
	return &OfferHandler{svc: svc}
}

func (h *OfferHandler) PrivateRoutes(_ *gin.Engine) {}

func (h *OfferHandler) MemberRoutes(server *gin.Engine) {
	server.POST("/offer/send", ginx.B[OfferSendRequest](h.Send))
}

// OfferSendRequest 发送 Offer 的请求体
type OfferSendRequest struct {
	Email       string `json:"email"`
	CompanyName string `json:"companyName"`
	JobName     string `json:"jobName"`
	Salary      string `json:"salary"`
	EntryTime   int64  `json:"entryTime"`
}

// Send 发送 Offer 到指定邮箱
// POST /offer/send
func (h *OfferHandler) Send(ctx *ginx.Context, req OfferSendRequest) (ginx.Result, error) {
	err := h.svc.Send(ctx, service.OfferSendReq{
		ToEmail:     req.Email,
		CompanyName: req.CompanyName,
		JobName:     req.JobName,
		Salary:      req.Salary,
		EntryTime:   req.EntryTime,
	})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{Msg: "OK"}, nil
}
