// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package startup

import (
	"sync"

	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/payment"
	"github.com/ecodeclub/webook/internal/payment/internal/event"
	"github.com/ecodeclub/webook/internal/payment/internal/repository"
	"github.com/ecodeclub/webook/internal/payment/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/payment/internal/service"
	"github.com/ecodeclub/webook/internal/payment/internal/service/wechat"
	"github.com/ecodeclub/webook/internal/payment/ioc"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ecodeclub/webook/internal/user"
	"github.com/google/wire"
	"gorm.io/gorm"
)

// Injectors from wire.go:

func InitService(p event.PaymentEventProducer, cm *credit.Module, um *user.Module, native wechat.NativeAPIService, js wechat.JSAPIService) service.Service {
	wechatConfig := initWechatConfig()
	nativePaymentService := ioc.InitWechatNativePaymentService(native, wechatConfig)
	userService := um.Svc
	jsapiPaymentService := ioc.InitWechatJSAPIPaymentService(js, userService, wechatConfig)
	v := newPaymentServices(nativePaymentService, jsapiPaymentService)
	serviceService := cm.Svc
	generator := sequencenumber.NewGenerator()
	db := testioc.InitDB()
	daoPaymentDAO := InitDAO(db)
	paymentRepository := repository.NewPaymentRepository(daoPaymentDAO)
	service2 := service.NewService(v, serviceService, generator, paymentRepository, p)
	return service2
}

// wire.go:

var serviceSet = wire.NewSet(
	initWechatConfig, wire.FieldsOf(new(*credit.Module), "Svc"), wire.FieldsOf(new(*user.Module), "Svc"), sequencenumber.NewGenerator, testioc.BaseSet, InitDAO, repository.NewPaymentRepository,
)

func newPaymentServices(n *wechat.NativePaymentService, j *wechat.JSAPIPaymentService) map[payment.ChannelType]service.PaymentService {
	return map[payment.ChannelType]service.PaymentService{payment.ChannelTypeWechat: n, payment.ChannelTypeWechatJS: j}
}

var (
	once       = &sync.Once{}
	paymentDAO dao.PaymentDAO
)

func InitDAO(db *gorm.DB) dao.PaymentDAO {
	once.Do(func() {
		_ = dao.InitTables(db)
		paymentDAO = dao.NewPaymentGORMDAO(db)
	})
	return paymentDAO
}

func initWechatConfig() ioc.WechatConfig {
	return ioc.WechatConfig{
		AppID:            "MockAPPID",
		MchID:            "MockMchID",
		PaymentNotifyURL: "MockPaymentNotifyURL",
	}
}
