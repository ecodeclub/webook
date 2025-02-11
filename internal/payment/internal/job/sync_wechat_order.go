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
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/service"
	"github.com/gotomicro/ego/core/elog"
	"github.com/gotomicro/ego/task/ecron"
)

var _ ecron.NamedJob = (*SyncWechatOrderJob)(nil)

type SyncWechatOrderJob struct {
	svc     service.Service
	minutes int64
	seconds int64
	limit   int
	l       *elog.Component
}

func NewSyncWechatOrderJob(svc service.Service, minutes int64, seconds int64, limit int) *SyncWechatOrderJob {
	return &SyncWechatOrderJob{
		svc:     svc,
		minutes: minutes,
		seconds: seconds,
		limit:   limit,
		l:       elog.DefaultLogger}
}

func (s *SyncWechatOrderJob) Name() string {
	return "sync_wechat_order_job"
}

func (s *SyncWechatOrderJob) Run(ctx context.Context) error {

	ctime := time.Now().Add(time.Duration(-s.minutes)*time.Minute + time.Duration(-s.seconds)*time.Second).UnixMilli()

	for {

		payments, total, err := s.svc.FindTimeoutPayments(ctx, 0, s.limit, ctime)
		if err != nil {
			return fmt.Errorf("获取过期支付记录失败: %w", err)
		}

		for _, pmt := range payments {
			_, ok := slice.Find(pmt.Records, func(src domain.PaymentRecord) bool {
				return src.Channel == domain.ChannelTypeWechat || src.Channel == domain.ChannelTypeWechatJS
			})
			if !ok {
				// 非微信支付渠道支付,直接关闭
				err = s.svc.CloseTimeoutPayment(ctx, pmt)
				if err != nil {
					s.l.Error("关闭超时支付失败",
						elog.FieldErr(err),
						elog.String("order_sn", pmt.OrderSN),
						elog.Int64("payment_id", pmt.ID),
					)
				}
				continue
			}
			err = s.svc.SyncWechatInfo(ctx, pmt)
			if err != nil {
				s.l.Error("同步微信支付信息失败",
					elog.FieldErr(err),
					elog.String("OutTradeNo", pmt.OrderSN),
				)
			}
		}

		if len(payments) < s.limit {
			return nil
		}

		if int64(s.limit) >= total {
			return nil
		}

	}
}
