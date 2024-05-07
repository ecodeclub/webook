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

	"github.com/ecodeclub/webook/internal/marketing/internal/domain"
	"github.com/ecodeclub/webook/internal/marketing/internal/event"
	"github.com/ecodeclub/webook/internal/marketing/internal/event/producer"
	"github.com/ecodeclub/webook/internal/marketing/internal/repository"
	"github.com/ecodeclub/webook/internal/order"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	"golang.org/x/sync/errgroup"
)

type Service interface {
	ExecuteOrderCompletedActivity(ctx context.Context, act domain.OrderCompletedActivity) error
	ListRedemptionCodes(ctx context.Context, uid int64, offset, list int) ([]domain.RedemptionCode, int64, error)
}

type service struct {
	orderSvc       order.Service
	memberProducer producer.MemberEventProducer
	codeGenerator  *sequencenumber.Generator
	repo           repository.MarketingRepository
}

func NewService(
	orderSvc order.Service,
	memberProducer producer.MemberEventProducer,
	codeGenerator *sequencenumber.Generator,
	repo repository.MarketingRepository,
) Service {
	return &service{
		orderSvc:       orderSvc,
		memberProducer: memberProducer,
		codeGenerator:  codeGenerator,
		repo:           repo,
	}
}

func (s *service) ExecuteOrderCompletedActivity(ctx context.Context, act domain.OrderCompletedActivity) error {
	o, err := s.orderSvc.FindUserVisibleOrderByUIDAndSN(ctx, act.BuyerID, act.OrderSN)
	if err != nil {
		return err
	}
	for _, item := range o.Items {
		if item.SPU.Category.Name == "member" {
			if er := s.handlerMemberOrder(ctx, o, item); er != nil {
				return fmt.Errorf("处理会员商品失败: %w", er)
			}
		}
	}
	return nil
}

func (s *service) handlerMemberOrder(ctx context.Context, o order.Order, item order.Item) error {
	type Attrs struct {
		Days uint64 `json:"days"`
	}
	var attrs Attrs
	err := json.Unmarshal([]byte(item.SKU.Attrs), &attrs)
	if err != nil {
		return fmt.Errorf("解析会员商品属性失败: %w, attrs: %s", err, item.SKU.Attrs)
	}
	return s.memberProducer.Produce(ctx, event.MemberEvent{
		Key:    o.SN,
		Uid:    o.BuyerID,
		Days:   attrs.Days * uint64(item.SKU.Quantity),
		Biz:    "order",
		BizId:  o.ID,
		Action: "购买会员商品",
	})
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
