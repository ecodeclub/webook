//go:build wireinject

package startup

import (
	"sync"

	"github.com/ecodeclub/webook/internal/ai/internal/event"

	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/knowledge_base/zhipu"

	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/knowledge_base"

	"github.com/ecodeclub/webook/internal/ai/internal/service"
	"github.com/ecodeclub/webook/internal/ai/internal/web"

	hdlmocks "github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/mocks"
	streamhdlmocks "github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/stream_mocks"

	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler"
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

func InitModule(db *egorm.Component,
	hdl *hdlmocks.MockHandler,
	streamHandler *streamhdlmocks.MockStreamHandler,
	baseSvc knowledge_base.RepositoryBaseSvc,
	creditSvc *credit.Module,
	consumer *event.KnowledgeBaseConsumer,
) (*ai.Module, error) {
	wire.Build(
		llm.NewLLMService,
		repository.NewLLMLogRepo,
		repository.NewLLMCreditLogRepo,
		repository.NewCachedConfigRepository,

		InitLLMCreditLogDAO,
		dao.NewGORMLLMLogDAO,
		dao.NewGORMConfigDAO,

		config.NewBuilder,
		log.NewHandler,
		record.NewHandler,
		aicredit.NewHandlerBuilder,

		ai.InitCommonHandlers,
		InitRootHandler,
		InitStreamHandler,
		service.NewGeneralService,
		service.NewJDService,
		service.NewConfigService,
		web.NewHandler,
		web.NewAdminHandler,
		wire.Struct(new(ai.Module), "*"),
		wire.FieldsOf(new(*credit.Module), "Svc"),
	)
	return new(ai.Module), nil
}

func InitKnowledgeBaseSvc(db *egorm.Component, apikey string) knowledge_base.RepositoryBaseSvc {
	knowledgeDao := dao.NewKnowledgeBaseDAO(db)
	knowledgeRepo := repository.NewKnowledgeBaseRepo(knowledgeDao)
	// 将智谱对应的apikey写到环境变量
	knowledgeSvc, err := zhipu.NewKnowledgeBase(apikey, knowledgeRepo)
	if err != nil {
		panic(err)
	}
	return knowledgeSvc
}

func InitRootHandler(common []handler.Builder, hdl *hdlmocks.MockHandler) handler.Handler {
	return handler.NewCompositionHandler(common, hdl)
}
func InitStreamHandler(streamHdl *streamhdlmocks.MockStreamHandler) handler.StreamHandler {
	return streamHdl
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
