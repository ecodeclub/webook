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
	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/order/internal/domain"
	"github.com/ecodeclub/webook/internal/order/internal/repository/dao"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error)
	UpdateOrderPaymentIDAndPaymentSN(ctx context.Context, uid, oid, pid int64, psn string) error
	FindOrderUIDAndSN(ctx context.Context, uid int64, sn string) (domain.Order, error)
	TotalOrders(ctx context.Context, uid int64) (int64, error)
	FindOrdersByUID(ctx context.Context, uid int64, offset, limit int) ([]domain.Order, error)
	CancelOrder(ctx context.Context, uid, oid int64) error
	CompleteOrder(ctx context.Context, uid int64, oid int64) error

	FindTimeoutOrders(ctx context.Context, offset, limit int, ctime int64) ([]domain.Order, error)
	TotalTimeoutOrders(ctx context.Context, ctime int64) (int64, error)
	CloseTimeoutOrders(ctx context.Context, orderIDs []int64, ctime int64) error
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
		Id:               order.ID,
		SN:               order.SN,
		BuyerId:          order.BuyerID,
		PaymentId:        sqlx.NewNullInt64(order.Payment.ID),
		PaymentSn:        sqlx.NewNullString(order.Payment.SN),
		OriginalTotalAmt: order.OriginalTotalAmt,
		RealTotalAmt:     order.RealTotalAmt,
		Status:           order.Status.ToUint8(),
	}
}

func (o *orderRepository) toOrderItemEntities(orderItems []domain.OrderItem) []dao.OrderItem {
	return slice.Map(orderItems, func(idx int, src domain.OrderItem) dao.OrderItem {
		return dao.OrderItem{
			SPUId:            src.SKU.SPUID,
			SKUId:            src.SKU.ID,
			SKUSN:            src.SKU.SN,
			SKUName:          src.SKU.Name,
			SKUImage:         src.SKU.Image,
			SKUDescription:   src.SKU.Description,
			SKUOriginalPrice: src.SKU.OriginalPrice,
			SKURealPrice:     src.SKU.RealPrice,
			Quantity:         src.SKU.Quantity,
		}
	})
}

func (o *orderRepository) UpdateOrderPaymentIDAndPaymentSN(ctx context.Context, uid, oid, pid int64, psn string) error {
	return o.dao.UpdateOrderPaymentIDAndPaymentSN(ctx, uid, oid, pid, psn)
}

func (o *orderRepository) FindOrderUIDAndSN(ctx context.Context, uid int64, sn string) (domain.Order, error) {
	order, err := o.dao.FindOrderByUIDAndSN(ctx, uid, sn)
	if err != nil {
		return domain.Order{}, fmt.Errorf("通过订单序列号及买家ID查找订单失败: %w", err)
	}

	orderItems, err := o.dao.FindOrderItemsByOrderID(ctx, order.Id)
	if err != nil {
		return domain.Order{}, fmt.Errorf("通过订单ID查找订单失败: %w", err)
	}
	return o.toOrderDomain(order, orderItems), nil
}

func (o *orderRepository) toOrderDomain(order dao.Order, orderItems []dao.OrderItem) domain.Order {
	return domain.Order{
		ID:      order.Id,
		SN:      order.SN,
		BuyerID: order.BuyerId,
		Payment: domain.Payment{
			ID: order.PaymentId.Int64,
			SN: order.PaymentSn.String,
		},
		OriginalTotalAmt: order.OriginalTotalAmt,
		RealTotalAmt:     order.RealTotalAmt,
		Status:           domain.OrderStatus(order.Status),
		Items: slice.Map(orderItems, func(idx int, src dao.OrderItem) domain.OrderItem {
			return domain.OrderItem{
				SKU: domain.SKU{
					SPUID:         src.SPUId,
					ID:            src.SKUId,
					SN:            src.SKUSN,
					Image:         src.SKUImage,
					Name:          src.SKUName,
					Description:   src.SKUDescription,
					OriginalPrice: src.SKUOriginalPrice,
					RealPrice:     src.SKURealPrice,
					Quantity:      src.Quantity,
				},
			}
		}),
		Ctime: order.Ctime,
		Utime: order.Utime,
	}
}

func (o *orderRepository) TotalOrders(ctx context.Context, uid int64) (int64, error) {
	return o.dao.CountOrdersByUID(ctx, uid)
}

func (o *orderRepository) FindOrdersByUID(ctx context.Context, uid int64, offset, limit int) ([]domain.Order, error) {
	os, err := o.dao.FindOrdersByUID(ctx, uid, offset, limit)
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

func (o *orderRepository) CancelOrder(ctx context.Context, uid, oid int64) error {
	return o.dao.CancelOrder(ctx, uid, oid)
}

func (o *orderRepository) CompleteOrder(ctx context.Context, uid int64, oid int64) error {
	return o.dao.CompleteOrder(ctx, uid, oid)
}

func (o *orderRepository) FindTimeoutOrders(ctx context.Context, offset, limit int, ctime int64) ([]domain.Order, error) {
	os, err := o.dao.FindTimeoutOrders(ctx, offset, limit, ctime)
	if err != nil {
		return nil, err
	}
	return slice.Map(os, func(idx int, src dao.Order) domain.Order {
		return o.toOrderDomain(src, nil)
	}), err
}

func (o *orderRepository) TotalTimeoutOrders(ctx context.Context, ctime int64) (int64, error) {
	return o.dao.CountTimeoutOrders(ctx, ctime)
}

func (o *orderRepository) CloseTimeoutOrders(ctx context.Context, orderIDs []int64, ctime int64) error {
	return o.dao.CloseTimeoutOrders(ctx, orderIDs, ctime)
}
