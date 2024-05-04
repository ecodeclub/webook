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

package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/search/internal/service"
	"github.com/gotomicro/ego/core/elog"
)

type SyncConsumer struct {
	svc      service.SyncService
	consumer mq.Consumer
	logger   *elog.Component
}

func NewSyncConsumer(svc service.SyncService, q mq.MQ) (*SyncConsumer, error) {
	groupID := "sync"
	consumer, err := q.Consumer(SyncTopic, groupID)
	if err != nil {
		return nil, err
	}
	return &SyncConsumer{
		svc:      svc,
		consumer: consumer,
		logger:   elog.DefaultLogger,
	}, nil
}

func (s *SyncConsumer) Consume(ctx context.Context) error {
	msg, err := s.consumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("获取消息失败: %w", err)
	}

	var evt SyncEvent
	err = json.Unmarshal(msg.Value, &evt)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}
	log.Println("xxxxxxxx", evt)
	indexName := getIndexName(evt.Biz)
	docId := strconv.Itoa(evt.BizID)
	err = s.svc.Input(ctx, indexName, docId, evt.Data)
	if err != nil {
		s.logger.Error("同步消息失败", elog.Any("SyncEvent", evt))
	}
	return err
}

func (s *SyncConsumer) Start(ctx context.Context) {
	go func() {
		for {
			err := s.Consume(ctx)
			if err != nil {
				s.logger.Error("同步事件失败", elog.FieldErr(err))
			}
		}
	}()
}
func (s *SyncConsumer) Stop(_ context.Context) error {
	return s.consumer.Close()
}

func getIndexName(biz string) string {
	return fmt.Sprintf("%s_index", biz)
}
