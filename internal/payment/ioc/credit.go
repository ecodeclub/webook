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
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/payment/internal/events"
	"github.com/ecodeclub/webook/internal/payment/internal/repository"
	credit2 "github.com/ecodeclub/webook/internal/payment/internal/service/credit"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	"github.com/gotomicro/ego/core/elog"
)

func InitCreditPaymentService(svc credit.Service,
	repo repository.PaymentRepository,
	producer events.Producer,
	paymentDDLFunc func() int64,
	l *elog.Component,
) *credit2.PaymentService {
	return credit2.NewCreditPaymentService(svc, repo, producer, paymentDDLFunc, sequencenumber.NewGenerator(), l)
}
