package errs

var (
	SystemError = ErrorCode{Code: 508001, Msg: "系统错误"}
)

type ErrorCode struct {
	Code int
	Msg  string
}
