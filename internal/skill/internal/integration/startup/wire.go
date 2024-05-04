//go:build wireinject

package startup

import (
	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/webook/internal/cases"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/skill/internal/event"
	"github.com/ecodeclub/webook/internal/skill/internal/repository"
	"github.com/ecodeclub/webook/internal/skill/internal/repository/cache"
	dao2 "github.com/ecodeclub/webook/internal/skill/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/skill/internal/service"
	"github.com/ecodeclub/webook/internal/skill/internal/web"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
	"sync"
)

func InitHandler(bm *baguwen.Module, cm *cases.Module, p event.SyncEventProducer) (*web.Handler, error) {
	wire.Build(testioc.BaseSet, initHandler)
	return new(web.Handler), nil
}

func initHandler(
	db *egorm.Component,
	ec ecache.Cache,
	queModule *baguwen.Module,
	caseModule *cases.Module,
	p event.SyncEventProducer) (*web.Handler, error) {
	wire.Build(
		InitSkillDAO,
		wire.FieldsOf(new(*baguwen.Module), "Svc"),
		wire.FieldsOf(new(*cases.Module), "Svc"),
		cache.NewSkillCache,
		repository.NewSkillRepo,
		service.NewSkillService,
		web.NewHandler,
	)
	return new(web.Handler), nil
}

var daoOnce = sync.Once{}

func InitTableOnce(db *gorm.DB) {
	daoOnce.Do(func() {
		err := dao2.InitTables(db)
		if err != nil {
			panic(err)
		}
	})
}

func InitSkillDAO(db *egorm.Component) dao2.SkillDAO {
	InitTableOnce(db)
	return dao2.NewSkillDAO(db)
}
