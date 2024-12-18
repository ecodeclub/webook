package ai

import (
	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/knowledge_base"
	"github.com/ecodeclub/webook/internal/ai/internal/web"
)

type LLMRequest = domain.LLMRequest
type LLMResponse = domain.LLMResponse
type LLMService = llm.Service
type KnowledgeBaseService = knowledge_base.RepositoryBaseSvc
type AdminHandler = web.AdminHandler
type LLMHandler = web.Handler
type KnowledgeBaseFile = domain.KnowledgeBaseFile

const (
	RepositoryBaseTypeRetrieval = "retrieval"
	RepositoryBaseTypeFineTune  = "finetune"
)
