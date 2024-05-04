package repository

import (
	"context"
	"time"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/ecodeclub/webook/internal/search/internal/repository/dao"
)

type skillRepo struct {
	skillDao dao.SkillDAO
}

func NewSKillRepo(skillDao dao.SkillDAO) SkillRepo {
	return &skillRepo{
		skillDao: skillDao,
	}
}

func (s *skillRepo) SearchSkill(ctx context.Context, keywords string) ([]domain.Skill, error) {
	skillList, err := s.skillDao.SearchSkill(ctx, keywords)
	if err != nil {
		return nil, err
	}
	ans := make([]domain.Skill, 0, len(skillList))
	for _, sk := range skillList {
		ans = append(ans, s.toSkillDomain(sk))
	}
	return ans, nil
}

func (sk *skillRepo) toSkillDomain(s dao.Skill) domain.Skill {
	return domain.Skill{
		ID:           s.ID,
		Labels:       s.Labels,
		Name:         s.Name,
		Desc:         s.Desc,
		Basic:        sk.toSkillLevelDomain(s.Basic),
		Intermediate: sk.toSkillLevelDomain(s.Intermediate),
		Advanced:     sk.toSkillLevelDomain(s.Advanced),
		Ctime:        time.UnixMilli(s.Ctime),
		Utime:        time.UnixMilli(s.Utime),
	}
}

func (s *skillRepo) toSkillLevelDomain(sk dao.SkillLevel) domain.SkillLevel {
	return domain.SkillLevel{
		ID:        sk.ID,
		Desc:      sk.Desc,
		Ctime:     time.UnixMilli(sk.Ctime),
		Utime:     time.UnixMilli(sk.Utime),
		Questions: sk.Questions,
		Cases:     sk.Cases,
	}
}
