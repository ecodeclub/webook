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

package web

import (
	"context"
	"fmt"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/order/internal/domain"
	"github.com/ecodeclub/webook/internal/order/internal/service"
	"github.com/ecodeclub/webook/internal/payment"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	"github.com/ecodeclub/webook/internal/product"
	"github.com/gin-gonic/gin"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	svc         service.Service
	paymentSvc  payment.Service
	productSvc  product.Service
	creditSvc   credit.Service
	snGenerator *sequencenumber.Generator
	cache       ecache.Cache
}

func NewHandler(svc service.Service, paymentSvc payment.Service, productSvc product.Service, creditSvc credit.Service, snGenerator *sequencenumber.Generator, cache ecache.Cache) *Handler {
	return &Handler{svc: svc, paymentSvc: paymentSvc, productSvc: productSvc, creditSvc: creditSvc, snGenerator: snGenerator, cache: cache}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/order")
	g.POST("/preview", ginx.BS[PreviewOrderReq](h.RetrievePreviewOrder))
	g.POST("/create", ginx.BS[CreateOrderReq](h.CreateOrderAndPayment))
	g.POST("", ginx.BS[RetrieveOrderStatusReq](h.RetrieveOrderStatus))
	g.POST("/list", ginx.BS[ListOrdersReq](h.ListOrders))
	g.POST("/detail", ginx.BS[RetrieveOrderDetailReq](h.RetrieveOrderDetail))
	g.POST("/cancel", ginx.BS[CancelOrderReq](h.CancelOrder))
}

func (h *Handler) PublicRoutes(_ *gin.Engine) {}

// RetrievePreviewOrder 获取订单预览信息, 此时订单尚未创建
func (h *Handler) RetrievePreviewOrder(ctx *ginx.Context, req PreviewOrderReq, sess session.Session) (ginx.Result, error) {
	p, err := h.productSvc.FindSKUBySN(ctx.Request.Context(), req.SKUSN)
	if err != nil {
		return systemErrorResult, fmt.Errorf("商品SKU序列号非法: %w", err)
	}
	if req.Quantity < 1 || req.Quantity > p.SKUs[0].Stock {
		// todo: 重新审视stockLimit的意义及用法
		return systemErrorResult, fmt.Errorf("要购买的商品数量非法")
	}
	c, err := h.creditSvc.GetCreditsByUID(ctx.Request.Context(), sess.Claims().Uid)
	if err != nil {
		return systemErrorResult, fmt.Errorf("获取用户积分失败: %w", err)
	}
	return ginx.Result{
		Data: PreviewOrderResp{
			Credits:  c.TotalAmount,
			Payments: h.toPaymentChannelVO(ctx),
			SKUs: slice.Map(p.SKUs, func(idx int, src product.SKU) SKU {
				return SKU{
					SN:            src.SN,
					Image:         src.Image,
					Name:          src.Name,
					Desc:          src.Desc,
					OriginalPrice: src.Price,
					RealPrice:     src.Price, // 引入优惠券时, 需要获取用户的优惠信息,动态计算
					Quantity:      req.Quantity,
				}
			}),
			Policy: "请注意: 虚拟商品、一旦支持成功不退、不换,请谨慎操作",
		},
	}, nil
}

func (h *Handler) toPaymentChannelVO(ctx *ginx.Context) []PaymentItem {
	pcs := h.paymentSvc.GetPaymentChannels(ctx.Request.Context())
	channels := make([]PaymentItem, 0, len(pcs))
	for _, pc := range pcs {
		channels = append(channels, PaymentItem{Type: pc.Type})
	}
	return channels
}

// CreateOrderAndPayment 创建订单和支付
func (h *Handler) CreateOrderAndPayment(ctx *ginx.Context, req CreateOrderReq, sess session.Session) (ginx.Result, error) {

	if err := h.checkRequestID(ctx.Request.Context(), req.RequestID); err != nil {
		return systemErrorResult, fmt.Errorf("请求ID错误: %w", err)
	}

	order, err := h.createOrder(ctx, req, sess.Claims().Uid)
	if err != nil {
		// 创建订单失败
		return systemErrorResult, fmt.Errorf("创建订单失败: %w", err)
	}

	p, err := h.createPayment(ctx, order, req.Payments)
	if err != nil {
		// 创建支付失败
		return systemErrorResult, fmt.Errorf("创建支付失败: %w", err)
	}

	err = h.svc.UpdateOrderPaymentIDAndPaymentSN(ctx.Request.Context(), order.BuyerID, order.ID, p.ID, p.SN)
	if err != nil {
		return systemErrorResult, fmt.Errorf("订单冗余支付ID及SN失败: %w", err)
	}

	// 微信支付需要返回二维码URL
	var wechatCodeURL string
	for _, r := range p.Records {
		if payment.ChannelTypeWechat == r.Channel {
			wechatCodeURL = r.WechatCodeURL
		}
	}

	return ginx.Result{
		Data: CreateOrderResp{
			OrderSN:       order.SN,
			WechatCodeURL: wechatCodeURL,
		},
	}, nil
}

