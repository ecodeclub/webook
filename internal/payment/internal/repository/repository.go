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
	"time"

	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/repository/dao"
)

type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error)
	CreatePaymentRecord(ctx context.Context, record domain.PaymentRecord) (int64, error)

	AddPayment(ctx context.Context, pmt domain.Payment) error
	// UpdatePayment 这个设计有点差，因为
	UpdatePayment(ctx context.Context, pmt domain.Payment) error
	FindExpiredPayment(ctx context.Context, offset int, limit int, t time.Time) ([]domain.Payment, error)
	GetPayment(ctx context.Context, bizTradeNO string) (domain.Payment, error)
}

func NewPaymentRepository(d dao.PaymentDAO) PaymentRepository {
	return &paymentRepository{
		dao: d,
	}
}

type paymentRepository struct {
	dao dao.PaymentDAO
}

func (p *paymentRepository) CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error) {
	// TODO implement me
	panic("implement me")
}

func (p *paymentRepository) CreatePaymentRecord(ctx context.Context, record domain.PaymentRecord) (int64, error) {
	// TODO implement me
	panic("implement me")
}

func (p *paymentRepository) GetPayment(ctx context.Context, bizTradeNO string) (domain.Payment, error) {
	r, err := p.dao.GetPayment(ctx, bizTradeNO)
	return p.toDomain(r), err
}

func (p *paymentRepository) FindExpiredPayment(ctx context.Context, offset int, limit int, t time.Time) ([]domain.Payment, error) {
	pmts, err := p.dao.FindExpiredPayment(ctx, offset, limit, t)
	if err != nil {
		return nil, err
	}
	res := make([]domain.Payment, 0, len(pmts))
	for _, pmt := range pmts {
		res = append(res, p.toDomain(pmt))
	}
	return res, nil
}

func (p *paymentRepository) AddPayment(ctx context.Context, pmt domain.Payment) error {
	return p.dao.Insert(ctx, p.toEntity(pmt))
}

func (p *paymentRepository) toDomain(pmt dao.Payment) domain.Payment {
	return domain.Payment{
		TotalAmount:      pmt.TotalAmount,
		OrderSN:          pmt.OrderSn,
		OrderDescription: pmt.OrderDescription,
		Status:           pmt.Status,
	}
}

func (p *paymentRepository) toEntity(pmt domain.Payment) dao.Payment {
	return dao.Payment{
		TotalAmount:      pmt.TotalAmount,
		OrderSn:          pmt.OrderSN,
		OrderDescription: pmt.OrderDescription,
		Status:           domain.PaymentStatusUnpaid,
	}
}

func (p *paymentRepository) UpdatePayment(ctx context.Context, pmt domain.Payment) error {
	// todo: 应该是OrderSN, paymentNo3rd(txn_id), Status
	return p.dao.UpdateTxnIDAndStatus(ctx, pmt.OrderSN, pmt.OrderSN, pmt.Status)
}
