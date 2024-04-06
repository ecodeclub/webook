package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ecodeclub/mq-api"
)

type CreditsEventProducer struct {
	producer mq.Producer
}

func NewCreditsEventProducer(producer mq.Producer) *CreditsEventProducer {
	return &CreditsEventProducer{producer: producer}
}

func (p *CreditsEventProducer) Produce(ctx context.Context, evt CreditsEvent) error {
	data, err := json.Marshal(&evt)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	_, err = p.producer.Produce(ctx, &mq.Message{Value: data})
	if err != nil {
		return fmt.Errorf("发送反馈成功消息失败: %w", err)
	}
	return nil
}
