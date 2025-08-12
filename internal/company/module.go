package company

import (
	"github.com/ecodeclub/webook/internal/company/internal/service"
	"github.com/ecodeclub/webook/internal/company/internal/web"
)

type (
	Handler = web.CompanyHandler
	Service = service.CompanyService
)

type Module struct {
	Hdl *Handler
	Svc Service
}
