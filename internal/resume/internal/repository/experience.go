package repository

import (
	"context"
	"time"

	"github.com/ecodeclub/ekit/sqlx"

	"github.com/ecodeclub/webook/internal/resume/internal/domain"
	"github.com/ecodeclub/webook/internal/resume/internal/repository/dao"
)

type Experience interface {
	SaveExperience(ctx context.Context, experience domain.Experience) (int64, error)
	QueryAllExperiences(ctx context.Context, uid int64) ([]domain.Experience, error)
	DeleteExperience(ctx context.Context, uid int64, id int64) error
}

type experience struct {
	expdao dao.ExperienceDAO
}

func NewExperience(exp dao.ExperienceDAO) Experience {
	return &experience{
		expdao: exp,
	}
}

func (e *experience) SaveExperience(ctx context.Context, experience domain.Experience) (int64, error) {
	return e.expdao.Upsert(ctx, e.toExperienceEntity(experience))
}

func (e *experience) QueryAllExperiences(ctx context.Context, uid int64) ([]domain.Experience, error) {
	elist, err := e.expdao.Find(ctx, uid)
	if err != nil {
		return nil, err
	}

	ans := make([]domain.Experience, 0, len(elist))
	for _, exp := range elist {
		ans = append(ans, e.toExperienceDomain(exp))
	}
	return ans, nil
}

func (e *experience) DeleteExperience(ctx context.Context, uid int64, id int64) error {
	return e.expdao.Delete(ctx, uid, id)
}

func (e *experience) toExperienceEntity(experience domain.Experience) dao.Experience {
	return dao.Experience{
		ID:          experience.Id,
		StartTime:   experience.Start.UnixMilli(),
		EndTime:     experience.End.UnixMilli(),
		Title:       experience.Title,
		CompanyName: experience.CompanyName,
		Location:    experience.Location,
		Responsibilities: sqlx.JsonColumn[[]domain.Responsibility]{
			Valid: true,
			Val:   experience.Responsibilities,
		},
		Accomplishments: sqlx.JsonColumn[[]domain.Accomplishment]{
			Valid: true,
			Val:   experience.Accomplishments,
		},
		Skills: sqlx.JsonColumn[[]string]{
			Valid: true,
			Val:   experience.Skills,
		},
	}
}

func (e *experience) toExperienceDomain(experience dao.Experience) domain.Experience {
	return domain.Experience{
		Id:               experience.ID,
		Start:            time.UnixMilli(experience.StartTime),
		End:              time.UnixMilli(experience.EndTime),
		Title:            experience.Title,
		CompanyName:      experience.CompanyName,
		Location:         experience.Location,
		Responsibilities: experience.Responsibilities.Val,
		Accomplishments:  experience.Accomplishments.Val,
		Skills:           experience.Skills.Val,
	}
}
