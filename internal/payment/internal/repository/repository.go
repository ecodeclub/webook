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

package repository

import (
	"context"

	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/repository/dao"
)

type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error)
	CreateCreditPaymentRecord(ctx context.Context, record domain.PaymentRecord) (int64, error)
}

func NewRepository(d dao.PaymentDAO) PaymentRepository {
	return &paymentRepository{
		d: d,
	}
}

type paymentRepository struct {
	d dao.PaymentDAO
}

func (p *paymentRepository) CreateCreditPaymentRecord(ctx context.Context, record domain.PaymentRecord) (int64, error) {
	// TODO implement me
	panic("implement me")
}

func (p *paymentRepository) CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error) {
	// TODO implement me
	panic("implement me")
}
