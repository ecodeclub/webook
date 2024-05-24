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
	"log"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/marketing/internal/domain"
	"github.com/ecodeclub/webook/internal/marketing/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/marketing/internal/repository/dao"
	"github.com/gotomicro/ego/core/elog"
)

var (
	ErrRedemptionNotFound = dao.ErrRedemptionNotFound
	ErrRedemptionCodeUsed = dao.ErrRedemptionCodeUsed
)

type MarketingRepository interface {
	CreateRedemptionCodes(ctx context.Context, codes []domain.RedemptionCode) ([]int64, error)
	FindRedemptionCode(ctx context.Context, code string) (domain.RedemptionCode, error)
	SetUnusedRedemptionCodeStatusUsed(ctx context.Context, uid int64, code string) (domain.RedemptionCode, error)
	TotalRedemptionCodes(ctx context.Context, uid int64) (int64, error)
	FindRedemptionCodesByUID(ctx context.Context, uid int64, offset, limit int) ([]domain.RedemptionCode, error)

	CreateInvitationCode(ctx context.Context, i domain.InvitationCode) (domain.InvitationCode, error)
	FindInvitationCodesByUID(ctx context.Context, uid int64) ([]domain.InvitationCode, error)
	FindInvitationCodeByCode(ctx context.Context, code string) (domain.InvitationCode, error)
}

type marketingRepository struct {
	dao    dao.MarketingDAO
	cache  cache.InvitationCodeCache
	logger *elog.Component
}

func NewRepository(d dao.MarketingDAO, c cache.InvitationCodeCache) MarketingRepository {
	return &marketingRepository{
		dao:    d,
		cache:  c,
		logger: elog.DefaultLogger,
	}
}

func (m *marketingRepository) CreateRedemptionCodes(ctx context.Context, codes []domain.RedemptionCode) ([]int64, error) {
	entities := m.toRedemptionCodeEntities(codes)
	log.Printf("entities: %#v\n", entities)
	return m.dao.CreateRedemptionCodes(ctx, entities)
}

func (m *marketingRepository) FindRedemptionCode(ctx context.Context, code string) (domain.RedemptionCode, error) {
	r, err := m.dao.FindRedemptionCodeByCode(ctx, code)
	if err != nil {
		return domain.RedemptionCode{}, err
	}
	return m.toRedemptionDomain([]dao.RedemptionCode{r})[0], err
}

func (m *marketingRepository) SetUnusedRedemptionCodeStatusUsed(ctx context.Context, uid int64, code string) (domain.RedemptionCode, error) {
	r, err := m.dao.SetUnusedRedemptionCodeStatusUsed(ctx, uid, code)
	if err != nil {
		return domain.RedemptionCode{}, err
	}
	return m.toRedemptionDomain([]dao.RedemptionCode{r})[0], err
}

func (m *marketingRepository) TotalRedemptionCodes(ctx context.Context, uid int64) (int64, error) {
	return m.dao.CountRedemptionCodes(ctx, uid)
}

func (m *marketingRepository) FindRedemptionCodesByUID(ctx context.Context, uid int64, offset, limit int) ([]domain.RedemptionCode, error) {
	codes, err := m.dao.FindRedemptionCodesByUID(ctx, uid, offset, limit)
	if err != nil {
		return nil, err
	}
	return m.toRedemptionDomain(codes), nil
}

func (m *marketingRepository) toRedemptionDomain(codes []dao.RedemptionCode) []domain.RedemptionCode {
	return slice.Map(codes, func(idx int, src dao.RedemptionCode) domain.RedemptionCode {

		return domain.RedemptionCode{
			ID:      src.Id,
			OwnerID: src.OwnerId,
			Biz:     src.Biz,
			BizId:   src.BizId,
			Type:    src.Type,
			Attrs:   src.Attrs.Val,
			Code:    src.Code,
			Status:  domain.RedemptionCodeStatus(src.Status),
			Ctime:   src.Ctime,
			Utime:   src.Utime,
		}
	})
}

func (m *marketingRepository) toRedemptionCodeEntities(codes []domain.RedemptionCode) []dao.RedemptionCode {
	return slice.Map(codes, func(idx int, src domain.RedemptionCode) dao.RedemptionCode {
		return dao.RedemptionCode{
			OwnerId: src.OwnerID,
			Biz:     src.Biz,
			BizId:   src.BizId,
			Type:    src.Type,
			Attrs:   sqlx.JsonColumn[domain.CodeAttrs]{Val: src.Attrs, Valid: true},
			Code:    src.Code,
			Status:  src.Status.ToUint8(),
		}
	})
}

func (m *marketingRepository) CreateInvitationCode(ctx context.Context, i domain.InvitationCode) (domain.InvitationCode, error) {
	code, err := m.cache.GetInvitationCode(ctx, i.Uid)
	if err == nil {
		return domain.InvitationCode{Uid: i.Uid, Code: code}, err
	}
	_, err = m.dao.CreateInvitationCode(ctx, m.toInvitationCodeEntity(i))
	if err != nil {
		return domain.InvitationCode{}, err
	}
	err = m.cache.SetInvitationCode(ctx, i.Uid, i.Code)
	if err != nil {
		m.logger.Error("缓存邀请码失败", elog.FieldErr(err))
	}
	return i, nil
}

func (m *marketingRepository) toInvitationCodeEntity(i domain.InvitationCode) dao.InvitationCode {
	return dao.InvitationCode{
		OwnerId: i.Uid,
		Code:    i.Code,
	}
}

func (m *marketingRepository) FindInvitationCodesByUID(ctx context.Context, uid int64) ([]domain.InvitationCode, error) {
	cs, err := m.dao.FindInvitationCodesByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	return slice.Map(cs, func(idx int, src dao.InvitationCode) domain.InvitationCode {
		return m.toInvitationCodeDomain(src)
	}), nil
}

func (m *marketingRepository) toInvitationCodeDomain(i dao.InvitationCode) domain.InvitationCode {
	return domain.InvitationCode{
		Uid:  i.OwnerId,
		Code: i.Code,
	}
}

func (m *marketingRepository) FindInvitationCodeByCode(ctx context.Context, code string) (domain.InvitationCode, error) {
	c, err := m.dao.FindInvitationCodeByCode(ctx, code)
	if err != nil {
		return domain.InvitationCode{}, err
	}
	return m.toInvitationCodeDomain(c), nil
}
