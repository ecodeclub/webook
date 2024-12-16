package cases

import (
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/cases/internal/event"
	"github.com/ecodeclub/webook/internal/cases/internal/repository"
	"github.com/ecodeclub/webook/internal/cases/internal/service"

	"github.com/gotomicro/ego/core/econf"
)

func InitKnowledgeBaseSvc(svc ai.KnowledgeBaseService, repo repository.CaseRepo) service.KnowledgeBaseService {
	type Config struct {
		KnowledgeBaseID string `yaml:"knowledgeBaseID"`
	}
	var cfg Config
	err := econf.UnmarshalKey("case.zhipu", &cfg)
	if err != nil {
		panic(err)
	}
	return service.NewKnowledgeBaseService(repo, svc, cfg.KnowledgeBaseID)
}

func InitKnowledgeBaseEvt(q mq.MQ) event.KnowledgeBaseEventProducer {
	type Config struct {
		KnowledgeBaseID string `yaml:"knowledgeBaseID"`
	}
	var cfg Config
	err := econf.UnmarshalKey("case.zhipu", &cfg)
	if err != nil {
		panic(err)
	}
	p, err := event.NewKnowledgeBaseEventProducer(cfg.KnowledgeBaseID, q)
	if err != nil {
		panic(err)
	}
	return p
}
