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
	"encoding/json"
	"fmt"

	"github.com/ecodeclub/webook/internal/marketing/internal/event"
	"github.com/ecodeclub/webook/internal/marketing/internal/event/producer"
)

var _ OrderHandler = (*ProductMemberHandler)(nil)

type ProductMemberHandler struct {
	memberEventProducer producer.MemberEventProducer
	creditEventProducer producer.CreditEventProducer
}

func NewProductMemberHandler(memberEventProducer producer.MemberEventProducer, creditEventProducer producer.CreditEventProducer) *ProductMemberHandler {
	return &ProductMemberHandler{memberEventProducer: memberEventProducer, creditEventProducer: creditEventProducer}
}

func (h *ProductMemberHandler) Handle(ctx context.Context, info OrderInfo) error {
	type Attrs struct {
		Days uint64 `json:"days,omitempty"`
	}
	var days uint64
	for _, item := range info.Items {
		var attrs Attrs
		err := json.Unmarshal([]byte(item.SKU.Attrs), &attrs)
		if err != nil {
			return fmt.Errorf("解析会员商品属性失败: %w, oid: %d, skuid:%d, attrs: %s",
				err, info.Order.ID, item.SKU.ID, item.SKU.Attrs)
		}
		days += attrs.Days * uint64(item.SKU.Quantity)
	}
	return h.memberEventProducer.Produce(ctx, event.MemberEvent{
		Key:    info.Order.SN,
		Uid:    info.Order.BuyerID,
		Days:   days,
		Biz:    Biz,
		BizId:  info.Order.ID,
		Action: "购买会员商品",
	})
}
