// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/interactive/internal/service"
	"github.com/gotomicro/ego/core/elog"
)

const syncTopic = "interactive_events"

type Consumer struct {
	handlerMap map[string]handleFunc
	consumer   mq.Consumer
	svc        service.InteractiveService
	logger     *elog.Component
}

func NewSyncConsumer(svc service.InteractiveService, q mq.MQ) (*Consumer, error) {
	groupID := "interactive_group"
	consumer, err := q.Consumer(syncTopic, groupID)
	if err != nil {
		return nil, err
	}
	handlerMap := map[string]handleFunc{
		"like":    likeHandle,
		"collect": collectHandle,
		"view":    viewHandle,
	}
	return &Consumer{
		handlerMap: handlerMap,
		consumer:   consumer,
		svc:        svc,
		logger:     elog.DefaultLogger,
	}, nil
}

func (s *Consumer) Consume(ctx context.Context) error {
	msg, err := s.consumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("获取消息失败: %w", err)
	}

	var evt Event
	err = json.Unmarshal(msg.Value, &evt)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}
	handler, ok := s.handlerMap[evt.Action]
	if !ok {
		return errors.New("未找到相关业务的处理方法")
	}
	err = handler(ctx, s.svc, evt)
	if err != nil {
		s.logger.Error("同步消息失败", elog.Any("interactive_event", evt))
	}
	return err
}

func (s *Consumer) Start(ctx context.Context) {
	go func() {
		for {
			err := s.Consume(ctx)
			if err != nil {
				s.logger.Error("同步事件失败", elog.FieldErr(err))
			}
		}
	}()
}
func (s *Consumer) Stop(_ context.Context) error {
	return s.consumer.Close()
}
