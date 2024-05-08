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

package producer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/marketing/internal/event"
)

//go:generate mockgen -source=./credit_event_producer.go -package=evtmocks -destination=../mocks/credit.mock.go -typed CreditEventProducer
type CreditEventProducer interface {
	Produce(ctx context.Context, evt event.CreditIncreaseEvent) error
}

type creditEventProducer struct {
	producer mq.Producer
}

func NewCreditEventProducer(q mq.MQ) (CreditEventProducer, error) {
	producer, err := q.Producer(event.PermissionEventName)
	if err != nil {
		return nil, err
	}
	return &creditEventProducer{
		producer: producer,
	}, nil
}

func (s *creditEventProducer) Produce(ctx context.Context, evt event.CreditIncreaseEvent) error {
	data, err := json.Marshal(&evt)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	_, err = s.producer.Produce(ctx, &mq.Message{
		Key:   []byte(evt.Key),
		Value: data,
	})
	return err
}
