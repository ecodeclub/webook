package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/skill/internal/domain"
	"github.com/ecodeclub/webook/internal/skill/internal/repository/cache"
	dao "github.com/ecodeclub/webook/internal/skill/internal/repository/dao"
	"github.com/gotomicro/ego/core/elog"
	"golang.org/x/sync/errgroup"
)

type SkillRepo interface {
	// 管理端接口
	// Create 和 Update 返回值为  skillLevel的id
	Save(ctx context.Context, skill domain.Skill, skillLevels []domain.SkillLevel) (int64, error)
	UpdateRequest(ctx context.Context, skill domain.Skill, reqs []domain.SkillPreRequest) error
	// skill同步
	SyncSkill(ctx context.Context, skill domain.Skill, skillLevels []domain.SkillLevel) (int64, error)
	// id 为skillLevel的id
	SyncSKillRequest(ctx context.Context, skill domain.Skill, reqs []domain.SkillPreRequest) error
	// 列表
	List(ctx context.Context, offset, limit int) ([]domain.Skill, error)
	// 详情
	Info(ctx context.Context, id int64) (domain.Skill, error)
	Count(ctx context.Context) (int64, error)
	// c端
	Publist(ctx context.Context, offset int, limit int) ([]domain.Skill, error)
	PubCount(ctx context.Context) (int64, error)
	PubInfo(ctx context.Context, id int64) (domain.Skill, error)
}
type skillRepo struct {
	skillDao   dao.SkillDAO
	skillCache cache.SkillCache
	logger     *elog.Component
}

func NewSkillRepo(skillDao dao.SkillDAO, skillCache cache.SkillCache) SkillRepo {
	return &skillRepo{
		skillCache: skillCache,
		skillDao:   skillDao,
		logger:     elog.DefaultLogger,
	}
}

func (s *skillRepo) Save(ctx context.Context, skill domain.Skill, skillLevels []domain.SkillLevel) (int64, error) {
	var id int64
	var err error
	skillDao := s.skillToEntity(skill)
	levels := slice.Map(skillLevels, func(idx int, src domain.SkillLevel) dao.SkillLevel {
		return s.skillLevelToEntity(src)
	})
	if skill.ID == 0 {
		id, err = s.skillDao.Create(ctx, skillDao, levels)
	} else {
		id = skill.ID
		err = s.skillDao.Update(ctx, skillDao, levels)
	}
	return id, err

}

func (s *skillRepo) UpdateRequest(ctx context.Context, skill domain.Skill, reqs []domain.SkillPreRequest) error {
	if len(skill.Levels) != 1 {
		return errors.New("非法数据")
	}
	level := skill.Levels[0]
	reqDaos := slice.Map(reqs, func(idx int, src domain.SkillPreRequest) dao.SkillPreRequest {
		reqDao := s.skillPreRequestToEntity(src)
		reqDao.Sid = skill.ID
		reqDao.Slid = level.Id
		return reqDao
	})
	return s.skillDao.UpdateRequest(ctx, skill.ID, level.Id, reqDaos)
}

func (s *skillRepo) SyncSkill(ctx context.Context, skill domain.Skill, skillLevels []domain.SkillLevel) (int64, error) {
	skillDao := s.skillToEntity(skill)
	levels := slice.Map(skillLevels, func(idx int, src domain.SkillLevel) dao.SkillLevel {
		return s.skillLevelToEntity(src)
	})
	return s.skillDao.SyncSkill(ctx, skillDao, levels)
}

func (s *skillRepo) SyncSKillRequest(ctx context.Context, skill domain.Skill, reqs []domain.SkillPreRequest) error {
	if len(skill.Levels) != 1 {
		return errors.New("非法数据")
	}
	level := skill.Levels[0]
	reqDaos := slice.Map(reqs, func(idx int, src domain.SkillPreRequest) dao.SkillPreRequest {
		reqDao := s.skillPreRequestToEntity(src)
		reqDao.Sid = skill.ID
		reqDao.Slid = level.Id
		return reqDao
	})
	return s.skillDao.SyncSKillRequest(ctx, skill.ID, level.Id, reqDaos)
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
	var skillPreRequests []dao.SkillPreRequest
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
		skillPreRequests, err = s.skillDao.RequestInfo(ctx, id)
		return err
	})
	if err := eg.Wait(); err != nil {
		return domain.Skill{}, err
	}
	return s.skillToInfoDomain(skill, skillLevels, skillPreRequests), nil

}

func (s *skillRepo) Count(ctx context.Context) (int64, error) {
	return s.skillDao.Count(ctx)
}

