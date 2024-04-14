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

	"github.com/ecodeclub/webook/internal/order/internal/domain"
	"github.com/ecodeclub/webook/internal/order/internal/repository"
	"golang.org/x/sync/errgroup"
)

type Service interface {
	// CreateOrder 创建订单 web调用
	CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error)
	// UpdateOrderPaymentIDAndPaymentSN 更新订单冗余支付ID及SN字段 web调用
	UpdateOrderPaymentIDAndPaymentSN(ctx context.Context, uid, oid, pid int64, psn string) error
	// FindOrderByUIDAndOrderSN 查找订单 web调用
	FindOrderByUIDAndOrderSN(ctx context.Context, uid int64, orderSN string) (domain.Order, error)
	// FindOrdersByUID 分页查找用户订单 web调用
	FindOrdersByUID(ctx context.Context, uid int64, offset, limit int) ([]domain.Order, int64, error)
	// CancelOrder 取消订单 web 调用
	CancelOrder(ctx context.Context, uid, oid int64) error

	// CompleteOrder 完成订单 event调用
	CompleteOrder(ctx context.Context, uid, oid int64) error
	// FindExpiredOrders 查询过期订单 job调用
	FindExpiredOrders(ctx context.Context, offset, limit int, ctime int64) ([]domain.Order, int64, error)
	// CloseExpiredOrders 关闭过期订单 job调用
	CloseExpiredOrders(ctx context.Context, orderIDs []int64, ctime int64) error
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

func (s *service) UpdateOrderPaymentIDAndPaymentSN(ctx context.Context, uid, oid, pid int64, psn string) error {
	return s.repo.UpdateOrderPaymentIDAndPaymentSN(ctx, uid, oid, pid, psn)
}

func (s *service) FindOrderByUIDAndOrderSN(ctx context.Context, buyerID int64, orderSN string) (domain.Order, error) {
	return s.repo.FindOrderUIDAndSN(ctx, buyerID, orderSN)
}

func (s *service) FindOrdersByUID(ctx context.Context, uid int64, offset, limit int) ([]domain.Order, int64, error) {
	var (
		eg    errgroup.Group
		os    []domain.Order
		total int64
	)
	eg.Go(func() error {
		var err error
		os, err = s.repo.FindOrdersByUID(ctx, uid, offset, limit)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.TotalOrders(ctx, uid)
		return err
	})
	return os, total, eg.Wait()
}

func (s *service) CancelOrder(ctx context.Context, uid, oid int64) error {
	return s.repo.CancelOrder(ctx, uid, oid)
}

func (s *service) CompleteOrder(ctx context.Context, uid, oid int64) error {
	// 已收到用户付款,不管订单状态为什么一律标记为“已完成”
	return s.repo.CompleteOrder(ctx, uid, oid)
}

func (s *service) FindExpiredOrders(ctx context.Context, offset, limit int, ctime int64) ([]domain.Order, int64, error) {
	var (
		eg    errgroup.Group
		os    []domain.Order
		total int64
	)
	eg.Go(func() error {
		var err error
		os, err = s.repo.FindExpiredOrders(ctx, offset, limit, ctime)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.TotalExpiredOrders(ctx, ctime)
		return err
	})
	return os, total, eg.Wait()
}

func (s *service) CloseExpiredOrders(ctx context.Context, orderIDs []int64, ctime int64) error {
	return s.repo.CloseExpiredOrders(ctx, orderIDs, ctime)
}
