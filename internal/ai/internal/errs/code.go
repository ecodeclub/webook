package errs

var (
	SystemError        = ErrorCode{Code: 516001, Msg: "系统错误"}
	InsufficientCredit = ErrorCode{Code: 516002, Msg: "积分不足"}
)

type ErrorCode struct {
	Code int
	Msg  string
}
