package baguwen

import (
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/question/internal/event"
	"github.com/ecodeclub/webook/internal/question/internal/repository"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/gotomicro/ego/core/econf"
)

func InitKnowledgeBaseSvc(svc ai.KnowledgeBaseService, queSvc repository.Repository) service.QuestionKnowledgeBase {
	type Config struct {
		KnowledgeBaseID string `yaml:"knowledgeBaseID"`
	}
	var cfg Config
	err := econf.UnmarshalKey("question.zhipu", &cfg)
	if err != nil {
		panic(err)
	}
	return service.NewQuestionKnowledgeBase(cfg.KnowledgeBaseID, queSvc, svc)
}

func InitKnowledgeBaseUploadProducer(q mq.MQ) event.KnowledgeBaseEventProducer {
	type Config struct {
		KnowledgeBaseID string `yaml:"knowledgeBaseID"`
	}
	var cfg Config
	err := econf.UnmarshalKey("question.zhipu", &cfg)
	if err != nil {
		panic(err)
	}
	pro, err := event.NewKnowledgeBaseEventProducer(cfg.KnowledgeBaseID, q)
	if err != nil {
		panic(err)
	}
	return pro
}
