package errs

var (
	SystemError = ErrorCode{Code: 503001, Msg: "系统错误"}
)

type ErrorCode struct {
	Code int
	Msg  string
}
