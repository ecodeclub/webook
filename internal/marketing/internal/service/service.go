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

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/marketing/internal/domain"
	"github.com/ecodeclub/webook/internal/marketing/internal/event"
	"github.com/ecodeclub/webook/internal/marketing/internal/event/producer"
	"github.com/ecodeclub/webook/internal/marketing/internal/repository"
	"github.com/ecodeclub/webook/internal/order"
	"github.com/ecodeclub/webook/internal/product"
	"golang.org/x/sync/errgroup"
)

var (
	ErrRedemptionNotFound = repository.ErrRedemptionNotFound
	ErrRedemptionCodeUsed = repository.ErrRedemptionCodeUsed
)

type Service interface {
	ExecuteOrderCompletedActivity(ctx context.Context, act domain.OrderCompletedActivity) error
	RedeemRedemptionCode(ctx context.Context, uid int64, code string) error
	ListRedemptionCodes(ctx context.Context, uid int64, offset, list int) ([]domain.RedemptionCode, int64, error)
}

type skuAttrs struct {
	Days uint64 `json:"days,omitempty"`
}

type service struct {
	repo                    repository.MarketingRepository
	orderSvc                order.Service
	productSvc              product.Service
	redemptionCodeGenerator func(id int64) string
	eventKeyGenerator       func() string
	memberEventProducer     producer.MemberEventProducer
	creditEventProducer     producer.CreditEventProducer
	permissionEventProducer producer.PermissionEventProducer
}

func NewService(
	repo repository.MarketingRepository,
	orderSvc order.Service,
	productSvc product.Service,
	redemptionCodeGenerator func(id int64) string,
	eventKeyGenerator func() string,
	memberEventProducer producer.MemberEventProducer,
	creditEventProducer producer.CreditEventProducer,
	permissionEventProducer producer.PermissionEventProducer,
) Service {
	return &service{
		repo:                    repo,
		orderSvc:                orderSvc,
		productSvc:              productSvc,
		redemptionCodeGenerator: redemptionCodeGenerator,
		eventKeyGenerator:       eventKeyGenerator,
		memberEventProducer:     memberEventProducer,
		creditEventProducer:     creditEventProducer,
		permissionEventProducer: permissionEventProducer,
	}
}

func (s *service) ExecuteOrderCompletedActivity(ctx context.Context, act domain.OrderCompletedActivity) error {
	const (
		productCategory = "product"
		codeCategory    = "code"
	)
	o, err := s.orderSvc.FindUserVisibleOrderByUIDAndSN(ctx, act.BuyerID, act.OrderSN)
	if err != nil {
		return err
	}

	productItems := make([]order.Item, 0, len(o.Items))
	codeItems := make([]order.Item, 0, len(o.Items))

	_ = slice.FindAll(o.Items, func(src order.Item) bool {
		if src.SPU.Category == productCategory {
			productItems = append(productItems, src)
		} else if src.SPU.Category == codeCategory {
			codeItems = append(codeItems, src)
		}
		return false
	})

	if len(productItems) > 0 {
		err = s.handleMemberCategoryOrderItems(ctx, o, productItems)
		if err != nil {
			return fmt.Errorf("处理会员商品失败: %w", err)
		}
	}

	if len(codeItems) > 0 {
		err = s.handleCodeCategoryOrderItems(ctx, o, codeItems)
		if err != nil {
			return fmt.Errorf("处理兑换码商品失败: %w", err)
		}
	}

	return nil
}

func (s *service) handleMemberCategoryOrderItems(ctx context.Context, o order.Order, items []order.Item) error {
	var days uint64
	for _, item := range items {
		attrs, err := s.unmarshalSPUAttrs(item.SKU.Attrs)
		if err != nil {
			return fmt.Errorf("解析会员商品属性失败: %w, oid: %d, skuid:%d, attrs: %s", err, o.ID, item.SKU.ID, item.SKU.Attrs)
		}
		days += attrs.Days * uint64(item.SKU.Quantity)
	}
	return s.memberEventProducer.Produce(ctx, event.MemberEvent{
		Key:    o.SN,
		Uid:    o.BuyerID,
		Days:   days,
		Biz:    "order",
		BizId:  o.ID,
		Action: "购买会员商品",
	})
}

func (s *service) unmarshalSPUAttrs(attrs string) (skuAttrs, error) {
	var a skuAttrs
	err := json.Unmarshal([]byte(attrs), &a)
	return a, err
}

func (s *service) handleCodeCategoryOrderItems(ctx context.Context, o order.Order, items []order.Item) error {
	codes := make([]domain.RedemptionCode, 0, len(items))
	for _, item := range items {
		for i := int64(0); i < item.SKU.Quantity; i++ {
			codes = append(codes, domain.RedemptionCode{
				OwnerID:  o.BuyerID,
				OrderID:  o.ID,
				SPUID:    item.SPU.ID,
				SKUAttrs: item.SKU.Attrs,
				Code:     s.redemptionCodeGenerator(o.BuyerID),
				Status:   domain.RedemptionCodeStatusUnused,
			})
		}
	}
	log.Printf("codes = %#v\n", codes)
	_, err := s.repo.CreateRedemptionCodes(ctx, o.ID, codes)
	return err
}

func (s *service) RedeemRedemptionCode(ctx context.Context, uid int64, code string) error {
	r, err := s.repo.FindRedemptionCode(ctx, code)
	if err != nil {
		return err
	}
	if r.Status == domain.RedemptionCodeStatusUsed {
		return fmt.Errorf("%w: code:%s", ErrRedemptionCodeUsed, code)
	}
	err = s.repo.SetUnusedRedemptionCodeStatusUsed(ctx, uid, code)
	if err != nil {
		return err
	}
	return s.sendEvent(ctx, uid, r)
}

func (s *service) sendEvent(ctx context.Context, uid int64, code domain.RedemptionCode) error {
	// todo: 按照SPU分类执行不同的后续动作, 当前只支持发送会员消息
	return s.sendMemberEvent(ctx, uid, code)
}

func (s *service) sendMemberEvent(ctx context.Context, uid int64, code domain.RedemptionCode) error {
	attrs, err := s.unmarshalSPUAttrs(code.SKUAttrs)
	if err != nil {
		return fmt.Errorf("解析会员兑换码属性失败: %w, codeID:%d, spuAttrs:%s", err, code.ID, code.SKUAttrs)
	}
	memberEvent := event.MemberEvent{
		Key:    fmt.Sprintf("code-member-%d", code.ID),
		Uid:    uid,
		Days:   attrs.Days,
		Biz:    "order",
		BizId:  code.OrderID,
		Action: "兑换会员商品",
	}
	log.Printf("svc memberEvent = %#v\n", memberEvent)
	return s.memberEventProducer.Produce(ctx, memberEvent)
}

func (s *service) ListRedemptionCodes(ctx context.Context, uid int64, offset, list int) ([]domain.RedemptionCode, int64, error) {
	var (
		eg    errgroup.Group
		codes []domain.RedemptionCode
		total int64
	)
	eg.Go(func() error {
		var err error
		codes, err = s.repo.FindRedemptionCodesByUID(ctx, uid, offset, list)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.TotalRedemptionCodes(ctx, uid)
		return err
	})

	return codes, total, eg.Wait()
}
