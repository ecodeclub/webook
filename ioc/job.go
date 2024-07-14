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

package ioc

import (
	"context"
	"time"

	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/gotomicro/ego/task/ejob"

	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/order"
	"github.com/ecodeclub/webook/internal/payment"
	"github.com/ecodeclub/webook/internal/recon"
	"github.com/gotomicro/ego/core/elog"
	"github.com/gotomicro/ego/task/ecron"
)

// 手动运行，或者通过 http 来触发
func initJobs(knowledgeStarter *baguwen.KnowledgeJobStarter) []ejob.Ejob {
	return []ejob.Ejob{
		ejob.Job("gen-knowledge", knowledgeStarter.Start),
	}
}

// initCronJobs 定时任务
func initCronJobs(
	oJob *order.CloseTimeoutOrdersJob,
	cJob *credit.CloseTimeoutLockedCreditsJob,
	pJob *payment.SyncWechatOrderJob,
	rJob *recon.SyncPaymentAndOrderJob,
) []ecron.Ecron {
	return []ecron.Ecron{
		ecron.Load("cron.closeTimeoutOrder").Build(ecron.WithJob(funcJobWrapper(oJob))),
		ecron.Load("cron.unlockTimeoutCredit").Build(ecron.WithJob(funcJobWrapper(cJob))),
		ecron.Load("cron.syncWechatOrder").Build(ecron.WithJob(funcJobWrapper(pJob))),
		ecron.Load("cron.syncPaymentAndOrder").Build(ecron.WithJob(funcJobWrapper(rJob))),
	}
}

func funcJobWrapper(job ecron.NamedJob) ecron.FuncJob {
	name := job.Name()
	return func(ctx context.Context) error {
		start := time.Now()
		elog.DefaultLogger.Debug("开始运行",
			elog.String("cronjob", name))
		err := job.Run(ctx)
		if err != nil {
			elog.DefaultLogger.Error("执行失败",
				elog.FieldErr(err),
				elog.String("cronjob", name))
			return err
		}
		duration := time.Since(start)
		elog.DefaultLogger.Debug("结束运行",
			elog.String("cronjob", name),
			elog.FieldKey("运行时间"),
			elog.FieldCost(duration))
		return nil
	}
}
