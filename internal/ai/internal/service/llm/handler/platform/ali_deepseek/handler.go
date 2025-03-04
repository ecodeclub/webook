package ali_deepseek

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/gotomicro/ego/core/elog"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/ssestream"
)

const (
	baseUrl = "https://dashscope.aliyuncs.com/compatible-mode/v1/"
)

type Delta struct {
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content"`
}

type Handler struct {
	client     *openai.Client
	repo       repository.LLMLogRepo
	logger     *elog.Component
	configRepo repository.ConfigRepository
}

func NewHandler(apikey string, repo repository.LLMLogRepo, configRepo repository.ConfigRepository) *Handler {
	client := openai.NewClient(
		option.WithBaseURL(baseUrl),
		option.WithAPIKey(apikey),
	)
	return &Handler{
		client:     client,
		repo:       repo,
		configRepo: configRepo,
		logger:     elog.DefaultLogger,
	}
}

func (h *Handler) StreamHandle(ctx context.Context, req domain.LLMRequest) (chan domain.StreamEvent, error) {
	config, err := h.findConfig(ctx, req)
	if err != nil {
		return nil, err
	}
	req.Config = config
	eventCh := make(chan domain.StreamEvent, 10)
	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(req.Prompt()),
		}),

		Model: openai.F(req.Config.Model),
		StreamOptions: openai.F(openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.F(true),
		}),
	}

	go func() {
		newCtx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
		defer cancel()
		stream := h.client.Chat.Completions.NewStreaming(newCtx, params)
		h.recv(req, eventCh, stream)
	}()

	return eventCh, nil
}

func (h *Handler) findConfig(ctx context.Context, req domain.LLMRequest) (domain.BizConfig, error) {
	return h.configRepo.GetConfig(ctx, req.Biz)
}

func (h *Handler) recv(req domain.LLMRequest, eventCh chan domain.StreamEvent,
	stream *ssestream.Stream[openai.ChatCompletionChunk]) {
	defer close(eventCh)
	acc := openai.ChatCompletionAccumulator{}

	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)
		// 建议在处理完 JustFinished 事件后使用数据块
		if len(chunk.Choices) > 0 {
			// 说明没结束
			if chunk.Choices[0].FinishReason == "" {
				var delta Delta
				err := json.Unmarshal([]byte(chunk.Choices[0].Delta.JSON.RawJSON()), &delta)
				if err != nil {
					eventCh <- domain.StreamEvent{
						Error: err,
					}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					h.saveRecord(ctx, req, domain.RecordStatusFailed, domain.LLMResponse{})
					cancel()
					return
				}
				eventCh <- domain.StreamEvent{
					Content:          delta.Content,
					ReasoningContent: delta.ReasoningContent,
				}
			}
		}
	}
	eventCh <- domain.StreamEvent{
		Done: true,
	}
	// 记录数据
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if stream.Err() != nil {
		h.saveRecord(ctx, req, domain.RecordStatusFailed, domain.LLMResponse{})
		h.logger.Error("获取deepseek 流数据失败", elog.FieldErr(stream.Err()))
		return
	}
	var ans string
	if len(acc.Choices) > 0 {
		ans = acc.Choices[0].Message.Content
	}
	tokens := acc.Usage.TotalTokens
	amt := math.Ceil(float64(tokens*req.Config.Price) / float64(1000))

	h.saveRecord(ctx, req, domain.RecordStatusSuccess, domain.LLMResponse{
		Tokens: tokens,
		Answer: ans,
		Amount: int64(amt),
	})

}

func (h *Handler) saveRecord(ctx context.Context, req domain.LLMRequest, status domain.RecordStatus, resp domain.LLMResponse) {
	log := domain.LLMRecord{
		Tid:            req.Tid,
		Biz:            req.Biz,
		Uid:            req.Uid,
		Input:          req.Input,
		Status:         domain.RecordStatusProcessing,
		KnowledgeId:    req.Config.KnowledgeId,
		PromptTemplate: req.Config.PromptTemplate,
	}
	log.Tokens = resp.Tokens
	log.Status = status
	log.Answer = resp.Answer
	_, err1 := h.repo.SaveLog(ctx, log)
	if err1 != nil {
		h.logger.Error("保存 LLM 访问记录失败", elog.FieldErr(err1))
	}
}
