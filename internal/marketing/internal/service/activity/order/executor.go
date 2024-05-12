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
	order2 "github.com/ecodeclub/webook/internal/marketing/internal/service/handler/order"
	"github.com/ecodeclub/webook/internal/order"
)

type ActivityExecutor struct {
	orderSvc        order.Service
	handlerRegistry *HandlerRegistry
}

func NewOrderActivityExecutor(
	orderSvc order.Service,
	handlerRegistry *HandlerRegistry,
) *ActivityExecutor {
	return &ActivityExecutor{
		orderSvc:        orderSvc,
		handlerRegistry: handlerRegistry,
	}
}

func (s *ActivityExecutor) Execute(ctx context.Context, act domain.OrderCompletedActivity) error {
	o, err := s.orderSvc.FindUserVisibleOrderByUIDAndSN(ctx, act.BuyerID, act.OrderSN)
	if err != nil {
		return err
	}

	categorizedItems := NewCategorizedItems()
	for _, item := range o.Items {
		categorizedItems.AddItem(SPUCategory(item.SPU.Category), SPUType(item.SPU.Type), item)
	}

	for category, types := range categorizedItems.CategoriesAndTypes() {
		for itemType := range types {
			items := categorizedItems.GetItems(category, itemType)
			h, ok := s.handlerRegistry.Get(category, itemType)
			if !ok {
				return fmt.Errorf("未知 %s 类别 %s 类型订单处理器", category, itemType)
			}
			if er := h.Handle(ctx, order2.OrderInfo{Order: o, Items: items}); er != nil {
				return fmt.Errorf("处理 %s 类别 %s 类型商品失败: %w", category, itemType, er)
			}
		}
	}
	return nil
}
