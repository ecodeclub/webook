package web

var (
//	systemErrorResult = ginx.Result{
//		Code: errs.SystemError.Code,
//		Msg:  errs.SystemError.Msg,
//	}
)

// Outbox 发件箱
type Outbox struct {
	Id int64
	// 发件人
	Uid     int64
	Content string
	// 其它字段
}

type Inbox struct {
	Id int64
	// 收件人
	Uid     int64
	Content string
	// 其它字段
}
