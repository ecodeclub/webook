package errs

var (
	SystemError   = ErrorCode{Code: 501001, Msg: "系统错误"}
	PhoneNotFound = ErrorCode{
		Code: 501003,
		Msg:  "手机号不存在",
	}
)

type ErrorCode struct {
	Code int
	Msg  string
}

func NewVerificationErr(err error) ErrorCode {
	return ErrorCode{Code: 501002, Msg: err.Error()}
}
