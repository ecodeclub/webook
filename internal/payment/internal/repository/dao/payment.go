package dao

type PaymentDAO interface {
}

type Payment struct {
	Id          int64  `gorm:"primaryKey;autoIncrement;comment:支付自增ID"`
	SN          string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_payment_sn;comment:支付序列号"`
	OrderId     int64  `gorm:"uniqueIndex:uniq_order_id,comment:订单自增ID,冗余允许为NULL"`
	OrderSn     string `gorm:"type:varchar(255);uniqueIndex:uniq_order_sn;comment:订单序列号,冗余允许为NULL"`
	TotalAmount int64  `gorm:"not null;comment:支付总金额, 多种支付方式支付金额的总和"`
	PayDDL      int64  `gorm:"column:pay_ddl;not null;comment:支付截止时间"`
	PaidAt      int64  `gorm:"comment:支付时间"`
	Status      int64  `gorm:"type:tinyint unsigned;not null;default:1;comment:支付状态 1=未支付 2=已支付"`
	Ctime       int64
	Utime       int64
}

type PaymentRecord struct {
	Id        int64 `gorm:"primaryKey;autoIncrement;comment:支付记录自增ID"`
	PaymentId int64 `gorm:"not null;index:idx_payment_id,comment:支付自增ID"`
	// 积分模块会给ID凭证p
	PaymentNO3rd string `gorm:"column:payment_no_3rd;not null;uniqueIndex:uniq_payment_no_3rd;comment:支付单号, 支付渠道的事务ID"`
	Channel      int64  `gorm:"type:tinyint unsigned;not null;default:1;comment:支付渠道 1=积分, 2=微信"`
	Amount       int64  `gorm:"not null;comment:支付金额"`
	PaidAt       int64  `gorm:"comment:支付时间"`
	Status       int64  `gorm:"type:tinyint unsigned;not null;default:1;comment:支付状态 1=未支付 2=已支付"`
	Ctime        int64
	Utime        int64
}
