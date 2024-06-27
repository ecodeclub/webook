package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/ecodeclub/ekit/retry"
	"github.com/ecodeclub/webook/internal/marketing/internal/event"
	"github.com/ecodeclub/webook/internal/marketing/internal/event/producer"
)

var _ OrderHandler = (*ProductServiceHandler)(nil)

// ProductServiceHandler 面试服务商品处理器 —— 通过企业微信机器人发群消息
type ProductServiceHandler struct {
	qywechatEventProducer producer.QYWeiChatEventProducer
}

func NewProductServiceHandler(qywechatEventProducer producer.QYWeiChatEventProducer) *ProductServiceHandler {
	return &ProductServiceHandler{qywechatEventProducer: qywechatEventProducer}
}

func (h *ProductServiceHandler) Handle(ctx context.Context, info OrderInfo) error {
	strategy, _ := retry.NewExponentialBackoffRetryStrategy(time.Second, time.Second*32, 6)
	var err error
	for {

		err = h.qywechatEventProducer.Produce(ctx, event.QYWechatEvent{
			Content: fmt.Sprintf("新订单: ID=%d", info.Order.ID),
		})

		if err == nil {
			return nil
		}

		next, ok := strategy.Next()
		if !ok {
			return fmt.Errorf("超过重试次数: %w", err)
		}

		time.Sleep(next)
	}
}
