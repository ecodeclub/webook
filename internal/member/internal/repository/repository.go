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

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/member/internal/domain"
	"github.com/ecodeclub/webook/internal/member/internal/repository/dao"
)

var (
	ErrDuplicatedMemberRecord = dao.ErrDuplicatedMemberRecord
)

type MemberRepository interface {
	FindByUID(ctx context.Context, uid int64) (domain.Member, error)
	Upsert(ctx context.Context, member domain.Member) error
}

func NewMemberRepository(d dao.MemberDAO) MemberRepository {
	return &memberRepository{
		dao: d,
	}
}

type memberRepository struct {
	dao dao.MemberDAO
}

func (m *memberRepository) FindByUID(ctx context.Context, uid int64) (domain.Member, error) {
	d, err := m.dao.FindMemberByUID(ctx, uid)
	if err != nil {
		return domain.Member{}, err
	}
	r, err := m.dao.FindMemberRecordsByUID(ctx, uid)
	if err != nil {
		return domain.Member{}, err
	}
	return m.toDomain(d, r), nil
}

func (m *memberRepository) toDomain(d dao.Member, r []dao.MemberRecord) domain.Member {
	return domain.Member{
		Uid:   d.Uid,
		EndAt: d.EndAt,
		Records: slice.Map(r, func(idx int, src dao.MemberRecord) domain.MemberRecord {
			return domain.MemberRecord{
				Key:   src.Key,
				Days:  src.Days,
				Biz:   src.Biz,
				BizId: src.BizId,
				Desc:  src.Desc,
			}
		}),
	}
}

func (m *memberRepository) Upsert(ctx context.Context, member domain.Member) error {
	d, r := m.toEntity(member)
	return m.dao.Upsert(ctx, d, r)
}

func (m *memberRepository) toEntity(d domain.Member) (dao.Member, dao.MemberRecord) {
	member := dao.Member{
		Uid:   d.Uid,
		EndAt: d.EndAt,
	}
	record := slice.Map(d.Records, func(idx int, src domain.MemberRecord) dao.MemberRecord {
		return dao.MemberRecord{
			Key:   src.Key,
			Uid:   d.Uid,
			Biz:   src.Biz,
			BizId: src.BizId,
			Desc:  src.Desc,
			Days:  src.Days,
		}
	})
	return member, record[0]
}
