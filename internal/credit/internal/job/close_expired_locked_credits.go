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
	"github.com/ecodeclub/webook/internal/credit/internal/domain"
	"github.com/ecodeclub/webook/internal/credit/internal/service"
	"github.com/gotomicro/ego/task/ecron"
)

var _ ecron.NamedJob = (*CloseExpiredLockedCreditsJob)(nil)

type CloseExpiredLockedCreditsJob struct {
	svc     service.Service
	minutes int64
	seconds int64
	limit   int
}

func NewCloseExpiredOrdersJob(svc service.Service, minutes, seconds int64, limit int) *CloseExpiredLockedCreditsJob {
	return &CloseExpiredLockedCreditsJob{
		svc:     svc,
		minutes: minutes,
		seconds: seconds,
		limit:   limit,
	}
}

func (c *CloseExpiredLockedCreditsJob) Name() string {
	return "CloseExpiredLockedCreditsJob"
}

func (c *CloseExpiredLockedCreditsJob) Run(ctx context.Context) error {
	// 冗余10秒
	ctime := time.Now().Add(time.Duration(-c.minutes)*time.Minute + time.Duration(-c.seconds)*time.Second).UnixMilli()

	for {
		creditLogs, total, err := c.svc.FindExpiredLockedCreditLogs(ctx, 0, c.limit, ctime)
		if err != nil {
			return fmt.Errorf("获取过期订单失败: %w", err)
		}

		ids := slice.Map(creditLogs, func(idx int, src domain.CreditLog) int64 {
			return src.ID
		})

		err = c.svc.CancelExpiredLockedCreditLogs(ctx, ids, ctime)
		if err != nil {
			return fmt.Errorf("关闭过期订单失败: %w", err)
		}

		if len(creditLogs) < c.limit {
			break
		}

		if int64(c.limit) >= total {
			break
		}
	}
	return nil
}
