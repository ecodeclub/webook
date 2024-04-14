package events

import (
	"context"
	"encoding/json"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/mq-api/kafka"
)

type PaymentProducer struct {
	producer kafka.Producer
}

func NewPaymentProducer(p kafka.Producer) (*PaymentProducer, error) {
	return &PaymentProducer{
		p,
	}, nil
}

func (s *PaymentProducer) ProducePaymentEvent(ctx context.Context, evt PaymentEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, err = s.producer.Produce(ctx, &mq.Message{
		Key:   []byte(evt.OrderSN),
		Topic: evt.Topic(),
		Value: data,
	})
	return err
}
