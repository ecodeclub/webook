package service

import (
	"context"
	"fmt"

	"github.com/ecodeclub/webook/internal/resume/internal/domain"
	"github.com/ecodeclub/webook/internal/resume/internal/repository"
)

type ExperienceService interface {
	SaveExperience(ctx context.Context, experience domain.Experience) (int64, error)
	List(ctx context.Context, uid int64) ([]domain.Experience, string, error)
	Delete(ctx context.Context, uid int64, id int64) error
}

type experienceService struct {
	experience repository.Experience
}

func NewExperienceService(experience repository.Experience) ExperienceService {
	return &experienceService{
		experience: experience,
	}
}

func (e *experienceService) SaveExperience(ctx context.Context, experience domain.Experience) (int64, error) {
	return e.experience.SaveExperience(ctx, experience)
}

func (e *experienceService) List(ctx context.Context, uid int64) ([]domain.Experience, string, error) {
	expList, err := e.experience.QueryAllExperiences(ctx, uid)
	if err != nil {
		return nil, "", err
	}
	msg := e.checkOverlap(expList)
	return expList, msg, nil
}

func (e *experienceService) Delete(ctx context.Context, uid int64, id int64) error {
	return e.experience.DeleteExperience(ctx, uid, id)
}

func (e *experienceService) checkOverlap(experience []domain.Experience) string {
	l := len(experience)

	for i := 1; i < l; i++ {
		if experience[i-1].End.Unix() > experience[i].Start.Unix() {
			return fmt.Sprintf("第%d段工作经历和第%d段工作经历有重合，请提前准备好工作经历重合的理由", i-1, i)
		}
	}
	return ""
}
