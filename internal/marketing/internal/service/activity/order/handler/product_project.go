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

package handler

import (
	"context"

	"github.com/ecodeclub/webook/internal/marketing/internal/event"
	"github.com/ecodeclub/webook/internal/marketing/internal/event/producer"
)

var _ OrderHandler = (*ProductProjectHandler)(nil)

type ProductProjectHandler struct {
	permissionEventProducer producer.PermissionEventProducer
	creditEventProducer     producer.CreditEventProducer
}

func NewProductProjectHandler(permissionEventProducer producer.PermissionEventProducer, creditEventProducer producer.CreditEventProducer) *ProductProjectHandler {
	return &ProductProjectHandler{permissionEventProducer: permissionEventProducer, creditEventProducer: creditEventProducer}
}

func (h *ProductProjectHandler) Handle(ctx context.Context, info OrderInfo) error {
	ids := make([]int64, 0, len(info.Items))
	type Attrs struct {
		ProjectId int64 `json:"projectId"`
	}
	for _, item := range info.Items {
		var attrs Attrs
		err := item.SKU.UnmarshalAttrs(&attrs)
		if err != nil {
			return err
		}
		ids = append(ids, attrs.ProjectId)
	}
	return h.permissionEventProducer.Produce(ctx, event.PermissionEvent{
		Uid:    info.Order.BuyerID,
		Biz:    "project",
		BizIds: ids,
		Action: "购买项目商品",
	})
}
