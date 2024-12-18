package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/knowledge_base"
	"github.com/gotomicro/ego/core/elog"
)

type KnowledgeBaseConsumer struct {
	svc      knowledge_base.RepositoryBaseSvc
	consumer mq.Consumer
	logger   *elog.Component
}

func NewKnowledgeBaseConsumer(svc knowledge_base.RepositoryBaseSvc, q mq.MQ) (*KnowledgeBaseConsumer, error) {
	groupID := "knowledge_base_group"
	consumer, err := q.Consumer(KnowledgeBaseUploadTopic, groupID)
	if err != nil {
		return nil, err
	}
	return &KnowledgeBaseConsumer{
		svc:      svc,
		consumer: consumer,
		logger:   elog.DefaultLogger,
	}, nil
}

func (k *KnowledgeBaseConsumer) Consume(ctx context.Context) error {
	msg, err := k.consumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("获取消息失败: %w", err)
	}

	var evt KnowledgeBaseUploadEvent
	err = json.Unmarshal(msg.Value, &evt)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}
	log.Println("xxxxx", evt)
	err = k.svc.UploadFile(ctx, domain.KnowledgeBaseFile{
		Biz:             evt.Biz,
		BizID:           evt.BizID,
		Name:            evt.Name,
		Data:            evt.Data,
		Type:            evt.Type,
		KnowledgeBaseID: evt.KnowledgeBaseID,
	})
	if err != nil {
		return fmt.Errorf("上传文件到知识库失败 %w", err)
	}
	return nil
}

func (k *KnowledgeBaseConsumer) Start(ctx context.Context) {
	go func() {
		for {
			err := k.Consume(ctx)
			if err != nil {
				k.logger.Error("同步事件失败", elog.FieldErr(err))
			}
		}
	}()
}

func (k *KnowledgeBaseConsumer) Stop(_ context.Context) error {
	return k.consumer.Close()
}
