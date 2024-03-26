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
	"time"

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

	// todo: 重构为event和cronjob并添加相关测试后,需要去掉下面这两个方法及相关测试
	g.POST("/complete", ginx.B[CompleteOrderReq](h.CompleteOrder))
	g.POST("/close", ginx.B[CloseTimeoutOrdersReq](h.CloseTimeoutOrders))
}

// RetrievePreviewOrder 获取订单预览信息, 此时订单尚未创建
func (h *Handler) RetrievePreviewOrder(ctx *ginx.Context, req PreviewOrderReq, sess session.Session) (ginx.Result, error) {
	p, err := h.productSvc.FindBySN(ctx.Request.Context(), req.ProductSKUSN)
	if err != nil {
		return systemErrorResult, fmt.Errorf("商品SKU序列号非法: %w", err)
	}
	if req.Quantity < 1 || req.Quantity > p.SKU.Stock {
		// todo: 重新审视stockLimit的意义及用法
		return systemErrorResult, fmt.Errorf("要购买的商品数量非法")
	}
	c, err := h.creditSvc.GetByUID(ctx.Request.Context(), sess.Claims().Uid)
	if err != nil {
		return systemErrorResult, fmt.Errorf("获取用户积分失败: %w", err)
	}
	return ginx.Result{
		Data: PreviewOrderResp{
			Credits:  c.Amount,
			Payments: h.toPaymentChannelVO(ctx),
			Products: h.toProductVO(p, req.Quantity),
			Policy:   "请注意: 虚拟商品、一旦支持成功不退、不换,请谨慎操作",
		},
	}, nil
}

func (h *Handler) toPaymentChannelVO(ctx *ginx.Context) []Payment {
	pcs := h.paymentSvc.GetPaymentChannels(ctx.Request.Context())
	channels := make([]Payment, 0, len(pcs))
	for _, pc := range pcs {
		channels = append(channels, Payment{Type: pc.Type})
	}
	return channels
}

func (h *Handler) toProductVO(p product.Product, quantity int64) []Product {
	return []Product{
		{
			SPUSN:         p.SPU.SN,
			SKUSN:         p.SKU.SN,
			Name:          p.SKU.Name,
			Desc:          p.SKU.Desc,
			OriginalPrice: p.SKU.Price,
			RealPrice:     p.SKU.Price, // 引入优惠券时, 需要获取用户的优惠信息,动态计算
			Quantity:      quantity,
		},
	}
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

	order.PaymentID = p.ID
	order.PaymentSN = p.SN
	err = h.svc.UpdateOrder(ctx.Request.Context(), order)
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
	return fmt.Sprintf("webook:order:create:%s", requestID)
}

