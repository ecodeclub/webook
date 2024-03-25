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
	"fmt"
	"time"

	"github.com/ecodeclub/webook/internal/order/internal/domain"
	"github.com/ecodeclub/webook/internal/order/internal/repository"
	"golang.org/x/sync/errgroup"
)

type Service interface {
	CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error)
	FindOrder(ctx context.Context, orderSN string, buyerID int64) (domain.Order, error)
	UpdateOrder(ctx context.Context, order domain.Order) error
	CompleteOrder(ctx context.Context, order domain.Order) error
	ListOrders(ctx context.Context, offset, limit int, uid int64) ([]domain.Order, int64, error)
	ListExpiredOrders(ctx context.Context, offset, limit int, ctime int64) ([]domain.Order, int64, error)
	CloseExpiredOrders(ctx context.Context, orderIDs []int64) error
	CancelOrder(ctx context.Context, order domain.Order) error
}

func NewService(repo repository.OrderRepository) Service {
	return &service{repo: repo}
}

type service struct {
	repo repository.OrderRepository
}

func (s *service) CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	return s.repo.CreateOrder(ctx, order)
}

func (s *service) FindOrder(ctx context.Context, orderSN string, buyerID int64) (domain.Order, error) {
	return s.repo.FindOrderBySNAndBuyerID(ctx, orderSN, buyerID)
}

func (s *service) UpdateOrder(ctx context.Context, order domain.Order) error {
	return s.repo.UpdateOrder(ctx, order)
}

func (s *service) CompleteOrder(ctx context.Context, order domain.Order) error {
	// 已收到用户付款,不管订单状态为什么一律标记为“已完成”
	order.Status = domain.OrderStatusCompleted
	return s.repo.UpdateOrder(ctx, order)
}

func (s *service) ListOrders(ctx context.Context, offset, limit int, uid int64) ([]domain.Order, int64, error) {
	var (
		eg    errgroup.Group
		os    []domain.Order
		total int64
	)
	eg.Go(func() error {
		var err error
		os, err = s.repo.ListOrdersByUID(ctx, offset, limit, uid)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.TotalOrders(ctx, uid)
		return err
	})
	return os, total, eg.Wait()
}

func (s *service) ListExpiredOrders(ctx context.Context, offset, limit int, ctime int64) ([]domain.Order, int64, error) {
	var (
		eg    errgroup.Group
		os    []domain.Order
		total int64
	)
	eg.Go(func() error {
		var err error
		os, err = s.repo.ListExpiredOrders(ctx, offset, limit, ctime)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.TotalExpiredOrders(ctx, ctime)
		return err
	})
	return os, total, eg.Wait()
}

func (s *service) CloseExpiredOrders(ctx context.Context, orderIDs []int64) error {
	return s.repo.CloseExpiredOrders(ctx, orderIDs)
}

func (s *service) CancelOrder(ctx context.Context, order domain.Order) error {
	if order.Status != domain.OrderStatusUnpaid {
		return fmt.Errorf("订单状态非法")
	}
	order.Status = domain.OrderStatusCanceled
	order.ClosedAt = time.Now().UnixMilli()
	return s.repo.UpdateOrder(ctx, order)
}
