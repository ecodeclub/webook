package zhipu

import (
	"context"

	"github.com/yankeguo/zhipu"
)

type Client struct {
	client      *zhipu.Client
	apiKey      string
	knowledgeId string
}

func NewClient(apikey, knowledgeId string) (*Client, error) {
	client, err := zhipu.NewClient(zhipu.WithAPIKey(apikey))
	if err != nil {
		return nil, err
	}
	return &Client{
		client:      client,
		apiKey:      apikey,
		knowledgeId: knowledgeId,
	}, nil
}

func (c *Client) ChatCompletion(ctx context.Context, msg zhipu.ChatCompletionMessage) (zhipu.ChatCompletionResponse, error) {
	return c.client.ChatCompletion("glm-4").AddTool(zhipu.ChatCompletionToolRetrieval{
		KnowledgeID: c.knowledgeId,
	}).AddMessage(msg).Do(ctx)
}