func (h *Handler) createOrder(ctx context.Context, req CreateOrderReq, buyerID int64) (domain.Order, error) {
	orderItems, originalTotalPrice, realTotalPrice, err := h.getOrderItems(ctx, req)
	if err != nil {
		return domain.Order{}, err
	}
	if originalTotalPrice != req.OriginalTotalPrice {
		// 总原价非法
		return domain.Order{}, fmt.Errorf("商品总原价非法")
	}
	if realTotalPrice != req.RealTotalPrice {
		// 总实价非法
		return domain.Order{}, fmt.Errorf("商品总实价非法")
	}

	orderSN, err := h.snGenerator.Generate(buyerID)
	if err != nil {
		// 总实价非法
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
	if len(req.Products) == 0 {
		return nil, 0, 0, fmt.Errorf("商品信息非法")
	}
	orderItems := make([]domain.OrderItem, 0, len(req.Products))
	originalTotalPrice, realTotalPrice := int64(0), int64(0)
	for _, p := range req.Products {
		pp, err := h.productSvc.FindBySN(ctx, p.SKUSN)
		if err != nil {
			// SN非法
			return nil, 0, 0, fmt.Errorf("商品SKUSN非法: %w", err)
		}
		if p.Quantity < 1 || p.Quantity > pp.SKU.Stock {
			// todo: 重新审视stockLimit的意义及用法
			return nil, 0, 0, fmt.Errorf("商品数量非法")
		}

		item := domain.OrderItem{
			SPUID:            pp.SPU.ID,
			SKUID:            pp.SKU.ID,
			SKUName:          pp.SKU.Name,
			SKUDescription:   pp.SKU.Desc,
			SKUOriginalPrice: pp.SKU.Price,
			SKURealPrice:     pp.SKU.Price, // 引入优惠券时,需要重新计算
			Quantity:         p.Quantity,
		}
		originalTotalPrice += item.SKUOriginalPrice * p.Quantity
		realTotalPrice += item.SKURealPrice * p.Quantity
		orderItems = append(orderItems, item)
	}
	return orderItems, originalTotalPrice, realTotalPrice, nil
}

func (h *Handler) createPayment(ctx context.Context, order domain.Order, paymentChannels []Payment) (payment.Payment, error) {
	records := make([]payment.Record, 0, len(paymentChannels))
	for _, pc := range paymentChannels {
		if pc.Type != payment.ChannelTypeCredit && pc.Type != payment.ChannelTypeWechat {
			return payment.Payment{}, fmt.Errorf("支付渠道非法")
		}
		records = append(records, payment.Record{
			// 每个支付渠道具体分摊多少金额留给payment模块自己决定
			Channel: pc.Type,
		})
	}
	return h.paymentSvc.CreatePayment(ctx, payment.Payment{
		OrderID:     order.ID,
		OrderSN:     order.SN,
		TotalAmount: order.RealTotalPrice,
		Deadline:    time.Now().Add(30 * time.Minute).UnixMilli(),
		Records:     records,
	})
}

// RetrieveOrderStatus 获取订单状态
func (h *Handler) RetrieveOrderStatus(ctx *ginx.Context, req RetrieveOrderStatusReq, sess session.Session) (ginx.Result, error) {
	order, err := h.svc.FindOrder(ctx.Request.Context(), req.OrderSN, sess.Claims().Uid)
	if err != nil {
		return systemErrorResult, fmt.Errorf("订单未找到: %w", err)
	}
	return ginx.Result{
		Data: RetrieveOrderStatusResp{
			OrderStatus: order.Status,
		},
	}, nil
}

// CompleteOrder 完成订单
func (h *Handler) CompleteOrder(ctx *ginx.Context, req CompleteOrderReq) (ginx.Result, error) {
	// todo: 是否加入RequestID去重, 支付模块发送的消息可能重复, 当然订单这边是幂等的
	//       所以连查询出的order的状态都不用判断(可能是已完成),直接设置为已完成即可
	order, err := h.svc.FindOrder(ctx.Request.Context(), req.OrderSN, req.BuyerID)
	if err != nil {
		return systemErrorResult, fmt.Errorf("订单未找到: %w", err)
	}
	// todo: 通过消息将paymentID和paymentSN带回给订单模块?
	// order.PaymentID = req.PaymentID
	// order.PaymentSN = req.PaymentSN
	err = h.svc.CompleteOrder(ctx.Request.Context(), order)
	if err != nil {
		return systemErrorResult, fmt.Errorf("完成订单失败: %w", err)
	}
	return ginx.Result{Msg: "OK"}, nil
}

// CloseTimeoutOrders 关闭超时订单
func (h *Handler) CloseTimeoutOrders(ctx *ginx.Context, req CloseTimeoutOrdersReq) (ginx.Result, error) {
	for {
		orders, total, err := h.svc.ListExpiredOrders(ctx.Request.Context(), 0, req.Limit, time.Now().Add(time.Duration(-req.Minute)*time.Minute).UnixMilli())
		if err != nil {
			return systemErrorResult, fmt.Errorf("获取过期订单失败: %w", err)
		}

		ids := slice.Map(orders, func(idx int, src domain.Order) int64 {
			return src.ID
		})

		err = h.svc.CloseExpiredOrders(ctx.Request.Context(), ids)
		if err != nil {
			return systemErrorResult, fmt.Errorf("关闭过期订单失败: %w", err)
		}

		if len(orders) < req.Limit {
			break
		}

		if int64(req.Limit) >= total {
			break
		}
	}
	return ginx.Result{Msg: "OK"}, nil
}

// ListOrders 分页查询用户订单
func (h *Handler) ListOrders(ctx *ginx.Context, req ListOrdersReq, sess session.Session) (ginx.Result, error) {
	orders, total, err := h.svc.ListOrders(ctx, req.Offset, req.Limit, sess.Claims().Uid)
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
		PaymentSN:          order.PaymentSN,
		OriginalTotalPrice: order.OriginalTotalPrice,
		RealTotalPrice:     order.RealTotalPrice,
		Status:             order.Status,
		Items: slice.Map(order.Items, func(idx int, src domain.OrderItem) OrderItem {
			return OrderItem{
				SPUID:            src.SPUID,
				SKUID:            src.SKUID,
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

// RetrieveOrderDetail 查看订单详情
func (h *Handler) RetrieveOrderDetail(ctx *ginx.Context, req RetrieveOrderDetailReq, sess session.Session) (ginx.Result, error) {
	order, err := h.svc.FindOrder(ctx.Request.Context(), req.OrderSN, sess.Claims().Uid)
	if err != nil {
		return systemErrorResult, fmt.Errorf("订单未找到: %w", err)
	}
	paymentInfo, err := h.paymentSvc.FindPaymentByID(ctx.Request.Context(), order.PaymentID)
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
	vo.Payments = slice.Map(pr.Records, func(idx int, src payment.Record) Payment {
		return Payment{
			Type:   src.Channel,
			Amount: src.Amount,
		}
	})
	return vo
}

// CancelOrder 取消订单
func (h *Handler) CancelOrder(ctx *ginx.Context, req CancelOrderReq, sess session.Session) (ginx.Result, error) {
	order, err := h.svc.FindOrder(ctx.Request.Context(), req.OrderSN, sess.Claims().Uid)
	if err != nil {
		return systemErrorResult, fmt.Errorf("查找订单失败: %w", err)
	}
	err = h.svc.CancelOrder(ctx.Request.Context(), order)
	if err != nil {
		return systemErrorResult, fmt.Errorf("取消订单失败: %w", err)
	}
	return ginx.Result{Msg: "OK"}, nil
}
