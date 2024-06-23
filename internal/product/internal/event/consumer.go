package event

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/product/internal/service"
	"github.com/gotomicro/ego/core/elog"
)

type ProductConsumer struct {
	svc      service.Service
	consumer mq.Consumer
	logger   *elog.Component
}

func NewProductConsumer(svc service.Service, q mq.MQ) (*ProductConsumer, error) {
	groupID := "create_product_group"
	consumer, err := q.Consumer(CreateProductTopic, groupID)
	if err != nil {
		return nil, err
	}
	return &ProductConsumer{
		svc:      svc,
		consumer: consumer,
		logger:   elog.DefaultLogger,
	}, nil
}

func (s *ProductConsumer) Consume(ctx context.Context) error {
	msg, err := s.consumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("获取消息失败: %w", err)
	}

	var evt SPUEvent
	err = json.Unmarshal(msg.Value, &evt)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}
	_, err = s.svc.SaveProduct(ctx, evt.ToDomain(), evt.UID)
	return err
}

func (s *ProductConsumer) Start(ctx context.Context) {
	go func() {
		for {
			err := s.Consume(ctx)
			if err != nil {
				s.logger.Error("同步事件失败", elog.FieldErr(err))
			}
		}
	}()
}
func (s *ProductConsumer) Stop(_ context.Context) error {
	return s.consumer.Close()
}
