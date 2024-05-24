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

package mqx

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ecodeclub/mq-api"
)

type Producer[T any] interface {
	Produce(ctx context.Context, evt T) error
}

type GeneralProducer[T any] struct {
	producer mq.Producer
	topic    string
}

func NewGeneralProducer[T any](q mq.MQ, topic string) (*GeneralProducer[T], error) {
	p, err := q.Producer(topic)
	return &GeneralProducer[T]{
		producer: p,
		topic:    topic,
	}, err
}

func (p *GeneralProducer[T]) Produce(ctx context.Context, evt T) error {
	data, err := json.Marshal(&evt)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	_, err = p.producer.Produce(ctx, &mq.Message{Value: data})
	if err != nil {
		return fmt.Errorf("向topic=%s发送event=%#v失败: %w", p.topic, evt, err)
	}
	return nil
}