func (s *skillRepo) Publist(ctx context.Context, offset int, limit int) ([]domain.Skill, error) {
	skills, err := s.skillDao.Publist(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	ans := make([]domain.Skill, 0, len(skills))
	for _, sk := range skills {
		ans = append(ans, s.skillToListDomain(dao.Skill(sk)))
	}
	return ans, nil

}

func (s *skillRepo) PubCount(ctx context.Context) (int64, error) {
	total, err := s.skillCache.GetTotal(ctx)
	if err == nil {
		return total, nil
	}
	count, err := s.skillDao.PubCount(ctx)
	if err != nil {
		return 0, err
	}
	err = s.skillCache.SetTotal(ctx, count)
	if err != nil {
		s.logger.Error("skill total 写入缓存失败")
	}
	return count, nil
}

func (s *skillRepo) PubInfo(ctx context.Context, id int64) (domain.Skill, error) {
	var eg errgroup.Group
	var skill dao.PubSkill
	var skillLevels []dao.PubSkillLevel
	var skillPreRequests []dao.PubSKillPreRequest
	eg.Go(func() error {
		var err error
		skill, err = s.skillDao.PubInfo(ctx, id)
		return err
	})
	eg.Go(func() error {
		var err error
		skillLevels, err = s.skillDao.PubLevels(ctx, id)
		return err
	})
	eg.Go(func() error {
		var err error
		skillPreRequests, err = s.skillDao.PubRequestInfo(ctx, id)
		return err
	})
	if err := eg.Wait(); err != nil {
		return domain.Skill{}, err
	}
	pubLevels := slice.Map(skillLevels, func(idx int, src dao.PubSkillLevel) dao.SkillLevel {
		return dao.SkillLevel(src)
	})
	pubReqs := slice.Map(skillPreRequests, func(idx int, src dao.PubSKillPreRequest) dao.SkillPreRequest {
		return dao.SkillPreRequest(src)
	})

	return s.skillToInfoDomain(dao.Skill(skill), pubLevels, pubReqs), nil
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
		Level: level.Level,
		Desc:  level.Desc,
		Ctime: time.UnixMilli(level.Ctime),
		Utime: time.UnixMilli(level.Utime),
	}
}
func (s *skillRepo) skillPreRequestToDomain(req dao.SkillPreRequest) domain.SkillPreRequest {
	return domain.SkillPreRequest{
		Id:    req.Id,
		Rid:   req.Rid,
		Rtype: req.Rtype,
		Ctime: time.UnixMilli(req.Ctime),
		Utime: time.UnixMilli(req.Utime),
	}
}

func (s *skillRepo) skillToInfoDomain(skill dao.Skill,
	levels []dao.SkillLevel, reqs []dao.SkillPreRequest) domain.Skill {
	skillMap := make(map[int64][]domain.SkillPreRequest, len(reqs))
	for _, req := range reqs {
		if _, ok := skillMap[req.Slid]; !ok {
			skillMap[req.Slid] = []domain.SkillPreRequest{
				s.skillPreRequestToDomain(req),
			}
		} else {
			skillMap[req.Slid] = append(skillMap[req.Slid], s.skillPreRequestToDomain(req))
		}
	}
	domainLevels := make([]domain.SkillLevel, 0, len(levels))
	for _, level := range levels {
		domainLevel := s.skillLevelToDomain(level)
		domainLevel.Requests = skillMap[level.Id]
		domainLevels = append(domainLevels, domainLevel)
	}
	skillDao := s.skillToListDomain(skill)
	skillDao.Levels = domainLevels
	return skillDao
}

func (s *skillRepo) skillToEntity(skill domain.Skill) dao.Skill {
	return dao.Skill{
		Id:     skill.ID,
		Labels: sqlx.JsonColumn[[]string]{Val: skill.Labels, Valid: len(skill.Labels) != 0},
		Name:   skill.Name,
		Desc:   skill.Desc,
		Base:   dao.Base{},
	}
}

func (s *skillRepo) skillLevelToEntity(skillLevel domain.SkillLevel) dao.SkillLevel {
	return dao.SkillLevel{
		Id:    skillLevel.Id,
		Level: skillLevel.Level,
		Desc:  skillLevel.Desc,
	}
}

func (s *skillRepo) skillPreRequestToEntity(req domain.SkillPreRequest) dao.SkillPreRequest {
	return dao.SkillPreRequest{
		Id:    req.Id,
		Rid:   req.Rid,
		Rtype: req.Rtype,
	}
}
