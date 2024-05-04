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

//go:generate mockgen -source=./service.go -package=ordermocks -destination=../../mocks/order.mock.go -typed Service
type Service interface {
	// CreateOrder 创建订单 web调用
	CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error)
	// UpdateUnpaidOrderPaymentInfo 更新未支付订单冗余支付ID及SN字段 web调用
	UpdateUnpaidOrderPaymentInfo(ctx context.Context, uid, oid, pid int64, psn string) error
	// FindUserVisibleOrderByUIDAndSN 查找订单 web调用
	FindUserVisibleOrderByUIDAndSN(ctx context.Context, uid int64, orderSN string) (domain.Order, error)
	// FindUserVisibleOrdersByUID 分页查找用户订单 web调用
	FindUserVisibleOrdersByUID(ctx context.Context, uid int64, offset, limit int) ([]domain.Order, int64, error)
	// CancelOrder 取消订单 web 调用
	CancelOrder(ctx context.Context, uid, oid int64) error
	// SucceedOrder 订单支付失败 event调用
	SucceedOrder(ctx context.Context, uid int64, orderSN string) error
	// FailOrder 订单支付失败 event调用
	FailOrder(ctx context.Context, uid int64, orderSN string) error
	// FindTimeoutOrders 查询过期订单 job调用
	FindTimeoutOrders(ctx context.Context, offset, limit int, ctime int64) ([]domain.Order, int64, error)
	// CloseTimeoutOrders 关闭过期订单 job调用
	CloseTimeoutOrders(ctx context.Context, orderIDs []int64, ctime int64) error
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

func (s *service) UpdateUnpaidOrderPaymentInfo(ctx context.Context, uid, oid, pid int64, psn string) error {
	return s.repo.UpdateUnpaidOrderPaymentInfo(ctx, uid, oid, pid, psn)
}

func (s *service) FindUserVisibleOrderByUIDAndSN(ctx context.Context, buyerID int64, orderSN string) (domain.Order, error) {
	return s.repo.FindUserVisibleOrderByUIDAndSN(ctx, buyerID, orderSN)
}

func (s *service) FindUserVisibleOrdersByUID(ctx context.Context, uid int64, offset, limit int) ([]domain.Order, int64, error) {
	var (
		eg    errgroup.Group
		os    []domain.Order
		total int64
	)
	eg.Go(func() error {
		var err error
		os, err = s.repo.FindUserVisibleOrdersByUID(ctx, uid, offset, limit)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.TotalUserVisibleOrders(ctx, uid)
		return err
	})
	return os, total, eg.Wait()
}

func (s *service) CancelOrder(ctx context.Context, uid, oid int64) error {
	return s.repo.CancelOrder(ctx, uid, oid)
}

func (s *service) SucceedOrder(ctx context.Context, uid int64, orderSN string) error {
	// 已收到用户付款,不管订单状态为什么一律标记为“已完成”
	return s.repo.SucceedOrder(ctx, uid, orderSN)
}
func (s *service) FailOrder(ctx context.Context, uid int64, orderSN string) error {
	return s.repo.FailOrder(ctx, uid, orderSN)
}

func (s *service) FindTimeoutOrders(ctx context.Context, offset, limit int, ctime int64) ([]domain.Order, int64, error) {
	var (
		eg    errgroup.Group
		os    []domain.Order
		total int64
	)
	eg.Go(func() error {
		var err error
		os, err = s.repo.FindTimeoutOrders(ctx, offset, limit, ctime)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.TotalTimeoutOrders(ctx, ctime)
		return err
	})
	return os, total, eg.Wait()
}

func (s *service) CloseTimeoutOrders(ctx context.Context, orderIDs []int64, ctime int64) error {
	return s.repo.CloseTimeoutOrders(ctx, orderIDs, ctime)
}
