package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/skill/internal/domain"
	"github.com/ecodeclub/webook/internal/skill/internal/repository"
	"golang.org/x/sync/errgroup"
)

type SkillSvc interface {
	Save(ctx context.Context, skill domain.Skill, skillLevels []domain.SkillLevel) (int64, error)
	UpdateRequest(ctx context.Context, skill domain.Skill, reqs []domain.SkillPreRequest) error
	// skill同步
	SyncSkill(ctx context.Context, skill domain.Skill, skillLevels []domain.SkillLevel) (int64, error)
	// id 为skillLevel的id
	SyncSKillRequest(ctx context.Context, skill domain.Skill, reqs []domain.SkillPreRequest) error
	// 列表
	List(ctx context.Context, offset, limit int) ([]domain.Skill, int64, error)
	// 详情
	Info(ctx context.Context, id int64) (domain.Skill, error)
	// c端
	Publist(ctx context.Context, offset int, limit int) ([]domain.Skill, int64, error)
	PubInfo(ctx context.Context, id int64) (domain.Skill, error)
}

type skillSvc struct {
	repo repository.SkillRepo
}

func NewSkillSvc(repo repository.SkillRepo) SkillSvc {
	return &skillSvc{
		repo: repo,
	}
}

func (s *skillSvc) Save(ctx context.Context, skill domain.Skill, skillLevels []domain.SkillLevel) (int64, error) {
	return s.repo.Save(ctx, skill, skillLevels)
}

func (s *skillSvc) UpdateRequest(ctx context.Context, skill domain.Skill, reqs []domain.SkillPreRequest) error {
	return s.repo.UpdateRequest(ctx, skill, reqs)
}

func (s *skillSvc) SyncSkill(ctx context.Context, skill domain.Skill, skillLevels []domain.SkillLevel) (int64, error) {
	return s.repo.SyncSkill(ctx, skill, skillLevels)
}

func (s *skillSvc) SyncSKillRequest(ctx context.Context, skill domain.Skill, reqs []domain.SkillPreRequest) error {
	return s.repo.SyncSKillRequest(ctx, skill, reqs)
}

func (s *skillSvc) List(ctx context.Context, offset, limit int) ([]domain.Skill, int64, error) {
	count, err := s.repo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	skills, err := s.repo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	return skills, count, nil
}

func (s *skillSvc) Info(ctx context.Context, id int64) (domain.Skill, error) {
	return s.repo.Info(ctx, id)
}

func (s *skillSvc) Publist(ctx context.Context, offset int, limit int) ([]domain.Skill, int64, error) {
	var eg errgroup.Group
	var skills []domain.Skill
	var count int64
	eg.Go(func() error {
		var err error
		skills, err = s.repo.Publist(ctx, offset, limit)
		return err
	})
	eg.Go(func() error {
		var err error
		count, err = s.repo.PubCount(ctx)
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, 0, err
	}
	return skills, count, nil

}

func (s *skillSvc) PubInfo(ctx context.Context, id int64) (domain.Skill, error) {
	return s.repo.PubInfo(ctx, id)
}
