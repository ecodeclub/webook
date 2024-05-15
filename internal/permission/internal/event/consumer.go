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

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/permission/internal/service"
	"github.com/gotomicro/ego/core/elog"
)

type PermissionEventConsumer struct {
	svc      service.Service
	consumer mq.Consumer
	logger   *elog.Component
}

func NewPermissionEventConsumer(svc service.Service, q mq.MQ) (*PermissionEventConsumer, error) {
	groupID := "permission-personal"
	consumer, err := q.Consumer(PermissionEventName, groupID)
	if err != nil {
		return nil, err
	}
	return &PermissionEventConsumer{
		svc:      svc,
		consumer: consumer,
		logger:   elog.DefaultLogger,
	}, nil

}

// Start 后面要考虑借助 ctx 来优雅退出
func (c *PermissionEventConsumer) Start(ctx context.Context) {
	go func() {
		for {
			err := c.Consume(ctx)
			if err != nil {
				c.logger.Error("消费权限事件失败", elog.FieldErr(err))
			}
		}
	}()
}

func (c *PermissionEventConsumer) Consume(ctx context.Context) error {

	msg, err := c.consumer.Consume(ctx)
	if err != nil {
		return err
	}

	var evt PermissionEvent
	err = json.Unmarshal(msg.Value, &evt)
	if err != nil {
		return err
	}

	return c.svc.CreatePersonalPermission(ctx, evt.toDomain())
}
