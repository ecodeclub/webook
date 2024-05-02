package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ecodeclub/mq-api"
)

//go:generate mockgen -source=./producer.go -package=evtmocks -destination=./mocks/producer.mock.go -typed PaymentEventProducer
type PaymentEventProducer interface {
	Produce(ctx context.Context, evt PaymentEvent) error
}

type paymentEventProducer struct {
	producer mq.Producer
}

func NewPaymentEventProducer(p mq.Producer) (PaymentEventProducer, error) {
	return &paymentEventProducer{
		p,
	}, nil
}

func (s *paymentEventProducer) Produce(ctx context.Context, evt PaymentEvent) error {
	data, err := json.Marshal(&evt)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	_, err = s.producer.Produce(ctx, &mq.Message{
		Key:   []byte(evt.OrderSN),
		Value: data,
	})
	return err
}
