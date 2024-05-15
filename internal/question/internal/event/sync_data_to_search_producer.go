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

	"github.com/ecodeclub/mq-api"
)

const (
	SyncTopic = "sync_data_to_search"
)

type SyncDataToSearchEventProducer interface {
	Produce(ctx context.Context, evt QuestionEvent) error
}

type syncEventProducerProducer struct {
	producer mq.Producer
}

func NewSyncEventProducer(q mq.MQ) (SyncDataToSearchEventProducer, error) {
	p, err := q.Producer(SyncTopic)
	if err != nil {
		return nil, err
	}
	return &syncEventProducerProducer{
		producer: p,
	}, nil
}

func (s *syncEventProducerProducer) Produce(ctx context.Context, evt QuestionEvent) error {
	data, err := json.Marshal(&evt)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	_, err = s.producer.Produce(ctx, &mq.Message{Value: data})
	if err != nil {
		return fmt.Errorf("发送同步搜索消息失败: %w", err)
	}
	return nil
}
