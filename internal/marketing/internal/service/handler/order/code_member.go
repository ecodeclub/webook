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
	"encoding/json"
	"fmt"
	"log"

	"github.com/ecodeclub/webook/internal/marketing/internal/domain"
	"github.com/ecodeclub/webook/internal/marketing/internal/event"
	"github.com/ecodeclub/webook/internal/marketing/internal/event/producer"
	"github.com/ecodeclub/webook/internal/marketing/internal/repository"
)

var _ OrderHandler = (*CodeMemberHandler)(nil)
var _ RedeemerHandler = (*CodeMemberHandler)(nil)

type CodeMemberHandler struct {
	repo                    repository.MarketingRepository
	memberEventProducer     producer.MemberEventProducer
	creditEventProducer     producer.CreditEventProducer
	redemptionCodeGenerator func(id int64) string
}

func NewCodeMemberHandler(repo repository.MarketingRepository, memberEventProducer producer.MemberEventProducer, creditEventProducer producer.CreditEventProducer, redemptionCodeGenerator func(id int64) string) *CodeMemberHandler {
	return &CodeMemberHandler{repo: repo, memberEventProducer: memberEventProducer, creditEventProducer: creditEventProducer, redemptionCodeGenerator: redemptionCodeGenerator}
}

func (h *CodeMemberHandler) Handle(ctx context.Context, info OrderInfo) error {

	codes := make([]domain.RedemptionCode, 0, len(info.Items))
	for _, item := range info.Items {
		for i := int64(0); i < item.SKU.Quantity; i++ {
			codes = append(codes, domain.RedemptionCode{
				OwnerID: info.Order.BuyerID,
				Biz:     "order",
				BizId:   info.Order.ID,
				Type:    item.SPU.Category1,
				Attrs:   domain.CodeAttrs{SKU: domain.SKU{ID: item.SKU.ID, Attrs: item.SKU.Attrs}},
				Code:    h.redemptionCodeGenerator(info.Order.BuyerID),
				Status:  domain.RedemptionCodeStatusUnused,
			})
		}
	}
	log.Printf("member codes = %#v\n", codes)
	_, err := h.repo.CreateRedemptionCodes(ctx, codes)
	return err
}

func (h *CodeMemberHandler) Redeem(ctx context.Context, info RedeemInfo) error {
	type Attrs struct {
		Days uint64 `json:"days"`
	}
	var attrs Attrs
	skuAttrs := info.Code.Attrs.SKU.Attrs
	err := json.Unmarshal([]byte(skuAttrs), &attrs)
	if err != nil {
		return fmt.Errorf("解析会员兑换码属性失败: %w, codeID:%d, skuAttrs:%s", err, info.Code.ID, skuAttrs)
	}
	memberEvent := event.MemberEvent{
		Key:    fmt.Sprintf("code-member-%d", info.Code.ID),
		Uid:    info.RedeemerID,
		Days:   attrs.Days,
		Biz:    info.Code.Biz,
		BizId:  info.Code.BizId,
		Action: "兑换会员商品",
	}
	log.Printf("codeMember sendMemberEvent = %#v\n", memberEvent)
	return h.memberEventProducer.Produce(ctx, memberEvent)
}
