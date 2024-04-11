package repository

import (
	"context"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/member/internal/domain"
	"github.com/ecodeclub/webook/internal/member/internal/repository/dao"
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
		ID:      d.Id,
		Uid:     d.Uid,
		StartAt: d.StartAt,
		EndAt:   d.EndAt,
		Records: slice.Map(r, func(idx int, src dao.MemberRecord) domain.MemberRecord {
			return domain.MemberRecord{
				Key:   src.Key,
				Biz:   src.Biz,
				BizId: src.BizId,
				Desc:  src.Desc,
				Days:  src.Days,
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
		Id:      d.ID,
		Uid:     d.Uid,
		StartAt: d.StartAt,
		EndAt:   d.EndAt,
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
