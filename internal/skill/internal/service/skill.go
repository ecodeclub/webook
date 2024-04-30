package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/skill/internal/event"
	"github.com/gotomicro/ego/core/elog"

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
	RefsByLevelIDs(ctx context.Context, ids []int64) ([]domain.SkillLevel, error)
}

type skillService struct {
	repo     repository.SkillRepo
	producer event.SyncEventProducer
	logger   *elog.Component
}

func (s *skillService) RefsByLevelIDs(ctx context.Context, ids []int64) ([]domain.SkillLevel, error) {
	return s.repo.RefsByLevelIDs(ctx, ids)
}

func (s *skillService) SaveRefs(ctx context.Context, skill domain.Skill) error {
	err := s.repo.SaveRefs(ctx, skill)
	if err != nil {
		return err
	}
	evt := event.NewSkillEvent(skill)
	err = s.producer.Produce(ctx, evt)
	if err != nil {
		s.logger.Error("发送技能搜索信息",
			elog.FieldErr(err),
			elog.Any("event", evt),
		)
	}
	return nil
}

func (s *skillService) Save(ctx context.Context, skill domain.Skill) (int64, error) {
	id, err := s.repo.Save(ctx, skill)
	if err != nil {
		return 0, err
	}
	evt := event.NewSkillEvent(skill)
	err = s.producer.Produce(ctx, evt)
	if err != nil {
		s.logger.Error("发送技能搜索信息",
			elog.FieldErr(err),
			elog.Any("event", evt),
		)
	}
	return id, nil
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

func NewSkillService(repo repository.SkillRepo, p event.SyncEventProducer) SkillService {
	return &skillService{
		repo:     repo,
		producer: p,
	}
}
