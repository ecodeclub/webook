package payment

import (
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/service"
)

type Payment = domain.Payment
type Record = domain.PaymentRecord
type Channel = domain.PaymentChannel

var ChannelTypeCredit int64 = domain.ChannelTypeCredit
var ChannelTypeWechat int64 = domain.ChannelTypeWechat

type Service = service.Service
