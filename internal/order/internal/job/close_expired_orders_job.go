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

package job

import (
	"context"
	"fmt"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/order/internal/domain"
	"github.com/ecodeclub/webook/internal/order/internal/service"
)

type CloseExpiredOrdersJob struct {
	svc     service.Service
	limit   int
	minute  int64
	timeout time.Duration
}

func NewCloseExpiredOrdersJob(svc service.Service, limit int, minute int64, timeout time.Duration) *CloseExpiredOrdersJob {
	return &CloseExpiredOrdersJob{svc: svc, limit: limit, minute: minute, timeout: timeout}
}

func (c *CloseExpiredOrdersJob) Name() string {
	return "CloseExpiredOrdersJob"
}

func (c *CloseExpiredOrdersJob) Run() error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), c.timeout)
	defer cancelFunc()
	// 冗余10秒
	ctime := time.Now().Add(time.Duration(-c.minute)*time.Minute + 10*time.Second).UnixMilli()

	for {
		orders, total, err := c.svc.FindExpiredOrders(ctx, 0, c.limit, ctime)
		if err != nil {
			return fmt.Errorf("获取过期订单失败: %w", err)
		}

		ids := slice.Map(orders, func(idx int, src domain.Order) int64 {
			return src.ID
		})

		err = c.svc.CloseExpiredOrders(ctx, ids, ctime)
		if err != nil {
			return fmt.Errorf("关闭过期订单失败: %w", err)
		}

		if len(orders) < c.limit {
			break
		}

		if int64(c.limit) >= total {
			break
		}
	}
	return nil
}
