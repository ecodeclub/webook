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
	"github.com/ecodeclub/webook/internal/marketing/internal/service/handler/order"
)

type HandlerRegistry struct {
	orderHandlers map[SPUCategory]map[SPUCategory]order.OrderHandler
}

func NewOrderHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		orderHandlers: make(map[SPUCategory]map[SPUCategory]order.OrderHandler),
	}
}

func (r *HandlerRegistry) Register(category0 SPUCategory, category1 SPUCategory, h order.OrderHandler) {
	if r.orderHandlers[category0] == nil {
		r.orderHandlers[category0] = make(map[SPUCategory]order.OrderHandler)
	}
	r.orderHandlers[category0][category1] = h
}

func (r *HandlerRegistry) Get(category0 SPUCategory, category1 SPUCategory) (order.OrderHandler, bool) {
	if handlersByType, ok := r.orderHandlers[category0]; ok {
		h, ok := handlersByType[category1]
		return h, ok
	}
	return nil, false
}
