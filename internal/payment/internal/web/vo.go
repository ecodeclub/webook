package web

type Payment struct {
	SN          string
	OrderID     int64
	OrderSN     string
	TotalAmount int64
	Deadline    int64
	PaidAt      int64
	Status      int64
}

type PayReq struct {
	OrderSN  int64     `json:"order_sn"`
	Channels []Channel `json:"channels"`
}

type Channel struct {
	Type          int64  `json:"type,omitempty"`
	Desc          string `json:"desc,omitempty"`
	Amount        int64  `json:"amount"`
	WechatCodeURL string `json:"wechatCodeURL,omitempty"` // 微信二维码
}
