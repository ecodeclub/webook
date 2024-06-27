package producer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ecodeclub/webook/internal/marketing/internal/event"
)

type QYWeiChatEventProducer interface {
	Produce(ctx context.Context, evt event.QYWechatEvent) error
}

type POSTFunc func(url, contentType string, body io.Reader) (resp *http.Response, err error)

type qyWeChatEventProducer struct {
	url      string
	postFunc POSTFunc
}

func NewQYWeChatEventProducer(url string, postFunc POSTFunc) QYWeiChatEventProducer {
	return &qyWeChatEventProducer{
		url:      url,
		postFunc: postFunc,
	}
}

func (q *qyWeChatEventProducer) Produce(_ context.Context, evt event.QYWechatEvent) error {
	type Message struct {
		MsgType string              `json:"msgtype"`
		Text    event.QYWechatEvent `json:"text"`
	}
	data, err := json.Marshal(&Message{MsgType: "text", Text: evt})
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	resp, err := q.postFunc(q.url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("处理请求失败: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return nil
}
