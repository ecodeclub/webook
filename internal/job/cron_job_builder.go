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
	"time"

	"github.com/gotomicro/ego/core/elog"
	"github.com/robfig/cron/v3"
)

type CronJobBuilder struct {
	l *elog.Component
}

func NewCronJobBuilder() *CronJobBuilder {
	return &CronJobBuilder{
		l: elog.DefaultLogger,
	}
}

func (b *CronJobBuilder) Build(job Job) cron.Job {
	jobName := job.Name()
	return cronJobAdapterFunc(func() {
		start := time.Now()
		b.l.Debug("开始运行",
			elog.String("job-name", jobName))
		err := job.Run()
		if err != nil {
			b.l.Error("执行失败",
				elog.FieldErr(err),
				elog.String("job-name", jobName))
		}
		b.l.Debug("结束运行",
			elog.String("job-name", jobName))
		duration := time.Since(start)
		b.l.Debug("运行时间",
			elog.FieldCost(duration),
			elog.String("job-name", jobName))
	})
}

type cronJobAdapterFunc func()

func (c cronJobAdapterFunc) Run() {
	c()
}