func (h *Handler) checkRequestID(ctx context.Context, requestID string) error {
	if requestID == "" {
		return fmt.Errorf("请求ID为空")
	}

	key := h.createOrderRequestKey(requestID)
	val := h.cache.Get(ctx, key)
	if !val.KeyNotFound() {
		return fmt.Errorf("重复请求")
	}
	if err := h.cache.Set(ctx, key, requestID, 0); err != nil {
		return fmt.Errorf("缓存请求ID失败: %w", err)
	}
	return nil
}

func (h *Handler) createOrderRequestKey(requestID string) string {
	return fmt.Sprintf("order:create:%s", requestID)
}

func (h *Handler) createOrder(ctx context.Context, req CreateOrderReq, buyerID int64) (domain.Order, error) {
	orderItems, originalTotalPrice, realTotalPrice, err := h.getOrderItems(ctx, req)
	if err != nil {
		return domain.Order{}, err
	}
	if originalTotalPrice != req.OriginalTotalPrice {
		return domain.Order{}, fmt.Errorf("商品总原价非法")
	}
	if realTotalPrice != req.RealTotalPrice {
		return domain.Order{}, fmt.Errorf("商品总实价非法")
	}

	orderSN, err := h.snGenerator.Generate(buyerID)
	if err != nil {
		return domain.Order{}, fmt.Errorf("生成订单序列号失败")
	}

	return h.svc.CreateOrder(ctx, domain.Order{
		SN:                 orderSN,
		BuyerID:            buyerID,
		OriginalTotalPrice: originalTotalPrice,
		RealTotalPrice:     realTotalPrice,
		Items:              orderItems,
	})
}

func (h *Handler) getOrderItems(ctx context.Context, req CreateOrderReq) ([]domain.OrderItem, int64, int64, error) {
	if len(req.SKUs) == 0 {
		return nil, 0, 0, fmt.Errorf("商品信息非法")
	}
	orderItems := make([]domain.OrderItem, 0, len(req.SKUs))
	originalTotalPrice, realTotalPrice := int64(0), int64(0)
	for _, p := range req.SKUs {
		pp, err := h.productSvc.FindSKUBySN(ctx, p.SN)
		if err != nil {
			// SN非法
			return nil, 0, 0, fmt.Errorf("商品SKUSN非法: %w", err)
		}
		if p.Quantity < 1 || p.Quantity > pp.SKUs[0].Stock {
			// todo: 重新审视stockLimit的意义及用法
			return nil, 0, 0, fmt.Errorf("商品数量非法")
		}

		item := domain.OrderItem{
			SKU: domain.SKU{
				SPUID:         pp.ID,
				ID:            pp.SKUs[0].ID,
				SN:            pp.SKUs[0].SN,
				Image:         pp.SKUs[0].Image,
				Name:          pp.SKUs[0].Name,
				Description:   pp.SKUs[0].Desc,
				OriginalPrice: pp.SKUs[0].Price,
				RealPrice:     pp.SKUs[0].Price, // 引入优惠券时,需要重新计算
				Quantity:      p.Quantity,
			},
		}
		originalTotalPrice += item.SKU.OriginalPrice * p.Quantity
		realTotalPrice += item.SKU.RealPrice * p.Quantity
		orderItems = append(orderItems, item)
	}
	return orderItems, originalTotalPrice, realTotalPrice, nil
}

