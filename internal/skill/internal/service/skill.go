package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/skill/internal/domain"
	"github.com/ecodeclub/webook/internal/skill/internal/repository"
)

type SkillService interface {
	// Save 保存基本信息
	Save(ctx context.Context, skill domain.Skill) (int64, error)
	// SaveRefs 保存关联信息
	SaveRefs(ctx context.Context, skill domain.Skill) error
	List(ctx context.Context, offset, limit int) ([]domain.Skill, int64, error)
	Info(ctx context.Context, id int64) (domain.Skill, error)
}

type skillService struct {
	repo repository.SkillRepo
}

func (s *skillService) SaveRefs(ctx context.Context, skill domain.Skill) error {
	return s.repo.SaveRefs(ctx, skill)
}

func (s *skillService) Save(ctx context.Context, skill domain.Skill) (int64, error) {
	return s.repo.Save(ctx, skill)
}

func (s *skillService) List(ctx context.Context, offset, limit int) ([]domain.Skill, int64, error) {
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

func (s *skillService) Info(ctx context.Context, id int64) (domain.Skill, error) {
	return s.repo.Info(ctx, id)
}

func NewSkillService(repo repository.SkillRepo) SkillService {
	return &skillService{
		repo: repo,
	}
}
