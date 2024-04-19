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
	"database/sql"
	"log"
	"time"

	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/repository/dao"
)

type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error)
	UpdatePayment(ctx context.Context, pmt domain.Payment) error
	FindPaymentByOrderSN(ctx context.Context, orderSN string) (domain.Payment, error)

	AddPayment(ctx context.Context, pmt domain.Payment) error
	// UpdatePayment 这个设计有点差，因为
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

func (p *paymentRepository) CreatePayment(ctx context.Context, pmt domain.Payment) (domain.Payment, error) {
	log.Printf("pmt = %#v\n", pmt)
	pp, records := p.toEntity(pmt)
	log.Printf("p = %#v, r = %#v\n", pp, records)
	id, err := p.dao.FindOrCreate(ctx, pp, records)
	if err != nil {
		return domain.Payment{}, err
	}
	pmt.ID = id
	return pmt, nil
}

func (p *paymentRepository) toEntity(pmt domain.Payment) (dao.Payment, []dao.PaymentRecord) {
	pp := dao.Payment{
		Id:               pmt.ID,
		SN:               pmt.SN,
		OrderId:          pmt.OrderID,
		OrderSn:          sql.NullString{String: pmt.OrderSN, Valid: true},
		PayerId:          pmt.PayerID,
		OrderDescription: pmt.OrderDescription,
		TotalAmount:      pmt.TotalAmount,
		PaidAt:           pmt.PaidAt,
		Status:           pmt.Status.ToUnit8(),
	}
	records := make([]dao.PaymentRecord, 0, len(pmt.Records))
	for _, r := range pmt.Records {
		records = append(records, dao.PaymentRecord{
			PaymentId:    r.PaymentID,
			PaymentNO3rd: sql.NullString{String: r.PaymentNO3rd, Valid: r.PaymentNO3rd != ""},
			Description:  r.Description,
			Channel:      r.Channel.ToUnit8(),
			Amount:       r.Amount,
			PaidAt:       r.PaidAt,
			Status:       r.Status.ToUnit8(),
		})
	}
	return pp, records
}

func (p *paymentRepository) UpdatePayment(ctx context.Context, pmt domain.Payment) error {
	// todo: 应该是OrderSN, paymentNo3rd(txn_id), PaymentStatus
	// return p.dao.UpdateTxnIDAndStatus(ctx, pmt.OrderSN, pmt.OrderSN, pmt.PaymentStatus)

	// 通过pmt.OrderSN -> pmt.ID -> []records{ {微信}, {积分}}
	// 找到的records可能有两条 —— 微信和积分
	entity, records := p.toEntity(pmt)
	return p.dao.Update(ctx, entity, records)
}

func (p *paymentRepository) FindPaymentByOrderSN(ctx context.Context, orderSN string) (domain.Payment, error) {
	pmt, records, err := p.dao.FindPaymentByOrderSN(ctx, orderSN)
	return p.toDomain(pmt, records), err
}

func (p *paymentRepository) toDomain(pmt dao.Payment, records []dao.PaymentRecord) domain.Payment {

	rs := make([]domain.PaymentRecord, 0, len(records))

	for i := 0; i < len(records); i++ {
		rs = append(rs, domain.PaymentRecord{
			PaymentID:    records[i].PaymentId,
			PaymentNO3rd: records[i].PaymentNO3rd.String,
			Description:  records[i].Description,
			Channel:      domain.ChannelType(records[i].Channel),
			Amount:       records[i].Amount,
			PaidAt:       records[i].PaidAt,
			Status:       domain.PaymentStatus(records[i].Status),
		})
	}

	return domain.Payment{
		ID:               pmt.Id,
		SN:               pmt.SN,
		PayerID:          pmt.PayerId,
		OrderID:          pmt.OrderId,
		OrderSN:          pmt.OrderSn.String,
		OrderDescription: pmt.OrderDescription,
		TotalAmount:      pmt.TotalAmount,
		PaidAt:           pmt.PaidAt,
		Status:           domain.PaymentStatus(pmt.Status),
		Records:          rs,
		Ctime:            pmt.Ctime,
		Utime:            pmt.Utime,
	}
}

func (p *paymentRepository) GetPayment(ctx context.Context, bizTradeNO string) (domain.Payment, error) {
	r, err := p.dao.GetPayment(ctx, bizTradeNO)
	return p.toDomain2(r), err
}

func (p *paymentRepository) FindExpiredPayment(ctx context.Context, offset int, limit int, t time.Time) ([]domain.Payment, error) {
	pmts, err := p.dao.FindExpiredPayment(ctx, offset, limit, t)
	if err != nil {
		return nil, err
	}
	res := make([]domain.Payment, 0, len(pmts))
	for _, pmt := range pmts {
		res = append(res, p.toDomain2(pmt))
	}
	return res, nil
}

func (p *paymentRepository) AddPayment(ctx context.Context, pmt domain.Payment) error {
	return p.dao.Insert(ctx, p.toEntity2(pmt))
}

func (p *paymentRepository) toEntity2(pmt domain.Payment) dao.Payment {
	return dao.Payment{
		TotalAmount:      pmt.TotalAmount,
		OrderSn:          sql.NullString{String: pmt.OrderSN, Valid: true},
		OrderDescription: pmt.OrderDescription,
		Status:           domain.PaymentStatusUnpaid.ToUnit8(),
	}
}

func (p *paymentRepository) toDomain2(pmt dao.Payment) domain.Payment {
	return domain.Payment{
		TotalAmount:      pmt.TotalAmount,
		OrderSN:          pmt.OrderSn.String,
		OrderDescription: pmt.OrderDescription,
		Status:           domain.PaymentStatus(pmt.Status),
	}
}
