package domain

type WechatInfo struct {
	// OpenId 是应用内唯一
	OpenId string
	// UnionId 是整个公司账号内唯一,同一公司账号下的多个应用之间均相同
	UnionId string
}
