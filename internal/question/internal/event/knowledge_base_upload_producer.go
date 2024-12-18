package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ecodeclub/mq-api"
)

type KnowledgeBaseEventProducer interface {
	Produce(ctx context.Context, evt KnowledgeBaseEvent) error
}

type knowledgeBaseEventProducer struct {
	producer mq.Producer
	baseId   string
}

func (k *knowledgeBaseEventProducer) Produce(ctx context.Context, evt KnowledgeBaseEvent) error {
	evt.KnowledgeBaseID = k.baseId
	data, err := json.Marshal(&evt)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	_, err = k.producer.Produce(ctx, &mq.Message{Value: data})
	if err != nil {
		return fmt.Errorf("向topic=%s发送event=%#v失败: %w", KnowledgeBaseUploadTopic, evt, err)
	}
	return nil
}

func NewKnowledgeBaseEventProducer(baseId string, q mq.MQ) (KnowledgeBaseEventProducer, error) {
	pro, err := q.Producer(KnowledgeBaseUploadTopic)
	if err != nil {
		return nil, err
	}
	return &knowledgeBaseEventProducer{
		producer: pro,
		baseId:   baseId,
	}, nil
}
