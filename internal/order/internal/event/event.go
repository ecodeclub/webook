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

package event

const (
	paymentEventName = "payment_events"
	orderEventName   = "order_events"
)

type PaymentEvent struct {
	OrderSN string `json:"orderSN"`
	PayerID int64  `json:"payerID"`
	Status  uint8  `json:"status"` // Success, Failed
}

type OrderEvent struct {
	OrderSN string `json:"orderSN"`
	BuyerID int64  `json:"buyerID"`
	SPUs    []SPU  `json:"spus"`
}

type SPU struct {
	ID        int64  `json:"id"`
	Category0 string `json:"category0"`
	Category1 string `json:"category1"`
}
