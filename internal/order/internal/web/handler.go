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
	g.POST("/preview", ginx.BS[PreviewOrderReq](h.PreviewOrder))
	g.POST("/create", ginx.BS[CreateOrderReq](h.CreateOrder))
	g.POST("/repay", ginx.BS[OrderSNReq](h.RepayOrder))
	g.POST("/list", ginx.BS[ListOrdersReq](h.ListOrders))
	g.POST("/detail", ginx.BS[OrderSNReq](h.RetrieveOrderDetail))
	g.POST("/cancel", ginx.BS[OrderSNReq](h.CancelOrder))
}

func (h *Handler) PublicRoutes(_ *gin.Engine) {}

// PreviewOrder 获取订单预览信息, 此时订单尚未创建
func (h *Handler) PreviewOrder(ctx *ginx.Context, req PreviewOrderReq, sess session.Session) (ginx.Result, error) {

	orderItems, originalTotalPrice, realTotalPrice, err := h.getDomainOrderItems(ctx, req.SKUs)
	if err != nil {
		return systemErrorResult, fmt.Errorf("获取预览订单项失败: %w", err)
	}

	c, err := h.creditSvc.GetCreditsByUID(ctx.Request.Context(), sess.Claims().Uid)
	if err != nil {
		return systemErrorResult, fmt.Errorf("获取用户积分失败: %w", err)
	}

	pcs := h.paymentSvc.GetPaymentChannels(ctx.Request.Context())
	items := make([]PaymentItem, 0, len(pcs))
	for _, pc := range pcs {
		items = append(items, PaymentItem{Type: int64(pc.Type)})
	}

	return ginx.Result{
		Data: PreviewOrderResp{
			Order: Order{
				Payment: Payment{
					Items: items,
				},
				OriginalTotalAmt: originalTotalPrice,
				RealTotalAmt:     realTotalPrice,
				Items: slice.Map(orderItems, func(idx int, src domain.OrderItem) OrderItem {
					return OrderItem{
						SPU: h.toSPUVO(src.SPU),
						SKU: h.toSKUVO(src.SKU),
					}
				}),
			},
			Credits: c.TotalAmount,
			Policy:  "请注意: 虚拟商品、一旦支付成功不退、不换,请谨慎操作",
		},
	}, nil
}

func (h *Handler) toSPUVO(spu domain.SPU) SPU {
	return SPU{Category0: spu.Category0, Category1: spu.Category1}
}

func (h *Handler) toSKUVO(sku domain.SKU) SKU {
	return SKU{
		SN:            sku.SN,
		Image:         sku.Image,
		Name:          sku.Name,
		Desc:          sku.Description,
		OriginalPrice: sku.OriginalPrice,
		RealPrice:     sku.RealPrice, // 引入优惠券时, 需要获取用户的优惠信息,动态计算
		Quantity:      sku.Quantity,
	}
}

// CreateOrder 创建订单和支付
func (h *Handler) CreateOrder(ctx *ginx.Context, req CreateOrderReq, sess session.Session) (ginx.Result, error) {

	if err := h.checkRequestID(ctx.Request.Context(), req.RequestID); err != nil {
		return systemErrorResult, fmt.Errorf("请求ID错误: %w", err)
	}

	uid := sess.Claims().Uid
	order, err := h.createOrder(ctx, req.SKUs, uid)
	if err != nil {
		return systemErrorResult, fmt.Errorf("创建订单失败: %w, uid: %d", err, uid)
	}

	p, err := h.createPayment(ctx, order, req.PaymentItems)
	if err != nil {
		return systemErrorResult, fmt.Errorf("创建支付失败: %w", err)
	}

	err = h.svc.UpdateUnpaidOrderPaymentInfo(ctx.Request.Context(), order.BuyerID, order.ID, p.ID, p.SN)
	if err != nil {
		return systemErrorResult, err
	}

	r, err := h.processPaymentForOrder(ctx.Request.Context(), p.ID)

	if err != nil {
		return systemErrorResult, err
	}

	ret := h.buildCreateOrderResp(order, r)
	return ginx.Result{
		Data: ret,
	}, nil
}

