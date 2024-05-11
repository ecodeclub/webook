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

	"github.com/ecodeclub/webook/internal/marketing/internal/event/producer"
	"github.com/ecodeclub/webook/internal/order"
)

var _ OrderItemHandler = (*ProductMemberHandler)(nil)

type ProductMemberHandler struct {
	memberEventProducer producer.MemberEventProducer
}

func NewProductMemberHandler(memberEventProducer producer.MemberEventProducer) *ProductMemberHandler {
	return &ProductMemberHandler{
		memberEventProducer: memberEventProducer,
	}
}

func (h *ProductMemberHandler) Handle(ctx context.Context, o order.Order, items []order.Item) error {
	return nil
}
