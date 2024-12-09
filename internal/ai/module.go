package ai

type Module struct {
	Svc              LLMService
	KnowledgeBaseSvc KnowledgeBaseService
	Hdl              *LLMHandler
	AdminHandler     *AdminHandler
}
