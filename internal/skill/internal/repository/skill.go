package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ecodeclub/ekit/mapx"
	"github.com/ecodeclub/ekit/slice"

	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/skill/internal/domain"
	"github.com/ecodeclub/webook/internal/skill/internal/repository/cache"
	dao "github.com/ecodeclub/webook/internal/skill/internal/repository/dao"
	"github.com/gotomicro/ego/core/elog"
	"golang.org/x/sync/errgroup"
)

type SkillRepo interface {
	// Save 管理端接口
	// 和 Update 返回值为  skill 的 id
	Save(ctx context.Context, skill domain.Skill) (int64, error)
	SaveRefs(ctx context.Context, skill domain.Skill) error
	// List 列表
	List(ctx context.Context, offset, limit int) ([]domain.Skill, error)
	// Info 详情
	Info(ctx context.Context, id int64) (domain.Skill, error)
	Count(ctx context.Context) (int64, error)
}
type skillRepo struct {
	skillDao dao.SkillDAO
	// 暂时没用上
	cache  cache.SkillCache
	logger *elog.Component
}

func (s *skillRepo) Save(ctx context.Context, skill domain.Skill) (int64, error) {
	var id int64
	var err error
	skillDao := s.skillToEntity(skill)
	levels := []dao.SkillLevel{
		s.skillLevelToEntity(skill.Basic, dao.LevelBasic),
		s.skillLevelToEntity(skill.Intermediate, dao.LevelIntermediate),
		s.skillLevelToEntity(skill.Advanced, dao.LevelAdvanced),
	}
	if skill.ID == 0 {
		id, err = s.skillDao.Create(ctx, skillDao, levels)
	} else {
		id = skill.ID
		err = s.skillDao.Update(ctx, skillDao, levels)
	}
	return id, err

}

func (s *skillRepo) SaveRefs(ctx context.Context, skill domain.Skill) error {
	refs := make([]dao.SkillRef, 0, 32)
	refs = append(refs, s.toRef(skill.ID, skill.Basic)...)
	refs = append(refs, s.toRef(skill.ID, skill.Intermediate)...)
	refs = append(refs, s.toRef(skill.ID, skill.Advanced)...)
	return s.skillDao.SaveRefs(ctx, refs)
}

func (s *skillRepo) toRef(sid int64, level domain.SkillLevel) []dao.SkillRef {
	res := make([]dao.SkillRef, 0, len(level.Cases)+len(level.Questions))
	now := time.Now().UnixMilli()
	for i := 0; i < len(level.Questions); i++ {
		res = append(res, dao.SkillRef{
			Rid:   level.Questions[i],
			Rtype: dao.RTypeQuestion,
			Sid:   sid,
			Slid:  level.Id,
			Ctime: now,
			Utime: now,
		})
	}

	for i := 0; i < len(level.Cases); i++ {
		res = append(res, dao.SkillRef{
			Rid:   level.Cases[i],
			Rtype: dao.RTypeCase,
			Sid:   sid,
			Slid:  level.Id,
			Ctime: now,
			Utime: now,
		})
	}
	return res
}

func (s *skillRepo) List(ctx context.Context, offset, limit int) ([]domain.Skill, error) {
	skillList, err := s.skillDao.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	domainSkills := make([]domain.Skill, 0, len(skillList))
	for _, sk := range skillList {
		domainSkills = append(domainSkills, s.skillToListDomain(sk))
	}
	return domainSkills, nil
}

func (s *skillRepo) Info(ctx context.Context, id int64) (domain.Skill, error) {
	var eg errgroup.Group
	var skill dao.Skill
	var skillLevels []dao.SkillLevel
	var refs []dao.SkillRef
	eg.Go(func() error {
		var err error
		skill, err = s.skillDao.Info(ctx, id)
		return err
	})
	eg.Go(func() error {
		var err error
		skillLevels, err = s.skillDao.SkillLevelInfo(ctx, id)
		return err
	})
	eg.Go(func() error {
		var err error
		refs, err = s.skillDao.Refs(ctx, id)
		return err
	})
	if err := eg.Wait(); err != nil {
		return domain.Skill{}, err
	}
	return s.skillToInfoDomain(skill, skillLevels, refs), nil
}

func (s *skillRepo) Count(ctx context.Context) (int64, error) {
	return s.skillDao.Count(ctx)
}

func (s *skillRepo) skillToListDomain(skill dao.Skill) domain.Skill {
	return domain.Skill{
		ID:     skill.Id,
		Labels: skill.Labels.Val,
		Name:   skill.Name,
		Desc:   skill.Desc,
		Ctime:  time.UnixMilli(skill.Ctime),
		Utime:  time.UnixMilli(skill.Utime),
	}
}
func (s *skillRepo) skillLevelToDomain(level dao.SkillLevel) domain.SkillLevel {
	return domain.SkillLevel{
		Id:    level.Id,
		Desc:  level.Desc,
		Ctime: time.UnixMilli(level.Ctime),
		Utime: time.UnixMilli(level.Utime),
	}
}

func (s *skillRepo) skillToInfoDomain(skill dao.Skill,
	levels []dao.SkillLevel, reqs []dao.SkillRef) domain.Skill {
	res := s.skillToListDomain(skill)
	reqsMap := mapx.NewMultiBuiltinMap[string, dao.SkillRef](4)
	for _, req := range reqs {
		_ = reqsMap.Put(fmt.Sprintf("%d_%s", req.Slid, req.Rtype), req)
	}
	for _, sl := range levels {
		dsl := s.skillLevelToDomain(sl)
		slQues, _ := reqsMap.Get(fmt.Sprintf("%d_%s", sl.Id, dao.RTypeQuestion))
		slCases, _ := reqsMap.Get(fmt.Sprintf("%d_%s", sl.Id, dao.RTypeCase))
		dsl.Questions = slice.Map(slQues, func(idx int, src dao.SkillRef) int64 {
			return src.Rid
		})
		dsl.Cases = slice.Map(slCases, func(idx int, src dao.SkillRef) int64 {
			return src.Rid
		})
		switch sl.Level {
		case dao.LevelBasic:
			res.Basic = dsl
		case dao.LevelIntermediate:
			res.Intermediate = dsl
		case dao.LevelAdvanced:
			res.Advanced = dsl
		}
	}
	return res
}

func (s *skillRepo) skillToEntity(skill domain.Skill) dao.Skill {
	return dao.Skill{
		Id:     skill.ID,
		Labels: sqlx.JsonColumn[[]string]{Val: skill.Labels, Valid: len(skill.Labels) != 0},
		Name:   skill.Name,
		Desc:   skill.Desc,
	}
}

func (s *skillRepo) skillLevelToEntity(skillLevel domain.SkillLevel, level string) dao.SkillLevel {
	return dao.SkillLevel{
		Id:    skillLevel.Id,
		Level: level,
		Desc:  skillLevel.Desc,
	}
}

func NewSkillRepo(skillDao dao.SkillDAO, skillCache cache.SkillCache) SkillRepo {
	return &skillRepo{
		skillDao: skillDao,
		cache:    skillCache,
		logger:   elog.DefaultLogger,
	}
}
