package web

import (
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/user/internal/errs"
)

var (
	systemErrorResult = ginx.Result{
		Code: errs.SystemError.Code,
		Msg:  errs.SystemError.Msg,
	}
	phoneNotFoundResult = ginx.Result{
		Code: errs.PhoneNotFound.Code,
		Msg:  errs.PhoneNotFound.Msg,
	}
)

func newVerificationErr(err error) ginx.Result {
	errCode := errs.NewVerificationErr(err)
	return ginx.Result{
		Code: errCode.Code,
		Msg:  errCode.Msg,
	}
}
