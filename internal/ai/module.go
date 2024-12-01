package ai

type Module struct {
	Svc          LLMService
	Hdl          *LLMHandler
	AdminHandler *AdminHandler
}
