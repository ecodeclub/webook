package event

const PaymentEventName = "payment_events"

// PaymentEvent 也是最简设计
// 有一些人会习惯把支付详情也放进来，但是目前来看是没有必要的
// 后续如果要接入大数据之类的，那么就可以考虑提供 payment 详情
type PaymentEvent struct {
	OrderSN string `json:"orderSN"`
	PayerID int64  `json:"payerID"`
	Status  uint8  `json:"status"`
}
