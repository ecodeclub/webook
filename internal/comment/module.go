package comment

import "github.com/ecodeclub/webook/internal/comment/internal/web"

type Module struct {
	Hdl *Handler
}
type Handler = web.Handler
