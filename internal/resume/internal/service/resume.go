package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/resume/internal/domain"
	"github.com/ecodeclub/webook/internal/resume/internal/repository"
)

type Service interface {
	SaveProject(ctx context.Context, pro domain.Project) (int64, error)
	// 删除project及其所有关联数据
	DeleteProject(ctx context.Context, uid, id int64) error
	FindProjects(ctx context.Context, uid int64) ([]domain.Project, error)
	ProjectInfo(ctx context.Context, id int64) (domain.Project, error)
	SaveContribution(ctx context.Context, id int64, contribution domain.Contribution) error
	// 删除职责
	DeleteContribution(ctx context.Context, id int64) error
	// 保存难点
	SaveDifficulty(ctx context.Context, id int64, difficulty domain.Difficulty) error
	// 删除难点
	DeleteDifficulty(ctx context.Context, id int64) error
}

type service struct {
	repo repository.ResumeProjectRepo
}

func NewService(repo repository.ResumeProjectRepo) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) SaveProject(ctx context.Context, pro domain.Project) (int64, error) {
	return s.repo.SaveProject(ctx, pro)
}

func (s *service) DeleteProject(ctx context.Context, uid, id int64) error {
	return s.repo.DeleteProject(ctx, uid, id)
}

func (s *service) FindProjects(ctx context.Context, uid int64) ([]domain.Project, error) {
	return  s.repo.FindProjects(ctx, uid)

}

func (s *service) ProjectInfo(ctx context.Context, id int64) (domain.Project, error) {
	return s.repo.ProjectInfo(ctx, id)
}

func (s *service) SaveContribution(ctx context.Context, id int64, contribution domain.Contribution) error {
	return s.repo.SaveContribution(ctx, id, contribution)
}

func (s *service) DeleteContribution(ctx context.Context, id int64) error {
	return s.repo.DeleteContribution(ctx, id)
}

func (s *service) SaveDifficulty(ctx context.Context, id int64, difficulty domain.Difficulty) error {
	return s.repo.SaveDifficulty(ctx, id, difficulty)
}

func (s *service) DeleteDifficulty(ctx context.Context, id int64) error {
	return s.repo.DeleteDifficulty(ctx, id)
}
