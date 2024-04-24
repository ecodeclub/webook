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

	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/order"
	"github.com/gotomicro/ego/core/elog"
	"github.com/gotomicro/ego/task/ecron"
)

func initCronJobs(
	oJob *order.CloseTimeoutOrdersJob,
	cJob *credit.CloseTimeoutLockedCreditsJob,
) []*ecron.Component {
	return []*ecron.Component{
		ecron.Load("cron.close").Build(ecron.WithJob(funcJobWrapper(oJob))),
		ecron.Load("cron.close").Build(ecron.WithJob(funcJobWrapper(cJob))),
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
