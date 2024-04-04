package web

import "github.com/ecodeclub/webook/internal/user/internal/domain"

type Profile struct {
	Id        int64  `json:"id,omitempty"`
	Nickname  string `json:"nickname,omitempty"`
	Avatar    string `json:"avatar,omitempty"`
	SN        string `json:"sn,omitempty"`
	IsCreator bool   `json:"isCreator,omitempty"`
	MemberDDL string `json:"memberDDL,omitempty"`
}

func newProfile(u domain.User) Profile {
	return Profile{
		Nickname: u.Nickname,
		Avatar:   u.Avatar,
		SN:       u.SN,
	}
}

type WechatCallback struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type EditReq struct {
	Avatar   string `json:"avatar"`
	Nickname string `json:"nickname"`
}
