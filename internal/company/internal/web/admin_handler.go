package web

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/company/internal/domain"
	"github.com/ecodeclub/webook/internal/company/internal/service"
	"github.com/gin-gonic/gin"
)

type CompanyHandler struct {
	svc service.CompanyService
}

func NewCompanyHandler(svc service.CompanyService) *CompanyHandler {
	return &CompanyHandler{
		svc: svc,
	}
}

// PrivateRoutes 按照 interactive 的风格注册路由
func (h *CompanyHandler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/companies")
	g.POST("/save", ginx.BS[SaveCompanyReq](h.Save))
	g.POST("/detail", ginx.B[IdReq](h.GetById))
	g.POST("/list", ginx.B[Page](h.List))
	g.POST("/delete", ginx.B[IdReq](h.Delete))
}

func (h *CompanyHandler) Save(ctx *ginx.Context, req SaveCompanyReq, _ session.Session) (ginx.Result, error) {
	resultId, err := h.svc.Save(ctx, domain.Company{
		ID:   req.ID,
		Name: req.Name,
	})
	if err != nil {
		return systemErrorResult, err
	}

	return ginx.Result{
		Data: resultId,
	}, nil
}

func (h *CompanyHandler) GetById(ctx *ginx.Context, req IdReq) (ginx.Result, error) {
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

func (h *CompanyHandler) List(ctx *ginx.Context, req Page) (ginx.Result, error) {
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

func (h *CompanyHandler) Delete(ctx *ginx.Context, req IdReq) (ginx.Result, error) {
	err := h.svc.Delete(ctx, req.Id)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Msg: "删除成功",
	}, nil
}
