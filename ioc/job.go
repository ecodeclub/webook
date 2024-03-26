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
	"github.com/ecodeclub/webook/internal/job"
	"github.com/ecodeclub/webook/internal/order"
	"github.com/robfig/cron/v3"
)

func InitCronJobs(cjob *order.CloseExpiredOrdersJob) *cron.Cron {
	builder := job.NewCronJobBuilder()
	expr := cron.New(cron.WithSeconds())
	_, err := expr.AddJob("@midnight", builder.Build(cjob))
	if err != nil {
		panic(err)
	}
	return expr
}
