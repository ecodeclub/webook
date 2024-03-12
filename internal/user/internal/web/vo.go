package web

type Profile struct {
	Id        int64  `json:"id,omitempty"`
	Nickname  string `json:"nickname,omitempty"`
	Avatar    string `json:"avatar,omitempty"`
	IsCreator bool   `json:"isCreator,omitempty"`
}

type WechatCallback struct {
	Code  string `json:"code"`
	State string `json:"state"`
}
