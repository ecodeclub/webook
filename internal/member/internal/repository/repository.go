package repository

import (
	"context"

	"github.com/ecodeclub/webook/internal/member/internal/domain"
	"github.com/ecodeclub/webook/internal/member/internal/repository/dao"
)

type MemberRepository interface {
	FindByUserID(ctx context.Context, userID int64) (domain.Member, error)
	Create(ctx context.Context, member domain.Member) (int64, error)
	// Update(ctx context.Context, member domain.Member) error
}

func NewMemberRepository(d dao.MemberDAO) MemberRepository {
	return &memberRepository{
		dao: d,
	}
}

type memberRepository struct {
	dao dao.MemberDAO
}

func (m *memberRepository) FindByUserID(ctx context.Context, userID int64) (domain.Member, error) {
	d, err := m.dao.FindByUserID(ctx, userID)
	if err != nil {
		return domain.Member{}, err
	}
	return m.toDomain(d), nil
}

func (m *memberRepository) toDomain(d dao.Member) domain.Member {
	return domain.Member{
		ID:      d.Id,
		UserID:  d.UserId,
		StartAt: d.StartAt,
		EndAt:   d.EndAt,
		Status:  d.Status,
	}
}

func (m *memberRepository) Create(ctx context.Context, member domain.Member) (int64, error) {
	return m.dao.Create(ctx, m.toEntity(member))
}

func (m *memberRepository) toEntity(d domain.Member) dao.Member {
	return dao.Member{
		Id:      d.ID,
		UserId:  d.UserID,
		StartAt: d.StartAt,
		EndAt:   d.EndAt,
		Status:  d.Status,
	}
}