func (h *Handler) buildCreateOrderResp(order domain.Order, r payment.Record) CreateOrderResp {
	ret := CreateOrderResp{
		SN: order.SN,
	}
	if r.Channel == payment.ChannelTypeWechat {
		ret.WechatCodeURL = r.WechatCodeURL
	} else if r.Channel == payment.ChannelTypeWechatJS {
		jsAPI := r.WechatJsAPIResp
		ret.WechatJsAPI = WechatJsAPI{
			PrepayId:  jsAPI.PrepayId,
			Appid:     jsAPI.Appid,
			TimeStamp: jsAPI.TimeStamp,
			NonceStr:  jsAPI.NonceStr,
			Package:   jsAPI.Package,
			SignType:  jsAPI.SignType,
			PaySign:   jsAPI.PaySign,
		}
	}
	return ret
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
	// TODO: 这里有一个隐患，就是如果要是最终并没有创建 ORDER 成功，
	//       这会要求用户必须重新创建一个订单
	if err := h.cache.Set(ctx, key, requestID, 0); err != nil {
		return fmt.Errorf("缓存请求ID失败: %w", err)
	}
	return nil
}

func (h *Handler) createOrderRequestKey(requestID string) string {
	return fmt.Sprintf("order:create:%s", requestID)
}

func (h *Handler) createOrder(ctx context.Context, skus []SKU, buyerID int64) (domain.Order, error) {
	orderItems, originalTotalAmt, realTotalAmt, err := h.getDomainOrderItems(ctx, skus)
	if err != nil {
		return domain.Order{}, err
	}

	orderSN, err := h.snGenerator.Generate(buyerID)
	if err != nil {
		return domain.Order{}, fmt.Errorf("生成订单序列号失败")
	}

	return h.svc.CreateOrder(ctx, domain.Order{
		SN:               orderSN,
		BuyerID:          buyerID,
		OriginalTotalAmt: originalTotalAmt,
		RealTotalAmt:     realTotalAmt,
		Items:            orderItems,
	})
}

func (h *Handler) getDomainOrderItems(ctx context.Context, skus []SKU) ([]domain.OrderItem, int64, int64, error) {
	if len(skus) == 0 {
		return nil, 0, 0, fmt.Errorf("商品信息非法")
	}
	orderItems := make([]domain.OrderItem, 0, len(skus))
	originalTotalAmt, realTotalAmt := int64(0), int64(0)
	for _, sku := range skus {
		productSKU, err := h.productSvc.FindSKUBySN(ctx, sku.SN)
		if err != nil {
			// SN非法
			return nil, 0, 0, fmt.Errorf("商品SKUSN非法: %w", err)
		}
		if sku.Quantity < 1 || sku.Quantity > productSKU.Stock {
			// todo: 重新审视stockLimit的意义及用法
			// 暂时不需要修改
			return nil, 0, 0, fmt.Errorf("商品数量非法")
		}
		spu, err := h.productSvc.FindSPUByID(ctx, productSKU.SPUID)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("商品SPU ID非法: %w", err)
		}
		item := domain.OrderItem{
			SPU: domain.SPU{
				ID:        spu.ID,
				Category0: spu.Category0,
				Category1: spu.Category1,
			},
			SKU: domain.SKU{
				ID:            productSKU.ID,
				SN:            productSKU.SN,
				Attrs:         productSKU.Attrs,
				Image:         productSKU.Image,
				Name:          productSKU.Name,
				Description:   productSKU.Desc,
				OriginalPrice: productSKU.Price,
				RealPrice:     productSKU.Price, // 引入优惠券时,需要重新计算
				Quantity:      sku.Quantity,
			},
		}
		originalTotalAmt += item.SKU.OriginalPrice * sku.Quantity
		realTotalAmt += item.SKU.RealPrice * sku.Quantity
		orderItems = append(orderItems, item)
	}
	return orderItems, originalTotalAmt, realTotalAmt, nil
}

