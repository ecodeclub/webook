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
	"github.com/gotomicro/ego/task/ecron"
)

var _ ecron.NamedJob = (*CloseExpiredOrdersJob)(nil)

type CloseExpiredOrdersJob struct {
	svc     service.Service
	minutes int64
	seconds int64
	limit   int
}

func NewCloseExpiredOrdersJob(svc service.Service, minutes, seconds int64, limit int) *CloseExpiredOrdersJob {
	return &CloseExpiredOrdersJob{
		svc:     svc,
		minutes: minutes,
		seconds: seconds,
		limit:   limit,
	}
}

func (c *CloseExpiredOrdersJob) Name() string {
	return "CloseExpiredOrdersJob"
}

func (c *CloseExpiredOrdersJob) Run(ctx context.Context) error {
	// 冗余10秒
	ctime := time.Now().Add(time.Duration(-c.minutes)*time.Minute + time.Duration(-c.seconds)*time.Second).UnixMilli()

	for {
		orders, total, err := c.svc.FindTimeoutOrders(ctx, 0, c.limit, ctime)
		if err != nil {
			return fmt.Errorf("获取过期订单失败: %w", err)
		}

		ids := slice.Map(orders, func(idx int, src domain.Order) int64 {
			return src.ID
		})

		err = c.svc.CloseTimeoutOrders(ctx, ids, ctime)
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