func (h *Handler) createPayment(ctx context.Context, order domain.Order, paymentChannels []PaymentItem) (payment.Payment, error) {
	records := make([]payment.Record, 0, len(paymentChannels))
	for _, pc := range paymentChannels {
		if pc.Type != payment.ChannelTypeCredit && pc.Type != payment.ChannelTypeWechat {
			return payment.Payment{}, fmt.Errorf("支付渠道非法")
		}
		records = append(records, payment.Record{
			Amount:  pc.Amount,
			Channel: pc.Type,
		})
	}
	return h.paymentSvc.CreatePayment(ctx, payment.Payment{
		OrderID:     order.ID,
		OrderSN:     order.SN,
		PayerID:     order.BuyerID,
		TotalAmount: order.RealTotalPrice,
		Records:     records,
	})
}

// RetrieveOrderStatus 获取订单状态
func (h *Handler) RetrieveOrderStatus(ctx *ginx.Context, req RetrieveOrderStatusReq, sess session.Session) (ginx.Result, error) {
	order, err := h.svc.FindOrderByUIDAndOrderSN(ctx.Request.Context(), sess.Claims().Uid, req.OrderSN)
	if err != nil {
		return systemErrorResult, fmt.Errorf("订单未找到: %w", err)
	}
	return ginx.Result{
		Data: RetrieveOrderStatusResp{
			OrderStatus: order.Status.ToUint8(),
		},
	}, nil
}

// ListOrders 分页查询用户订单
func (h *Handler) ListOrders(ctx *ginx.Context, req ListOrdersReq, sess session.Session) (ginx.Result, error) {
	orders, total, err := h.svc.FindOrdersByUID(ctx, sess.Claims().Uid, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: ListOrdersResp{
			Total: total,
			Orders: slice.Map(orders, func(idx int, src domain.Order) Order {
				return h.toOrderVO(src)
			}),
		},
	}, nil
}

func (h *Handler) toOrderVO(order domain.Order) Order {
	return Order{
		SN:                 order.SN,
		Payment:            Payment{SN: order.Payment.SN},
		OriginalTotalPrice: order.OriginalTotalPrice,
		RealTotalPrice:     order.RealTotalPrice,
		Status:             order.Status.ToUint8(),
		Items: slice.Map(order.Items, func(idx int, src domain.OrderItem) OrderItem {
			return OrderItem{
				SKU: SKU{
					SN:            src.SKU.SN,
					Image:         src.SKU.Image,
					Name:          src.SKU.Name,
					Desc:          src.SKU.Description,
					OriginalPrice: src.SKU.OriginalPrice,
					RealPrice:     src.SKU.RealPrice,
					Quantity:      src.SKU.Quantity,
				},
			}
		}),
		Ctime: order.Ctime,
		Utime: order.Utime,
	}
}

// RetrieveOrderDetail 查看订单详情
func (h *Handler) RetrieveOrderDetail(ctx *ginx.Context, req RetrieveOrderDetailReq, sess session.Session) (ginx.Result, error) {
	order, err := h.svc.FindOrderByUIDAndOrderSN(ctx.Request.Context(), sess.Claims().Uid, req.OrderSN)
	if err != nil {
		return systemErrorResult, fmt.Errorf("订单未找到: %w", err)
	}
	paymentInfo, err := h.paymentSvc.FindPaymentByID(ctx.Request.Context(), order.Payment.ID)
	if err != nil {
		return systemErrorResult, fmt.Errorf("支付未找到: %w", err)
	}
	return ginx.Result{
		Data: RetrieveOrderDetailResp{
			Order: h.toOrderVOWithPaymentInfo(order, paymentInfo),
		},
	}, nil
}

func (h *Handler) toOrderVOWithPaymentInfo(order domain.Order, pr payment.Payment) Order {
	vo := h.toOrderVO(order)
	vo.Payment.Items = slice.Map(pr.Records, func(idx int, src payment.Record) PaymentItem {
		return PaymentItem{
			Type:   src.Channel,
			Amount: src.Amount,
		}
	})
	return vo
}

// CancelOrder 取消订单
func (h *Handler) CancelOrder(ctx *ginx.Context, req CancelOrderReq, sess session.Session) (ginx.Result, error) {
	order, err := h.svc.FindOrderByUIDAndOrderSN(ctx.Request.Context(), sess.Claims().Uid, req.OrderSN)
	if err != nil {
		return systemErrorResult, fmt.Errorf("查找订单失败: %w", err)
	}
	err = h.svc.CancelOrder(ctx.Request.Context(), order.BuyerID, order.ID)
	if err != nil {
		return systemErrorResult, fmt.Errorf("取消订单失败: %w", err)
	}
	return ginx.Result{Msg: "OK"}, nil
}
