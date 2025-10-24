package web

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/company/internal/domain"
	"github.com/ecodeclub/webook/internal/company/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc service.CompanyService
}

func NewHandler(svc service.CompanyService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/companies")
	g.POST("/detail", ginx.B[IdReq](h.GetById))
	g.POST("/list", ginx.B[Page](h.List))
}

func (h *Handler) GetById(ctx *ginx.Context, req IdReq) (ginx.Result, error) {
	company, err := h.svc.GetById(ctx, req.Id)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: CompanyVO{
			ID:    company.ID,
			Name:  company.Name,
			Ctime: company.Ctime,
			Utime: company.Utime,
		},
	}, nil
}

func (h *Handler) List(ctx *ginx.Context, req Page) (ginx.Result, error) {
	companies, total, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: ListCompanyResp{
			Total: total,
			List: slice.Map(companies, func(idx int, src domain.Company) CompanyVO {
				return CompanyVO{
					ID:    src.ID,
					Name:  src.Name,
					Ctime: src.Ctime,
					Utime: src.Utime,
				}
			}),
		},
	}, nil
}
