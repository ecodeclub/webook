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

const (
	SyncTopic = "sync_data_to_search"
)

type SyncDataToSearchEventProducer interface {
	Produce(ctx context.Context, evt QuestionEvent) error
}

func NewSyncEventProducer(q mq.MQ) (SyncDataToSearchEventProducer, error) {
	return mqx.NewGeneralProducer[QuestionEvent](q, SyncTopic)
}
