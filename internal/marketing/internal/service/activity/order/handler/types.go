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

package handler

import (
	"context"

	"github.com/ecodeclub/webook/internal/marketing/internal/domain"
	"github.com/ecodeclub/webook/internal/order"
)

type (
	OrderInfo struct {
		Order order.Order
		Items []order.Item
	}
	OrderHandler interface {
		Handle(ctx context.Context, info OrderInfo) error
	}

	RedeemInfo struct {
		RedeemerID int64
		Code       domain.RedemptionCode
	}

	RedeemerHandler interface {
		Redeem(ctx context.Context, info RedeemInfo) error
	}
)

const Biz = "order"
