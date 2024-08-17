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
	RefsByLevelIDs(ctx context.Context, ids []int64) ([]domain.SkillLevel, error)
	LevelInfo(ctx context.Context, id int64) (domain.SkillLevel, error)
}
type skillRepo struct {
	skillDao dao.SkillDAO
	// 暂时没用上
	cache  cache.SkillCache
	logger *elog.Component
}

func (s *skillRepo) LevelInfo(ctx context.Context, id int64) (domain.SkillLevel, error) {
	sk, err := s.skillDao.SkillLevelFirst(ctx, id)
	if err != nil {
		return domain.SkillLevel{}, err
	}
	levels, err := s.RefsByLevelIDs(ctx, []int64{id})
	if err != nil {
		return domain.SkillLevel{}, err
	}
	level := domain.SkillLevel{
		Id:   sk.Id,
		Desc: sk.Desc,
	}
	if len(levels) > 0 {
		level.Cases = levels[0].Cases
		level.Questions = levels[0].Questions
		level.QuestionSets = levels[0].QuestionSets
		level.CaseSets = levels[0].CaseSets
	}
	return level, nil
}

func (s *skillRepo) RefsByLevelIDs(ctx context.Context, ids []int64) ([]domain.SkillLevel, error) {
	refs, err := s.skillDao.RefsByLevelIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	const keyPattern = "%s_%d"
	m := mapx.NewMultiBuiltinMap[string, int64](len(ids))
	for _, ref := range refs {
		_ = m.Put(fmt.Sprintf(keyPattern, ref.Rtype, ref.Slid), ref.Rid)
	}
	return slice.Map(ids, func(idx int, src int64) domain.SkillLevel {
		ques, _ := m.Get(fmt.Sprintf(keyPattern, dao.RTypeQuestion, src))
		cs, _ := m.Get(fmt.Sprintf(keyPattern, dao.RTypeCase, src))
		questionSets, _ := m.Get(fmt.Sprintf(keyPattern, dao.RTypeQuestionSet, src))
		caseSets, _ := m.Get(fmt.Sprintf(keyPattern, dao.RTypeCaseSet, src))
		return domain.SkillLevel{
			Id:           src,
			Questions:    ques,
			Cases:        cs,
			QuestionSets: questionSets,
			CaseSets:     caseSets,
		}
	}), nil
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

	for i := 0; i < len(level.QuestionSets); i++ {
		res = append(res, dao.SkillRef{
			Rid:   level.QuestionSets[i],
			Rtype: dao.RTypeQuestionSet,
			Sid:   sid,
			Slid:  level.Id,
			Ctime: now,
			Utime: now,
		})
	}

	for i := 0; i < len(level.CaseSets); i++ {
		res = append(res, dao.SkillRef{
			Rid:   level.CaseSets[i],
			Rtype: dao.RTypeCaseSet,
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

	ids := slice.Map(skillList, func(idx int, src dao.Skill) int64 {
		return src.Id
	})

	sls, err := s.skillDao.SkillLevelInfoByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	slm := mapx.NewMultiBuiltinMap[int64, dao.SkillLevel](len(ids))
	for _, sl := range sls {
		_ = slm.Put(sl.Sid, sl)
	}
	res := make([]domain.Skill, 0, len(skillList))
	for _, sk := range skillList {
		skSL, _ := slm.Get(sk.Id)
		res = append(res, s.skillToInfoDomain(sk, skSL, nil))
	}
	return res, nil
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
		slQueSets, _ := reqsMap.Get(fmt.Sprintf("%d_%s", sl.Id, dao.RTypeQuestionSet))
		slCaseSets, _ := reqsMap.Get(fmt.Sprintf("%d_%s", sl.Id, dao.RTypeCaseSet))
		dsl.Questions = slice.Map(slQues, func(idx int, src dao.SkillRef) int64 {
			return src.Rid
		})
		dsl.Cases = slice.Map(slCases, func(idx int, src dao.SkillRef) int64 {
			return src.Rid
		})
		dsl.QuestionSets = slice.Map(slQueSets, func(idx int, src dao.SkillRef) int64 {
			return src.Rid
		})
		dsl.CaseSets = slice.Map(slCaseSets, func(idx int, src dao.SkillRef) int64 {
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
