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

var _ OrderHandler = (*ProductCreditHandler)(nil)

type ProductCreditHandler struct {
	producer producer.CreditEventProducer
}

func NewProductCreditHandler(producer producer.CreditEventProducer) *ProductCreditHandler {
	return &ProductCreditHandler{producer: producer}
}

func (p *ProductCreditHandler) Handle(ctx context.Context, info OrderInfo) error {
	type Attrs struct {
		Credit uint64 `json:"credit"`
	}
	var attr Attrs
	err := info.Items[0].SKU.UnmarshalAttrs(&attr)
	if err != nil {
		return err
	}
	return p.producer.Produce(ctx, event.CreditIncreaseEvent{
		// TODO 当下，我们购买积分的时候也允许用积分，这会导致这个 key 冲突
		// 即购买的时候下单用的也是 SN 作为 key
		Key:    info.Order.SN + "_incr",
		Uid:    info.Order.BuyerID,
		Amount: attr.Credit,
		Biz:    Biz,
		BizId:  info.Order.ID,
		Action: "购买积分",
	})
}
