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

// PreviewOrderReq 预览订单请求
type PreviewOrderReq struct {
	SKUs []SKU `json:"skus"` // 商品信息
}

type PreviewOrderResp struct {
	Order   Order  `json:"order"`   // 预览oder, 包含支持的渠道, 和要购买的SKU
	Credits uint64 `json:"credits"` // 积分总数
	Policy  string `json:"policy"`  // 政策信息
}

type SKU struct {
	SN            string `json:"sn"`
	Image         string `json:"image"`
	Name          string `json:"name"`
	Desc          string `json:"desc"`
	OriginalPrice int64  `json:"originalPrice"`
	RealPrice     int64  `json:"realPrice"`
	Quantity      int64  `json:"quantity"`
}

type PaymentItem struct {
	Type   int64 `json:"type"` // 1 积分, 2微信
	Amount int64 `json:"amount,omitempty"`
}

// CreateOrderReq 创建订单请求
type CreateOrderReq struct {
	RequestID    string        `json:"requestID"`    // 请求去重,防止订单重复提交
	SKUs         []SKU         `json:"skus"`         // 商品信息
	PaymentItems []PaymentItem `json:"paymentItems"` // 支付通道
}

type CreateOrderResp struct {
	OrderSN       string `json:"orderSN"` // 前端用于轮训订单状态,然后根据状态/时间限制来跳转
	WechatCodeURL string `json:"wechatCodeURL,omitempty"`
}

// RetrieveOrderStatusReq 获取订单状态
type RetrieveOrderStatusReq struct {
	OrderSN string `json:"sn"`
}

type RetrieveOrderStatusResp struct {
	OrderStatus uint8 `json:"status"`
}

// ListOrdersReq 分页查询用户所有订单
type ListOrdersReq struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

type ListOrdersResp struct {
	Total  int64   `json:"total,omitempty"`
	Orders []Order `json:"orders,omitempty"`
}

// RetrieveOrderDetailReq 获取订单详情
type RetrieveOrderDetailReq struct {
	OrderSN string `json:"sn"`
}

type RetrieveOrderDetailResp struct {
	Order Order `json:"order"`
}

type Order struct {
	SN               string      `json:"sn"`
	Payment          Payment     `json:"payment"`
	OriginalTotalAmt int64       `json:"originalTotalAmt"`
	RealTotalAmt     int64       `json:"realTotalAmt"`
	Status           uint8       `json:"status"`
	Items            []OrderItem `json:"items"`
	Ctime            int64       `json:"ctime"`
	Utime            int64       `json:"utime"`
}

type Payment struct {
	SN    string        `json:"sn"`
	Items []PaymentItem `json:"items,omitempty"`
}

type OrderItem struct {
	SKU SKU `json:"sku"`
}

// CancelOrderReq 取消订单
type CancelOrderReq struct {
	OrderSN string `json:"sn"`
}
