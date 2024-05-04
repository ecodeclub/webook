package repository

import (
	"context"
	"time"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/ecodeclub/webook/internal/search/internal/repository/dao"
)

type caseRepository struct {
	caseDao dao.CaseDAO
}

func NewCaseRepo(caseDao dao.CaseDAO) CaseRepo {
	return &caseRepository{
		caseDao: caseDao,
	}
}

func (c *caseRepository) SearchCase(ctx context.Context, keywords string) ([]domain.Case, error) {
	cases, err := c.caseDao.SearchCase(ctx, keywords)
	if err != nil {
		return nil, err
	}
	ans := make([]domain.Case, 0, len(cases))
	for _, ca := range cases {
		ans = append(ans, c.toDomain(ca))
	}
	return ans, err
}

func (*caseRepository) toDomain(p dao.Case) domain.Case {
	return domain.Case{
		Id:        p.Id,
		Uid:       p.Uid,
		Labels:    p.Labels,
		Title:     p.Title,
		Content:   p.Content,
		Keywords:  p.Keywords,
		CodeRepo:  p.CodeRepo,
		Shorthand: p.Shorthand,
		Highlight: p.Highlight,
		Guidance:  p.Guidance,
		Status:    domain.CaseStatus(p.Status),
		Ctime:     time.UnixMilli(p.Ctime),
		Utime:     time.UnixMilli(p.Utime),
	}
}
