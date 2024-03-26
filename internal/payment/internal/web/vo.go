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

type Channel struct {
	Type          int64
	Desc          string
	Amount        int64
	WechatCodeURL string // 微信二维码
}
