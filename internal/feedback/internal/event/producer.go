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

//go:generate mockgen -source=./producer.go -package=evtmocks -destination=./mocks/producer.mock.go -typed IncreaseCreditsEventProducer
type IncreaseCreditsEventProducer interface {
	Produce(ctx context.Context, evt CreditIncreaseEvent) error
}

type increaseCreditsEventProducer struct {
	producer mq.Producer
}

func NewIncreaseCreditsEventProducer(q mq.MQ) (IncreaseCreditsEventProducer, error) {
	producer, err := q.Producer(creditIncreaseEvents)
	if err != nil {
		return nil, err
	}
	return &increaseCreditsEventProducer{producer: producer}, nil
}

func (p *increaseCreditsEventProducer) Produce(ctx context.Context, evt CreditIncreaseEvent) error {
	data, err := json.Marshal(&evt)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	_, err = p.producer.Produce(ctx, &mq.Message{Value: data})
	if err != nil {
		return fmt.Errorf("发送反馈成功消息失败: %w", err)
	}
	return nil
}
