package errs

var (
	SystemError = ErrorCode{Code: 515001, Msg: "系统错误"}
	// InsufficientCredit 这个不管说是客户端错误还是服务端错误，都有点勉强，所以随便用一个 5
	InsufficientCredit = ErrorCode{Code: 515002, Msg: "积分不足"}
)

type ErrorCode struct {
	Code int
	Msg  string
}
