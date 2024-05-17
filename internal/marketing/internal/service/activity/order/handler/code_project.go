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
	"log"

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
	log.Printf("project code handle + ")
	return h.baseCodeOrderHandler.Handle(ctx, info)
}

func (h *CodeProjectHandler) Redeem(ctx context.Context, info RedeemInfo) error {
	evt := event.PermissionEvent{
		Uid:    info.RedeemerID,
		Biz:    info.Code.Type,
		BizIds: []int64{info.Code.Attrs.SKU.ID},
		Action: "兑换项目商品",
	}
	log.Printf("codeProduct sendProjectEvent = %#v\n", evt)
	return h.permissionEventProducer.Produce(ctx, evt)
}
