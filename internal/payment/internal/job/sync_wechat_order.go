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
	"time"

	"github.com/ecodeclub/webook/internal/payment/internal/service"
	"github.com/gotomicro/ego/core/elog"
)

type SyncWechatOrderJob struct {
	svc service.Service
	l   *elog.Component
}

func (s *SyncWechatOrderJob) Name() string {
	return "sync_wechat_order_job"
}

func (s *SyncWechatOrderJob) Run() error {
	offset := 0
	// 也可以做成参数
	const limit = 100
	// 三十分钟之前的订单我们就认为已经过期了。
	now := time.Now().Add(-time.Minute * 30)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		payments, err := s.svc.FindExpiredPayment(ctx, offset, limit, now)
		cancel()
		if err != nil {
			return err
		}

		for _, pmt := range payments {
			// todo: pmt.Record.Channel != Wechat 直接关闭
			// todo: == wechat 与微信同步对账
			ctx, cancel = context.WithTimeout(context.Background(), time.Second)
			err = s.svc.SyncWechatInfo(ctx, pmt.OrderSN)
			if err != nil {
				s.l.Error("同步微信支付信息失败",
					elog.String("trade_no", pmt.OrderSN),
					elog.FieldErr(err))
			}
			cancel()
		}
		if len(payments) < limit {
			// 没数据了
			return nil
		}
		offset = offset + len(payments)
	}
}