func (h *Handler) createPayment(ctx context.Context, order domain.Order, paymentChannels []PaymentItem) (payment.Payment, error) {
	// TODO: 针对订单生成更精确的订单描述信息
	orderDescription := "面窝吧"
	records := make([]payment.Record, 0, len(paymentChannels))
	realTotalAmt := int64(0)
	channelsSet := map[payment.ChannelType]struct{}{
		payment.ChannelTypeCredit:   {},
		payment.ChannelTypeWechat:   {},
		payment.ChannelTypeWechatJS: {},
	}
	for _, pc := range paymentChannels {
		if _, ok := channelsSet[payment.ChannelType(pc.Type)]; !ok {
			return payment.Payment{}, fmt.Errorf("支付渠道非法")
		}
		records = append(records, payment.Record{
			Amount:  pc.Amount,
			Channel: payment.ChannelType(pc.Type),
		})
		realTotalAmt += pc.Amount
	}
	if realTotalAmt != order.RealTotalAmt {
		return payment.Payment{}, fmt.Errorf("支付信息错误：金额不匹配")
	}
	return h.paymentSvc.CreatePayment(ctx, payment.Payment{
		OrderID:          order.ID,
		OrderSN:          order.SN,
		PayerID:          order.BuyerID,
		OrderDescription: orderDescription,
		TotalAmount:      order.RealTotalAmt,
		Records:          records,
	})
}

func (h *Handler) processPaymentForOrder(ctx context.Context, pmtID int64) (payment.Record, error) {
	p, err := h.paymentSvc.PayByID(ctx, pmtID)
	if err != nil {
		return payment.Record{}, fmt.Errorf("执行支付失败: %w, pmtID: %d", err, pmtID)
	}
	r, _ := slice.Find(p.Records, func(r payment.Record) bool {
		return payment.ChannelTypeWechat == r.Channel || payment.ChannelTypeWechatJS == r.Channel
	})
	return r, nil
}

// RepayOrder 继续支付订单
func (h *Handler) RepayOrder(ctx *ginx.Context, req OrderSNReq, sess session.Session) (ginx.Result, error) {

	uid := sess.Claims().Uid
	order, err := h.svc.FindUserVisibleOrderByUIDAndSN(ctx.Request.Context(), uid, req.SN)
	if err != nil {
		return systemErrorResult, fmt.Errorf("订单未找到: %w", err)
	}

	if order.Status != domain.StatusProcessing {
		return systemErrorResult, fmt.Errorf("订单状态非法: %w, uid: %d, sn: %s", err, uid, req.SN)
	}

	r, err := h.processPaymentForOrder(ctx.Request.Context(), order.Payment.ID)
	if err != nil {
		return systemErrorResult, fmt.Errorf("执行支付失败: %w", err)
	}
	return ginx.Result{
		Data: h.buildCreateOrderResp(order, r),
	}, nil
}

// ListOrders 分页查询用户订单
func (h *Handler) ListOrders(ctx *ginx.Context, req ListOrdersReq, sess session.Session) (ginx.Result, error) {
	orders, total, err := h.svc.FindUserVisibleOrdersByUID(ctx, sess.Claims().Uid, req.Offset, req.Limit)
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
		SN:               order.SN,
		Payment:          Payment{SN: order.Payment.SN},
		OriginalTotalAmt: order.OriginalTotalAmt,
		RealTotalAmt:     order.RealTotalAmt,
		Status:           order.Status.ToUint8(),
		Items: slice.Map(order.Items, func(idx int, src domain.OrderItem) OrderItem {
			return OrderItem{
				SPU: h.toSPUVO(src.SPU),
				SKU: h.toSKUVO(src.SKU),
			}
		}),
		Ctime: order.Ctime,
		Utime: order.Utime,
	}
}

// RetrieveOrderDetail 查看订单详情
func (h *Handler) RetrieveOrderDetail(ctx *ginx.Context, req OrderSNReq, sess session.Session) (ginx.Result, error) {
	order, err := h.svc.FindUserVisibleOrderByUIDAndSN(ctx.Request.Context(), sess.Claims().Uid, req.SN)
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
			Type:   int64(src.Channel),
			Amount: src.Amount,
		}
	})
	return vo
}

// CancelOrder 取消订单
func (h *Handler) CancelOrder(ctx *ginx.Context, req OrderSNReq, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	order, err := h.svc.FindUserVisibleOrderByUIDAndSN(ctx.Request.Context(), uid, req.SN)
	if err != nil {
		return systemErrorResult, fmt.Errorf("查找订单失败: %w", err)
	}
	if order.Status != domain.StatusProcessing {
		return systemErrorResult, fmt.Errorf("订单状态非法: %w, uid: %d, sn: %s", err, uid, req.SN)
	}
	err = h.svc.CancelOrder(ctx.Request.Context(), order.BuyerID, order.ID)
	if err != nil {
		return systemErrorResult, fmt.Errorf("取消订单失败: %w", err)
	}
	return ginx.Result{Msg: "OK"}, nil
}
