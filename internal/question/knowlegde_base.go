package baguwen

import (
	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/question/internal/repository"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/gotomicro/ego/core/econf"
)

func InitKnowledgeBaseSvc(svc ai.KnowledgeBaseService, queSvc repository.Repository) service.QuestionKnowledgeBase {
	type Config struct {
		KnowledgeBaseID string `yaml:"knowledgeBaseID"`
	}
	var cfg Config
	err := econf.UnmarshalKey("zhipuBaseID", &cfg)
	if err != nil {
		panic(err)
	}
	return service.NewQuestionKnowledgeBase(cfg.KnowledgeBaseID, queSvc, svc)
}
