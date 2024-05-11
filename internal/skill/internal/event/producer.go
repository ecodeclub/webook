package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ecodeclub/mq-api"
)

const (
	SyncTopic = "sync_data_to_search"
)

type SyncEventProducer interface {
	Produce(ctx context.Context, evt SkillEvent) error
}
type syncEventProducerProducer struct {
	producer mq.Producer
}

func NewSyncEventProducer(q mq.MQ) (SyncEventProducer, error) {
	p, err := q.Producer(SyncTopic)
	if err != nil {
		return nil, err
	}
	return &syncEventProducerProducer{
		producer: p,
	}, nil
}

func (s *syncEventProducerProducer) Produce(ctx context.Context, evt SkillEvent) error {
	data, err := json.Marshal(&evt)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	_, err = s.producer.Produce(ctx, &mq.Message{
		Value: data,
	})
	if err != nil {
		return fmt.Errorf("发送同步搜索消息失败: %w", err)
	}
	return nil
}
