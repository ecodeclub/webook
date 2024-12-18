package ai

import (
	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/knowledge_base/zhipu"
	"github.com/gotomicro/ego/core/econf"
)

func InitZhipuKnowledgeBase(repo repository.KnowledgeBaseRepo) KnowledgeBaseService {
	type Config struct {
		APIKey string `yaml:"apikey"`
	}
	var cfg Config
	err := econf.UnmarshalKey("zhipu", &cfg)
	if err != nil {
		panic(err)
	}
	svc, err := zhipu.NewKnowledgeBase(cfg.APIKey, repo)
	if err != nil {
		panic(err)
	}
	return svc
}
