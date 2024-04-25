package web

import (
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/search/internal/errs"
)

var (
	systemErrorResult = ginx.Result{
		Code: errs.SystemError.Code,
		Msg:  errs.SystemError.Msg,
	}
)
