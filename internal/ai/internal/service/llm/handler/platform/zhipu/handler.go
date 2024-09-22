package zhipu

import (
	"context"
	"math"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/yankeguo/zhipu"
)

// Handler 如果后续有不同的实现，就提供不同的实现
type Handler struct {
	client *zhipu.Client
}

func NewHandler(apikey string) (*Handler, error) {
	client, err := zhipu.NewClient(zhipu.WithAPIKey(apikey))
	if err != nil {
		return nil, err
	}
	return &Handler{
		client: client,
		// 后续可以做成可配置的
	}, err
}

func (h *Handler) Name() string {
	return "zhipu"
}

func (h *Handler) Handle(ctx context.Context, req domain.LLMRequest) (domain.LLMResponse, error) {
	// 这边它不会调用 next，因为它是最终的出口
	chatReq := h.buildReq(req)
	completion, err := chatReq.Do(ctx)
	if err != nil {
		return domain.LLMResponse{}, err
	}
	tokens := completion.Usage.TotalTokens
	// 现在的报价都是 N/1k token
	// 而后向上取整
	amt := math.Ceil(float64(tokens*req.Config.Price) / float64(1000))
	// 金额只有具体的模型才知道怎么算
	resp := domain.LLMResponse{
		Tokens: tokens,
		Amount: int64(amt),
	}

	if len(completion.Choices) > 0 {
		resp.Answer = completion.Choices[0].Message.Content
	}
	return resp, nil
}

func (h *Handler) buildReq(req domain.LLMRequest) *zhipu.ChatCompletionService {
	svc := h.client.ChatCompletion(req.Config.Model)
	chatReq := svc.AddMessage(zhipu.ChatCompletionMessage{
		Role:    zhipu.RoleUser,
		Content: req.Prompt,
	})

	if req.Config.Temperature > 0 {
		chatReq = chatReq.SetTemperature(req.Config.Temperature)
	}

	if req.Config.TopP > 0 {
		chatReq = chatReq.SetTopP(req.Config.TopP)
	}

	if req.Config.SystemPrompt != "" {
		chatReq = chatReq.AddMessage(zhipu.ChatCompletionMessage{
			Role:    zhipu.RoleSystem,
			Content: req.Config.SystemPrompt,
		})
	}

	if req.Config.KnowledgeId != "" {
		chatReq = chatReq.AddTool(zhipu.ChatCompletionToolRetrieval{
			KnowledgeID: req.Config.KnowledgeId,
		})
	}
	return chatReq
}
