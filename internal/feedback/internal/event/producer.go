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

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/pkg/mqx"
)

//go:generate mockgen -source=./producer.go -package=evtmocks -destination=./mocks/producer.mock.go -typed IncreaseCreditsEventProducer
type IncreaseCreditsEventProducer interface {
	Produce(ctx context.Context, evt CreditIncreaseEvent) error
}

func NewIncreaseCreditsEventProducer(q mq.MQ) (IncreaseCreditsEventProducer, error) {
	return mqx.NewGeneralProducer[CreditIncreaseEvent](q, creditIncreaseEvents)
}
