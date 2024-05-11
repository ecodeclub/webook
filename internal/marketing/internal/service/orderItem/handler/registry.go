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

	"github.com/ecodeclub/webook/internal/marketing/internal/service/orderItem"
	"github.com/ecodeclub/webook/internal/order"
)

type OrderItemHandler interface {
	Handle(ctx context.Context, order order.Order, items []order.Item) error
}

type OrderItemHandlerRegistry struct {
	handlers map[orderItem.SPUCategory]map[orderItem.SPUType]OrderItemHandler
}

func NewOrderItemHandlerRegistry() *OrderItemHandlerRegistry {
	return &OrderItemHandlerRegistry{
		handlers: make(map[orderItem.SPUCategory]map[orderItem.SPUType]OrderItemHandler),
	}
}

func (r *OrderItemHandlerRegistry) Register(category orderItem.SPUCategory, itemType orderItem.SPUType, handler OrderItemHandler) {
	if r.handlers[category] == nil {
		r.handlers[category] = make(map[orderItem.SPUType]OrderItemHandler)
	}
	r.handlers[category][itemType] = handler
}

func (r *OrderItemHandlerRegistry) Get(category orderItem.SPUCategory, typ orderItem.SPUType) (OrderItemHandler, bool) {
	if handlersByType, ok := r.handlers[category]; ok {
		handler, exists := handlersByType[typ]
		return handler, exists
	}
	return nil, false
}

func (r *OrderItemHandlerRegistry) Handle(ctx context.Context, category orderItem.SPUCategory, itemType orderItem.SPUType, order order.Order, items []order.Item) error {
	handler, exists := r.Get(category, itemType)
	if !exists {
		return fmt.Errorf("未注册的订单项处理器: category: %s, type: %s", category, itemType)
	}
	return handler.Handle(ctx, order, items)
}
