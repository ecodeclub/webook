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
	"fmt"

	"github.com/ecodeclub/webook/internal/marketing/internal/event"
	"github.com/ecodeclub/webook/internal/marketing/internal/event/producer"
	"github.com/ecodeclub/webook/internal/marketing/internal/repository"
)

var _ OrderHandler = (*CodeProjectHandler)(nil)
var _ RedeemerHandler = (*CodeProjectHandler)(nil)

type CodeProjectHandler struct {
	baseCodeOrderHandler
	permissionEventProducer producer.PermissionEventProducer
	creditEventProducer     producer.CreditEventProducer
}

func NewCodeProjectHandler(repo repository.MarketingRepository, permissionEventProducer producer.PermissionEventProducer, creditEventProducer producer.CreditEventProducer, redemptionCodeGenerator func(id int64) string) *CodeProjectHandler {
	return &CodeProjectHandler{baseCodeOrderHandler: baseCodeOrderHandler{repo: repo, redemptionCodeGenerator: redemptionCodeGenerator}, permissionEventProducer: permissionEventProducer, creditEventProducer: creditEventProducer}
}

func (h *CodeProjectHandler) Handle(ctx context.Context, info OrderInfo) error {
	return h.baseCodeOrderHandler.Handle(ctx, info)
}

func (h *CodeProjectHandler) Redeem(ctx context.Context, info RedeemInfo) error {
	type Attrs struct {
		ProjectId int64 `json:"projectId"`
	}
	var attrs Attrs
	err := h.unmarshalAttrs(info.Code, &attrs)
	if err != nil {
		return fmt.Errorf("解析项目兑换码属性失败: %w, codeID:%d", err, info.Code.ID)
	}
	evt := event.PermissionEvent{
		Uid:    info.RedeemerID,
		Biz:    info.Code.Type,
		BizIds: []int64{attrs.ProjectId},
		Action: "兑换项目商品",
	}
	return h.permissionEventProducer.Produce(ctx, evt)
}
