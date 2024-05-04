package service

import (
	"context"
	"fmt"
	"time"

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

const defaultSyncTimeout = 10 * time.Second

func (s *skillService) RefsByLevelIDs(ctx context.Context, ids []int64) ([]domain.SkillLevel, error) {
	return s.repo.RefsByLevelIDs(ctx, ids)
}

func (s *skillService) SaveRefs(ctx context.Context, skill domain.Skill) error {
	err := s.repo.SaveRefs(ctx, skill)
	if err != nil {
		return err
	}
	s.syncSkill(skill.ID)
	return nil
}

func (s *skillService) Save(ctx context.Context, skill domain.Skill) (int64, error) {
	id, err := s.repo.Save(ctx, skill)
	if err != nil {
		return 0, err
	}
	s.syncSkill(id)
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
		logger:   elog.DefaultLogger,
	}
}

func (s *skillService) syncSkill(id int64) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSyncTimeout)
	defer cancel()
	sk, err := s.repo.Info(ctx, id)
	fmt.Printf("开始发送 %d\n", id)
	if err != nil {
		s.logger.Error("发送同步搜索信息",
			elog.FieldErr(err),
		)
		return
	}
	evt := event.NewSkillEvent(sk)
	err = s.producer.Produce(ctx, evt)
	fmt.Println("发送成功")
	if err != nil {
		s.logger.Error("发送同步搜索信息",
			elog.FieldErr(err),
			elog.Any("event", evt),
		)
	}
}
