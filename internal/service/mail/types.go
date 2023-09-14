package mail

import "context"

// Service 发送邮件的抽象
// 目前你可以理解为，这是一个为了适配不同的发送邮件的抽象
type Service interface {
	// Send 考虑暂时没有附件需求，暂时不定义。
	// 为了适配AWS SES内容类型也需要传入。
	Send(ctx context.Context, from, subject, body, to string, isHTML bool) error
}
