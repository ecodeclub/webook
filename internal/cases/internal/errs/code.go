package errs

var (
	SystemError = ErrorCode{Code: 505001, Msg: "系统错误"}
	// InsufficientCredits 这个不管说是客户端错误还是服务端错误，都有点勉强，所以随便用一个 5
	InsufficientCredits = ErrorCode{Code: 505002, Msg: "积分不足"}
)

type ErrorCode struct {
	Code int
	Msg  string
}
