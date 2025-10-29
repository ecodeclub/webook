package ai

import (
	"github.com/ecodeclub/webook/internal/ai/internal/event"
	"github.com/ecodeclub/webook/internal/ai/internal/web"
)

type MockInterviewHandler = web.MockInterviewHandler

type Module struct {
	Svc              LLMService
	KnowledgeBaseSvc KnowledgeBaseService
	Hdl              *LLMHandler
	AdminHandler     *AdminHandler
	MockInterviewHdl *MockInterviewHandler
	C                *event.KnowledgeBaseConsumer
}
