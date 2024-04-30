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

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/repository/dao"
)

type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error)
	FindPaymentByID(ctx context.Context, pmtID int64) (domain.Payment, error)
	UpdatePayment(ctx context.Context, pmt domain.Payment) error
	FindPaymentByOrderSN(ctx context.Context, orderSN string) (domain.Payment, error)
	FindTimeoutPayments(ctx context.Context, offset int, limit int, ctime int64) ([]domain.Payment, error)
	TotalTimeoutPayments(ctx context.Context, ctime int64) (int64, error)
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
	pp, records := p.toEntity(pmt)
	daoPmt, daoRecords, err := p.dao.FindOrCreate(ctx, pp, records)
	if err != nil {
		return domain.Payment{}, err
	}
	return p.toDomain(daoPmt, daoRecords), nil
}

func (p *paymentRepository) toEntity(pmt domain.Payment) (dao.Payment, []dao.PaymentRecord) {
	pp := dao.Payment{
		Id:               pmt.ID,
		SN:               pmt.SN,
		OrderId:          pmt.OrderID,
		OrderSn:          sql.NullString{String: pmt.OrderSN, Valid: pmt.OrderSN != ""},
		PayerId:          pmt.PayerID,
		OrderDescription: pmt.OrderDescription,
		TotalAmount:      pmt.TotalAmount,
		PaidAt:           pmt.PaidAt,
		Status:           pmt.Status.ToUint8(),
	}
	records := slice.Map(pmt.Records, func(idx int, src domain.PaymentRecord) dao.PaymentRecord {
		return dao.PaymentRecord{
			PaymentId:    src.PaymentID,
			PaymentNO3rd: sql.NullString{String: src.PaymentNO3rd, Valid: src.PaymentNO3rd != ""},
			Description:  src.Description,
			Channel:      src.Channel.ToUnit8(),
			Amount:       src.Amount,
			PaidAt:       src.PaidAt,
			Status:       src.Status.ToUint8(),
		}
	})
	return pp, records
}

func (p *paymentRepository) toDomain(pmt dao.Payment, records []dao.PaymentRecord) domain.Payment {
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
		Records: slice.Map(records, func(idx int, src dao.PaymentRecord) domain.PaymentRecord {
			return domain.PaymentRecord{
				PaymentID:    src.PaymentId,
				PaymentNO3rd: src.PaymentNO3rd.String,
				Description:  src.Description,
				Channel:      domain.ChannelType(src.Channel),
				Amount:       src.Amount,
				PaidAt:       src.PaidAt,
				Status:       domain.PaymentStatus(src.Status),
			}
		}),
		Ctime: pmt.Ctime,
	}
}

func (p *paymentRepository) FindPaymentByID(ctx context.Context, pmtID int64) (domain.Payment, error) {
	pmt, records, err := p.dao.FindPaymentByID(ctx, pmtID)
	return p.toDomain(pmt, records), err
}

func (p *paymentRepository) UpdatePayment(ctx context.Context, pmt domain.Payment) error {
	// 确保设置OrderSN,pmt.OrderSN -> pmt.ID -> []records{ {微信}, {积分}}
	// 找到的records可能有两条 —— 微信和积分
	entity, records := p.toEntity(pmt)
	return p.dao.UpdateByOrderSN(ctx, entity, records)
}

func (p *paymentRepository) FindPaymentByOrderSN(ctx context.Context, orderSN string) (domain.Payment, error) {
	pmt, records, err := p.dao.FindPaymentByOrderSN(ctx, orderSN)
	return p.toDomain(pmt, records), err
}

func (p *paymentRepository) FindTimeoutPayments(ctx context.Context, offset int, limit int, ctime int64) ([]domain.Payment, error) {
	pmts, err := p.dao.FindTimeoutPayments(ctx, offset, limit, ctime)
	if err != nil {
		return nil, err
	}
	// todo: 读扩散会对DB造成压力
	pp := make([]domain.Payment, 0, len(pmts))
	for _, pmt := range pmts {
		pmtDomain, _ := p.FindPaymentByID(ctx, pmt.Id)
		pp = append(pp, pmtDomain)
	}
	return pp, nil
}

func (p *paymentRepository) TotalTimeoutPayments(ctx context.Context, ctime int64) (int64, error) {
	return p.dao.CountTimeoutPayments(ctx, ctime)
}
