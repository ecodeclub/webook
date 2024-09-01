package repository

import (
	"context"
	"encoding/json"

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
	responsibilitiesJsonData, err := json.Marshal(experience.Responsibilities)
	if err != nil {
		responsibilitiesJsonData = nil
	}
	accomplishmentsJsonData, err := json.Marshal(experience.Accomplishments)
	if err != nil {
		accomplishmentsJsonData = nil
	}
	skillsJsonData, err := json.Marshal(experience.Skills)
	if err != nil {
		skillsJsonData = nil
	}

	return dao.Experience{
		ID:               experience.Id,
		StartTime:        experience.Start,
		EndTime:          experience.End,
		Title:            experience.Title,
		CompanyName:      experience.CompanyName,
		Location:         experience.Location,
		Responsibilities: string(responsibilitiesJsonData),
		Accomplishments:  string(accomplishmentsJsonData),
		Skills:           string(skillsJsonData),
	}
}

func (e *experience) toExperienceDomain(experience dao.Experience) domain.Experience {
	var responsibilities []domain.Responsibility
	err := json.Unmarshal([]byte(experience.Responsibilities), &responsibilities)
	if err != nil {
		responsibilities = nil
	}

	var accomplishments []domain.Accomplishment
	err = json.Unmarshal([]byte(experience.Accomplishments), &accomplishments)
	if err != nil {
		accomplishments = nil
	}

	var skills []string
	err = json.Unmarshal([]byte(experience.Skills), &skills)
	if err != nil {
		skills = nil
	}

	return domain.Experience{
		Id:               experience.ID,
		Start:            experience.StartTime,
		End:              experience.EndTime,
		Title:            experience.Title,
		CompanyName:      experience.CompanyName,
		Location:         experience.Location,
		Responsibilities: responsibilities,
		Accomplishments:  accomplishments,
		Skills:           skills,
	}
}
