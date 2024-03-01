package web

type Profile struct {
	Id       int64  `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type WechatCallback struct {
	Code  string `json:"code"`
	State string `json:"state"`
}
