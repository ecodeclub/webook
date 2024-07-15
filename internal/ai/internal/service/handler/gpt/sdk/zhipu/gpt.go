package zhipu

import (
	"context"

	"github.com/yankeguo/zhipu"
)

// 智谱

type GPT struct {
	sdk *Client
}

func NewGpt(sdk *Client) *GPT {
	return &GPT{
		sdk: sdk,
	}
}

func (g *GPT) Invoke(ctx context.Context, input []string) (int64, string, error) {
	params := g.newParams(input)
	resp, err := g.sdk.ChatCompletion(ctx, params)
	if err != nil {
		return 0, "", err
	}
	tokens := resp.Usage.TotalTokens
	// 默认是一个问题
	var ans string
	if len(resp.Choices) > 0 {
		ans = resp.Choices[0].Message.Content
	}
	return tokens, ans, nil
}

func (g *GPT) newParams(inputs []string) zhipu.ChatCompletionMessage {
	msg := inputs[0]
	return zhipu.ChatCompletionMessage{
		Role:    "user",
		Content: msg,
	}
}
