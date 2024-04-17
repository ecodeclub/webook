package events

import (
	"context"
	"encoding/json"

	"github.com/ecodeclub/mq-api"
)

type PaymentProducer struct {
	producer mq.Producer
}

func NewPaymentProducer(p mq.Producer) (Producer, error) {
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
