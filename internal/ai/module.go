package ai

import "github.com/ecodeclub/webook/internal/ai/internal/event"

type Module struct {
	Svc              LLMService
	KnowledgeBaseSvc KnowledgeBaseService
	Hdl              *LLMHandler
	AdminHandler     *AdminHandler
	C                *event.KnowledgeBaseConsumer
}
