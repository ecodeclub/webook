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

	"github.com/ecodeclub/webook/internal/marketing/internal/domain"
	"github.com/ecodeclub/webook/internal/marketing/internal/repository"
)

// baseCodeOrderHandler 兑换码类型的订单处理器的公共父类
type baseCodeOrderHandler struct {
	repo                    repository.MarketingRepository
	redemptionCodeGenerator func(id int64) string
}

func (b *baseCodeOrderHandler) Handle(ctx context.Context, info OrderInfo) error {
	codes := make([]domain.RedemptionCode, 0, len(info.Items))
	for _, item := range info.Items {
		for i := int64(0); i < item.SKU.Quantity; i++ {
			codes = append(codes, domain.RedemptionCode{
				OwnerID: info.Order.BuyerID,
				Biz:     "order",
				BizId:   info.Order.ID,
				Type:    item.SPU.Category1,
				Attrs: domain.CodeAttrs{
					SKU: domain.SKU{
						ID:    item.SKU.ID,
						SN:    item.SKU.SN,
						Name:  item.SKU.Name,
						Attrs: item.SKU.Attrs,
					},
				},
				Code:   b.redemptionCodeGenerator(info.Order.BuyerID),
				Status: domain.RedemptionCodeStatusUnused,
			})
		}
	}
	log.Printf("base codes = %#v\n", codes)
	_, err := b.repo.CreateRedemptionCodes(ctx, codes)
	return err
}
