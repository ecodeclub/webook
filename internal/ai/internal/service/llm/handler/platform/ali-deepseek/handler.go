package ali_deepseek

import (
	"context"
	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/ssestream"
)

const (
	baseUrl = "https://dashscope.aliyuncs.com/compatible-mode/v1/"
)

type Handler struct {
	client *openai.Client
}

func NewHandler(apikey string) *Handler {
	client := openai.NewClient(
		option.WithBaseURL(baseUrl),
		option.WithBaseURL(apikey),
	)
	return &Handler{
		client: client,
	}
}

func (h *Handler) StreamHandle(ctx context.Context, req domain.LLMRequest) (chan domain.StreamEvent, error) {
	eventCh := make(chan domain.StreamEvent, 10)
	model := openai.ChatModel("deepseek-r1")
	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(req.Prompt()),
		}),
		Model: openai.F(model),
		StreamOptions: openai.F(openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.F(true),
		}),
	}
	stream := h.client.Chat.Completions.NewStreaming(ctx, params)
	go h.Recv(eventCh, stream)
	return eventCh,nil
}

func (h *Handler) Recv(eventCh chan domain.StreamEvent, stream *ssestream.Stream[openai.ChatCompletionChunk]) {
	acc := openai.ChatCompletionAccumulator{}
	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)
		// 建议在处理完 JustFinished 事件后使用数据块
		if len(chunk.Choices) > 0 {
			// 说明没结束
			if chunk.Choices[0].FinishReason == "" {
				eventCh <- domain.StreamEvent{
					Content: chunk.Choices[0].Delta.Content,
				}
			}
		}
	}
	// 记录

}
