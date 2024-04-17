//go:build wireinject

package payment

import (
	"sync"
	"time"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/events"
	"github.com/ecodeclub/webook/internal/payment/internal/repository"
	"github.com/ecodeclub/webook/internal/payment/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/payment/internal/service"
	credit2 "github.com/ecodeclub/webook/internal/payment/internal/service/credit"
	"github.com/ecodeclub/webook/internal/payment/internal/web"
	"github.com/ecodeclub/webook/internal/payment/ioc"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"github.com/gotomicro/ego/core/elog"
	"gorm.io/gorm"
)

type Handler = web.Handler
type Payment = domain.Payment
type Record = domain.PaymentRecord
type Channel = domain.PaymentChannel

var ChannelTypeCredit int64 = domain.ChannelTypeCredit
var ChannelTypeWechat int64 = domain.ChannelTypeWechat

type Service = service.Service

func InitModule(db *egorm.Component,
	mq mq.MQ,
	c ecache.Cache,
	cm *credit.Module) (*Module, error) {
	wire.Build(
		initLogger,
		ioc.InitWechatNativeService,
		ioc.InitWechatConfig,
		ioc.InitWechatNotifyHandler,
		ioc.InitWechatClient,
		initDAO,
		events.NewPaymentProducer,
		web.NewHandler,
		service.NewService,
		credit2.NewCreditPaymentService,
		repository.NewPaymentRepository,
		paymentDDLFunc,
		initProducer,
		sequencenumber.NewGenerator,
		wire.FieldsOf(new(*credit.Module), "Svc"),
		wire.Struct(new(Module), "*"),
	)
	return new(Module), nil
}

var (
	once     = &sync.Once{}
	orderDAO dao.PaymentDAO
)

func initProducer(mq mq.MQ) (mq.Producer, error) {
	return mq.Producer("payment_events")
}

func paymentDDLFunc() func() int64 {
	return func() int64 {
		return time.Now().Add(time.Minute * 30).UnixMilli()
	}
}

func initDAO(db *gorm.DB) dao.PaymentDAO {
	once.Do(func() {
		_ = dao.InitTables(db)
		orderDAO = dao.NewPaymentGORMDAO(db)
	})
	return orderDAO
}

func initLogger() *elog.Component {
	return elog.DefaultLogger
}
