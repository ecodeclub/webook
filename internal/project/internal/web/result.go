package web

import (
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/project/internal/errs"
)

var (
	systemErrorResult = ginx.Result{
		Code: errs.SystemError.Code,
		Msg:  errs.SystemError.Msg,
	}
)
