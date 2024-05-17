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
	"github.com/ecodeclub/webook/internal/marketing/internal/service/activity/order/handler"
)

type HandlerRegistry struct {
	orderHandlers    map[SPUCategory]map[SPUCategory]handler.OrderHandler
	redeemerHandlers map[SPUCategory]handler.RedeemerHandler
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		orderHandlers:    make(map[SPUCategory]map[SPUCategory]handler.OrderHandler),
		redeemerHandlers: make(map[SPUCategory]handler.RedeemerHandler),
	}
}

func (r *HandlerRegistry) RegisterOrderHandler(category0 SPUCategory, category1 SPUCategory, h handler.OrderHandler) {
	if r.orderHandlers[category0] == nil {
		r.orderHandlers[category0] = make(map[SPUCategory]handler.OrderHandler)
	}
	r.orderHandlers[category0][category1] = h
}

func (r *HandlerRegistry) GetOrderHandler(category0 SPUCategory, category1 SPUCategory) (handler.OrderHandler, bool) {
	if category1Set, ok := r.orderHandlers[category0]; ok {
		h, ok := category1Set[category1]
		return h, ok
	}
	return nil, false
}

func (r *HandlerRegistry) RegisterRedeemerHandler(category1 SPUCategory, h handler.RedeemerHandler) {
	r.redeemerHandlers[category1] = h
}

func (r *HandlerRegistry) GetRedeemerHandler(category1 SPUCategory) (handler.RedeemerHandler, bool) {
	h, ok := r.redeemerHandlers[category1]
	return h, ok
}
