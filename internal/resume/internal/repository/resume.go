package repository

import (
	"context"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/resume/internal/domain"
	"github.com/ecodeclub/webook/internal/resume/internal/repository/dao"
	"golang.org/x/sync/errgroup"
)

type ResumeProjectRepo interface {
	SaveProject(ctx context.Context, pro domain.Project) (int64, error)
	// 删除project及其所有关联数据
	DeleteProject(ctx context.Context, uid, id int64) error
	FindProjects(ctx context.Context, uid int64) ([]domain.Project, error)
	ProjectInfo(ctx context.Context, id int64) (domain.Project, error)
	SaveContribution(ctx context.Context, id int64, contribution domain.Contribution) (int64, error)
	// 删除职责
	DeleteContribution(ctx context.Context, id int64) error
	// 保存难点
	SaveDifficulty(ctx context.Context, id int64, difficulty domain.Difficulty) error
	// 删除难点
	DeleteDifficulty(ctx context.Context, id int64) error
}

type resumeProjectRepo struct {
	pdao dao.ResumeProjectDAO
}

func NewResumeProjectRepo(pdao dao.ResumeProjectDAO) ResumeProjectRepo {
	return &resumeProjectRepo{pdao: pdao}
}

func (r *resumeProjectRepo) SaveContribution(ctx context.Context, id int64, contribution domain.Contribution) (int64, error) {
	contributionDao := r.toContributionEntity(contribution)
	contributionDao.ProjectID = id
	refcases := slice.Map(contribution.RefCases, func(idx int, src domain.Case) dao.RefCase {
		return r.toRefCaseEntity(src)
	})
	return r.pdao.SaveContribution(ctx, contributionDao, refcases)
}

func (r *resumeProjectRepo) DeleteContribution(ctx context.Context, id int64) error {
	return r.pdao.DeleteContribution(ctx, id)
}

func (r *resumeProjectRepo) SaveDifficulty(ctx context.Context, id int64, difficulty domain.Difficulty) error {
	difficultyDao := r.toDifficultyEntity(difficulty)
	difficultyDao.ProjectID = id
	return r.pdao.SaveDifficulty(ctx, difficultyDao)
}

func (r *resumeProjectRepo) DeleteDifficulty(ctx context.Context, id int64) error {
	return r.pdao.DeleteDifficulty(ctx, id)
}

func (r *resumeProjectRepo) SaveProject(ctx context.Context, pro domain.Project) (int64, error) {
	return r.pdao.Upsert(ctx, r.toProjectEntity(pro))
}

func (r *resumeProjectRepo) DeleteProject(ctx context.Context, uid, id int64) error {
	return r.pdao.Delete(ctx, uid, id)
}

func (r *resumeProjectRepo) FindProjects(ctx context.Context, uid int64) ([]domain.Project, error) {
	pList, err := r.pdao.Find(ctx, uid)
	if err != nil {
		return nil, err
	}

	ids := slice.Map(pList, func(idx int, src dao.ResumeProject) int64 {
		return src.ID
	})
	var eg errgroup.Group
	var contributionMap map[int64][]dao.Contribution
	var difficultyMap map[int64][]dao.Difficulty
	eg.Go(func() error {
		var eerr error
		contributionMap, eerr = r.pdao.BatchFindContributions(ctx, ids)
		return eerr
	})
	eg.Go(func() error {
		var eerr error
		difficultyMap, eerr = r.pdao.BatchFindDifficulty(ctx, ids)
		return eerr
	})
	err = eg.Wait()
	if err != nil {
		return nil, err
	}
	contributionIds := make([]int64, 0, 16)
	for _, contributions := range contributionMap {
		cids := slice.Map(contributions, func(idx int, src dao.Contribution) int64 {
			return src.ID
		})
		contributionIds = append(contributionIds, cids...)
	}
	caMap, err := r.pdao.FindRefCases(ctx, contributionIds)
	if err != nil {
		return nil, err
	}
	projects := slice.Map(pList, func(idx int, project dao.ResumeProject) domain.Project {
		pro := r.toProjectDomain(project)
		contributions, ok := contributionMap[project.ID]
		if ok {
			pro.Contributions = slice.Map(contributions, func(idx int, src dao.Contribution) domain.Contribution {
				cas := caMap[src.ID]
				return r.toContributionDomain(src, cas)
			})
		}
		diffs, ok := difficultyMap[project.ID]
		if ok {
			pro.Difficulties = slice.Map(diffs, func(idx int, src dao.Difficulty) domain.Difficulty {
				return r.toDifficultyDomain(src)
			})
		}
		return pro
	})
	return projects, nil
}

