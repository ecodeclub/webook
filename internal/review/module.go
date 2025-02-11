package review

import "github.com/ecodeclub/webook/internal/review/internal/web"

type Module struct {
	Hdl      *Hdl
	AdminHdl *AdminHdl
}
type AdminHdl = web.AdminHandler
type Hdl = web.Handler
