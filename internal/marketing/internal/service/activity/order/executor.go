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

package order

import (
	"context"
	"fmt"

	"github.com/ecodeclub/webook/internal/marketing/internal/domain"
	"github.com/ecodeclub/webook/internal/marketing/internal/event/producer"
	"github.com/ecodeclub/webook/internal/marketing/internal/repository"
	"github.com/ecodeclub/webook/internal/marketing/internal/service/activity/order/handler"
	"github.com/ecodeclub/webook/internal/order"
)

type ActivityExecutor struct {
	orderSvc        order.Service
	handlerRegistry *HandlerRegistry
}

func NewOrderActivityExecutor(
	repo repository.MarketingRepository,
	orderSvc order.Service,
	redemptionCodeGenerator func(id int64) string,
	memberEventProducer producer.MemberEventProducer,
	creditEventProducer producer.CreditEventProducer,
	permissionEventProducer producer.PermissionEventProducer,
) *ActivityExecutor {

	registry := NewHandlerRegistry()
	registry.RegisterOrderHandler("product", "member", handler.NewProductMemberHandler(memberEventProducer, creditEventProducer))

	codeMemberHandler := handler.NewCodeMemberHandler(repo, memberEventProducer, creditEventProducer, redemptionCodeGenerator)
	registry.RegisterOrderHandler("code", "member", codeMemberHandler)
	registry.RegisterRedeemerHandler("member", codeMemberHandler)

	return &ActivityExecutor{
		orderSvc:        orderSvc,
		handlerRegistry: registry,
	}
}

func (s *ActivityExecutor) Execute(ctx context.Context, act domain.OrderCompletedActivity) error {
	o, err := s.orderSvc.FindUserVisibleOrderByUIDAndSN(ctx, act.BuyerID, act.OrderSN)
	if err != nil {
		return err
	}

	categorizedItems := NewCategorizedItems()
	for _, item := range o.Items {
		categorizedItems.AddItem(SPUCategory(item.SPU.Category0), SPUCategory(item.SPU.Category1), item)
	}

	for category0, category1Set := range categorizedItems.CategoriesAndTypes() {
		for category1 := range category1Set {
			items := categorizedItems.GetItems(category0, category1)
			h, ok := s.handlerRegistry.GetOrderHandler(category0, category1)
			if !ok {
				return fmt.Errorf("未知 %s 类别0 %s 类别1订单处理器", category0, category1)
			}
			if er := h.Handle(ctx, handler.OrderInfo{Order: o, Items: items}); er != nil {
				return fmt.Errorf("处理 %s 类别0 %s 类别1商品失败: %w", category0, category1, er)
			}
		}
	}
	return nil
}

func (s *ActivityExecutor) Redeem(ctx context.Context, redeemerID int64, r domain.RedemptionCode) error {
	h, ok := s.handlerRegistry.GetRedeemerHandler(SPUCategory(r.Type))
	if !ok {
		return fmt.Errorf("未知兑换处理器: category1=%s", SPUCategory(r.Type))
	}
	return h.Redeem(ctx, handler.RedeemInfo{RedeemerID: redeemerID, Code: r})
}
