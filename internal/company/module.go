package company

import (
	"github.com/ecodeclub/webook/internal/company/internal/domain"
	"github.com/ecodeclub/webook/internal/company/internal/service"
	"github.com/ecodeclub/webook/internal/company/internal/web"
)

type (
	AdminHandler = web.CompanyHandler
	Handler      = web.Handler
	Service      = service.CompanyService
	Company      = domain.Company
)

type Module struct {
	AdminHdl *AdminHandler
	Hdl      *Handler
	Svc      Service
}