func (r *resumeProjectRepo) ProjectInfo(ctx context.Context, id int64) (domain.Project, error) {
	var eg errgroup.Group
	var project dao.ResumeProject
	var refCasesMap map[int64][]dao.RefCase
	var contributions []dao.Contribution
	var difficulties []dao.Difficulty
	eg.Go(func() error {
		var err error
		project, err = r.pdao.First(ctx, id)
		return err
	})
	eg.Go(func() error {
		var err error
		contributions, err = r.pdao.FindContributions(ctx, id)
		return err
	})
	eg.Go(func() error {
		var err error
		difficulties, err = r.pdao.FindDifficulties(ctx, id)
		return err
	})
	err := eg.Wait()
	if err != nil {
		return domain.Project{}, err
	}
	contirbutionIds := slice.Map(contributions, func(idx int, src dao.Contribution) int64 {
		return src.ID
	})
	refCasesMap, err = r.pdao.FindRefCases(ctx, contirbutionIds)
	if err != nil {
		return domain.Project{}, err
	}
	pro := r.toProjectDomain(project)
	pro.Contributions = slice.Map(contributions, func(idx int, src dao.Contribution) domain.Contribution {
		cas := refCasesMap[src.ID]
		return r.toContributionDomain(src, cas)
	})
	pro.Difficulties = slice.Map(difficulties, func(idx int, src dao.Difficulty) domain.Difficulty {
		return r.toDifficultyDomain(src)
	})
	return pro, nil
}

func (r *resumeProjectRepo) toProjectEntity(project domain.Project) dao.ResumeProject {
	return dao.ResumeProject{
		ID:           project.Id,
		StartTime:    project.StartTime,
		EndTime:      project.EndTime,
		Uid:          project.Uid,
		Name:         project.Name,
		Introduction: project.Introduction,
		Core:         project.Core,
	}
}

func (r *resumeProjectRepo) toProjectDomain(p dao.ResumeProject) domain.Project {
	return domain.Project{
		Id:           p.ID,
		StartTime:    p.StartTime,
		EndTime:      p.EndTime,
		Uid:          p.Uid,
		Name:         p.Name,
		Introduction: p.Introduction,
		Core:         p.Core,
	}
}

func (r *resumeProjectRepo) toContributionDomain(contribution dao.Contribution, cas []dao.RefCase) domain.Contribution {
	cases := slice.Map(cas, func(idx int, src dao.RefCase) domain.Case {
		return r.toRefCaseDomain(src)
	})
	return domain.Contribution{
		ID:       contribution.ID,
		Type:     contribution.Type,
		Desc:     contribution.Desc,
		RefCases: cases,
	}
}

func (r *resumeProjectRepo) toRefCaseDomain(ca dao.RefCase) domain.Case {
	return domain.Case{
		Id:        ca.CaseID,
		Highlight: ca.Highlight,
		Level:     ca.Level,
	}
}

func (r *resumeProjectRepo) toDifficultyDomain(difficulty dao.Difficulty) domain.Difficulty {
	return domain.Difficulty{
		ID:   difficulty.ID,
		Desc: difficulty.Desc,
		Case: domain.Case{
			Id:    difficulty.CaseID,
			Level: difficulty.Level,
		},
	}
}

func (r *resumeProjectRepo) toContributionEntity(contribution domain.Contribution) dao.Contribution {
	return dao.Contribution{
		ID:   contribution.ID,
		Type: contribution.Type,
		Desc: contribution.Desc,
	}
}

func (r *resumeProjectRepo) toRefCaseEntity(refCase domain.Case) dao.RefCase {
	return dao.RefCase{
		CaseID:    refCase.Id,
		Highlight: refCase.Highlight,
		Level:     refCase.Level,
	}
}

func (r *resumeProjectRepo) toDifficultyEntity(difficulty domain.Difficulty) dao.Difficulty {
	return dao.Difficulty{
		ID:     difficulty.ID,
		Desc:   difficulty.Desc,
		Level:  difficulty.Case.Level,
		CaseID: difficulty.Case.Id,
	}
}
