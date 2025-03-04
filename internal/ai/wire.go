//go:build wireinject

package ai

import (
	"context"
	"sync"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/ai/internal/event"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/knowledge_base"

	"github.com/ecodeclub/webook/internal/ai/internal/service"
	"github.com/ecodeclub/webook/internal/ai/internal/web"

	"github.com/ecodeclub/webook/internal/ai/internal/service/llm"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/config"
	aicredit "github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/credit"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/log"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/record"

	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

func InitModule(db *egorm.Component, creditSvc *credit.Module, q mq.MQ) (*Module, error) {
	wire.Build(
		InitAliDeepSeekHandler,
		llm.NewLLMService,
		repository.NewLLMLogRepo,
		repository.NewLLMCreditLogRepo,
		repository.NewCachedConfigRepository,

		InitLLMCreditLogDAO,
		dao.NewGORMLLMLogDAO,
		dao.NewGORMConfigDAO,

		InitZhipuKnowledgeBase,
		dao.NewKnowledgeBaseDAO,
		repository.NewKnowledgeBaseRepo,

		config.NewBuilder,
		log.NewHandler,
		record.NewHandler,
		aicredit.NewHandlerBuilder,

		InitCompositionHandlerUsingZhipu,
		InitCommonHandlers,
		InitZhipu,

		service.NewGeneralService,
		service.NewJDService,
		service.NewConfigService,
		web.NewHandler,
		web.NewAdminHandler,

		initKnowledgeConsumer,
		wire.Struct(new(Module), "*"),
		wire.FieldsOf(new(*credit.Module), "Svc"),
	)
	return new(Module), nil
}

var daoOnce = sync.Once{}

func InitTableOnce(db *gorm.DB) {
	daoOnce.Do(func() {
		err := dao.InitTables(db)
		if err != nil {
			panic(err)
		}
	})
}

func InitLLMCreditLogDAO(db *egorm.Component) dao.LLMCreditDAO {
	InitTableOnce(db)
	return dao.NewLLMCreditLogDAO(db)
}

func initKnowledgeConsumer(svc knowledge_base.RepositoryBaseSvc, q mq.MQ) *event.KnowledgeBaseConsumer {
	c, err := event.NewKnowledgeBaseConsumer(svc, q)
	if err != nil {
		panic(err)
	}
	c.Start(context.Background())
	return c
}
