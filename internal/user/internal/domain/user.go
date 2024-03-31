package domain

type User struct {
	Id       int64
	Avatar   string
	Nickname string
	SN       string
	// 不要使用组合，因为你将来可能还有 DingDingInfo 之类的
	WechatInfo WechatInfo
}
