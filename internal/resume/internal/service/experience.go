package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/resume/internal/domain"
	"github.com/ecodeclub/webook/internal/resume/internal/repository"
)

type ExperienceService interface {
	SaveExperience(ctx context.Context, experience domain.Experience) (int64, error)
	List(ctx context.Context, uid int64) ([]domain.Experience, error)
	Delete(ctx context.Context, uid int64, id int64) error
}

type experienceService struct {
	experience repository.Experience
}

//func NewExperienceService(experience repository.Experience) ExperienceService {
//	return &experienceService{
//		experience: experience,
//	}
//}

func (e *experienceService) SaveExperience(ctx context.Context, experience domain.Experience) (int64, error) {
	return e.experience.SaveExperience(ctx, experience)
}

func (e *experienceService) List(ctx context.Context, uid int64) ([]domain.Experience, error) {
	return e.experience.QueryAllExperiences(ctx, uid)
}

func (e *experienceService) Delete(ctx context.Context, uid int64, id int64) error {
	return e.experience.DeleteExperience(ctx, uid, id)
}
