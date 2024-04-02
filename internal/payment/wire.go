package payment

import (
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/repository"
	"github.com/ecodeclub/webook/internal/payment/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/payment/internal/service"
	"github.com/ecodeclub/webook/internal/payment/internal/web"
	"github.com/google/wire"
)

type Handler = web.Handler
type Payment = domain.Payment
type Record = domain.PaymentRecord
type Channel = domain.PaymentChannel

var ChannelTypeCredit int64 = domain.ChannelTypeCredit
var ChannelTypeWechat int64 = domain.ChannelTypeWechat

type Service = service.Service

var RepoSet = wire.NewSet(
	dao.NewPaymentGORMDAO,
	repository.NewPaymentRepository,
)

//
// var HandlerSet = wire.NewSet(
//
// 	web.NewHandler)
//
// func InitHandler(db *egorm.Component, paymentSvc payment.Service, productSvc product.Service, creditSvc credit.Service, cache ecache.Cache) *Handler {
// 	wire.Build(HandlerSet,
// 		initLogger,
// 		ioc.InitWechatNativeService,
// 		ioc.InitWechatConfig,
// 		ioc.InitWechatNotifyHandler,
// 	)
// 	return new(Handler)
// }
//
// var ServiceSet = wire.NewSet(
// 	dao.NewPaymentGORMDAO,
// 	repository.NewPaymentRepository,
// 	service.NewService,
// )
//
// var (
// 	once = &sync.Once{}
// 	svc  service.Service
// )
//
// func initService(db *gorm.DB) service.Service {
// 	once.Do(func() {
// 		_ = dao.InitTables(db)
// 		orderDAO := dao.NewPaymentGORMDAO(db)
// 		orderRepository := repository.NewPaymentRepository(orderDAO)
// 		svc = service.NewService(orderRepository)
// 	})
// 	return svc
// }
//
// func initLogger() *elog.Component {
// 	return elog.DefaultLogger
// }
