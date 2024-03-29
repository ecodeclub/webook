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

package repository

import (
	"context"
	"fmt"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/order/internal/domain"
	"github.com/ecodeclub/webook/internal/order/internal/repository/dao"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error)
	UpdateOrder(ctx context.Context, order domain.Order) error
	FindOrderBySN(ctx context.Context, sn string) (domain.Order, error)
	FindOrderBySNAndBuyerID(ctx context.Context, sn string, buyerID int64) (domain.Order, error)

	TotalOrders(ctx context.Context, uid int64) (int64, error)
	ListOrdersByUID(ctx context.Context, offset, limit int, uid int64) ([]domain.Order, error)

	TotalExpiredOrders(ctx context.Context, ctime int64) (int64, error)
	ListExpiredOrders(ctx context.Context, offset, limit int, ctime int64) ([]domain.Order, error)
	CloseExpiredOrders(ctx context.Context, orderIDs []int64) error
}

func NewRepository(d dao.OrderDAO) OrderRepository {
	return &orderRepository{
		dao: d,
	}
}

type orderRepository struct {
	dao dao.OrderDAO
}

func (o *orderRepository) CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	oid, err := o.dao.CreateOrder(ctx, o.toOrderEntity(order), o.toOrderItemEntities(order.Items))
	if err != nil {
		return domain.Order{}, err
	}
	order.ID = oid
	return order, nil
}

func (o *orderRepository) toOrderEntity(order domain.Order) dao.Order {
	return dao.Order{
		Id:                 order.ID,
		SN:                 order.SN,
		BuyerId:            order.BuyerID,
		PaymentId:          order.PaymentID,
		PaymentSn:          order.PaymentSN,
		OriginalTotalPrice: order.OriginalTotalPrice,
		RealTotalPrice:     order.RealTotalPrice,
		ClosedAt:           order.ClosedAt,
		Status:             order.Status,
	}
}

func (o *orderRepository) toOrderItemEntities(orderItems []domain.OrderItem) []dao.OrderItem {
	return slice.Map(orderItems, func(idx int, src domain.OrderItem) dao.OrderItem {
		return dao.OrderItem{
			SPUId:            src.SPUID,
			SKUId:            src.SKUID,
			SKUName:          src.SKUName,
			SKUDescription:   src.SKUDescription,
			SKUOriginalPrice: src.SKUOriginalPrice,
			SKURealPrice:     src.SKURealPrice,
			Quantity:         src.Quantity,
		}
	})
}

func (o *orderRepository) UpdateOrder(ctx context.Context, order domain.Order) error {
	return o.dao.UpdateOrder(ctx, o.toOrderEntity(order))
}

func (o *orderRepository) FindOrderBySN(ctx context.Context, sn string) (domain.Order, error) {
	order, err := o.dao.FindOrderBySN(ctx, sn)
	if err != nil {
		return domain.Order{}, err
	}
	orderItems, err := o.dao.FindOrderItemsByOrderID(ctx, order.Id)
	if err != nil {
		return domain.Order{}, err
	}
	return o.toOrderDomain(order, orderItems), nil
}

func (o *orderRepository) toOrderDomain(order dao.Order, orderItems []dao.OrderItem) domain.Order {
	return domain.Order{
		ID:                 order.Id,
		SN:                 order.SN,
		BuyerID:            order.BuyerId,
		PaymentID:          order.PaymentId,
		PaymentSN:          order.PaymentSn,
		OriginalTotalPrice: order.OriginalTotalPrice,
		RealTotalPrice:     order.RealTotalPrice,
		ClosedAt:           order.ClosedAt,
		Status:             order.Status,
		Items: slice.Map(orderItems, func(idx int, src dao.OrderItem) domain.OrderItem {
			return domain.OrderItem{
				OrderID:          src.OrderId,
				SPUID:            src.SPUId,
				SKUID:            src.SKUId,
				SKUName:          src.SKUName,
				SKUDescription:   src.SKUDescription,
				SKUOriginalPrice: src.SKUOriginalPrice,
				SKURealPrice:     src.SKURealPrice,
				Quantity:         src.Quantity,
			}
		}),
		Ctime: order.Ctime,
		Utime: order.Utime,
	}
}

func (o *orderRepository) FindOrderBySNAndBuyerID(ctx context.Context, sn string, buyerID int64) (domain.Order, error) {
	order, err := o.dao.FindOrderBySNAndBuyerID(ctx, sn, buyerID)
	if err != nil {
		return domain.Order{}, fmt.Errorf("通过订单序列号及买家ID查找订单失败: %w", err)
	}

	orderItems, err := o.dao.FindOrderItemsByOrderID(ctx, order.Id)
	if err != nil {
		return domain.Order{}, fmt.Errorf("通过订单ID查找订单失败: %w", err)
	}
	return o.toOrderDomain(order, orderItems), nil
}

func (o *orderRepository) TotalOrders(ctx context.Context, uid int64) (int64, error) {
	return o.dao.CountOrdersByUID(ctx, uid)
}

func (o *orderRepository) ListOrdersByUID(ctx context.Context, offset, limit int, uid int64) ([]domain.Order, error) {
	os, err := o.dao.ListOrdersByUID(ctx, offset, limit, uid)
	if err != nil {
		return nil, err
	}
	return slice.Map(os, func(idx int, src dao.Order) domain.Order {
		items, er := o.dao.FindOrderItemsByOrderID(ctx, src.Id)
		if er != nil {
			return domain.Order{}
		}
		return o.toOrderDomain(src, items)
	}), err
}

func (o *orderRepository) TotalExpiredOrders(ctx context.Context, ctime int64) (int64, error) {
	return o.dao.CountExpiredOrders(ctx, ctime)
}

func (o *orderRepository) ListExpiredOrders(ctx context.Context, offset, limit int, ctime int64) ([]domain.Order, error) {
	os, err := o.dao.ListExpiredOrders(ctx, offset, limit, ctime)
	if err != nil {
		return nil, err
	}
	return slice.Map(os, func(idx int, src dao.Order) domain.Order {
		return o.toOrderDomain(src, nil)
	}), err
}

func (o *orderRepository) CloseExpiredOrders(ctx context.Context, orderIDs []int64) error {
	return o.dao.UpdateExpiredOrders(ctx, orderIDs)
}
