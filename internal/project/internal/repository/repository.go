// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repository

import (
	"context"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/project/internal/domain"
	"github.com/ecodeclub/webook/internal/project/internal/repository/dao"
	"golang.org/x/sync/errgroup"
)

// Repository C 端接口
type Repository interface {
	List(ctx context.Context, offset int, limit int) ([]domain.Project, error)
	Count(ctx context.Context) (int64, error)
	Detail(ctx context.Context, id int64) (domain.Project, error)
	Brief(ctx context.Context, id int64) (domain.Project, error)
}

var _ Repository = &CachedRepository{}

type CachedRepository struct {
	dao dao.ProjectDAO
}

func (repo *CachedRepository) Count(ctx context.Context) (int64, error) {
	return repo.dao.Count(ctx)
}

func (repo *CachedRepository) Brief(ctx context.Context, id int64) (domain.Project, error) { //TODO implement me
	prj, err := repo.dao.BriefById(ctx, id)
	return repo.prjToDomain(prj, nil, nil, nil, nil, nil), err
}

func (repo *CachedRepository) Detail(ctx context.Context, id int64) (domain.Project, error) { //TODO implement me
	var (
		eg      errgroup.Group
		prj     dao.PubProject
		resumes []dao.PubProjectResume
		diffs   []dao.PubProjectDifficulty
		ques    []dao.PubProjectQuestion
		intrs   []dao.PubProjectIntroduction
		combos  []dao.PubProjectCombo
	)
	eg.Go(func() error {
		var err error
		prj, err = repo.dao.GetById(ctx, id)
		return err
	})

	eg.Go(func() error {
		var err error
		resumes, err = repo.dao.Resumes(ctx, id)
		return err
	})

	eg.Go(func() error {
		var err error
		diffs, err = repo.dao.Difficulties(ctx, id)
		return err
	})

	eg.Go(func() error {
		var err error
		ques, err = repo.dao.Questions(ctx, id)
		return err
	})

	eg.Go(func() error {
		var err error
		intrs, err = repo.dao.Introductions(ctx, id)
		return err
	})

	eg.Go(func() error {
		var err error
		combos, err = repo.dao.Combos(ctx, id)
		return err
	})

	err := eg.Wait()
	return repo.prjToDomain(prj, resumes, diffs, ques, intrs, combos), err
}

func (repo *CachedRepository) List(ctx context.Context, offset int, limit int) ([]domain.Project, error) {
	res, err := repo.dao.List(ctx, offset, limit)
	return slice.Map(res, func(idx int, src dao.PubProject) domain.Project {
		return repo.prjToDomain(src, nil, nil, nil, nil, nil)
	}), err
}

func (repo *CachedRepository) prjToDomain(prj dao.PubProject,
	resumes []dao.PubProjectResume,
	diff []dao.PubProjectDifficulty,
	ques []dao.PubProjectQuestion,
	intrs []dao.PubProjectIntroduction,
	combos []dao.PubProjectCombo,
) domain.Project {
	return domain.Project{
		Id:             prj.Id,
		SN:             prj.SN,
		Title:          prj.Title,
		Overview:       prj.Overview,
		SystemDesign:   prj.SystemDesign,
		GithubRepo:     prj.GithubRepo,
		GiteeRepo:      prj.GiteeRepo,
		RefQuestionSet: prj.RefQuestionSet,
		Status:         domain.ProjectStatus(prj.Status),
		Labels:         prj.Labels.Val,
		Desc:           prj.Desc,
		Utime:          prj.Utime,
		ProductSPU:     prj.ProductSPU.String,
		CodeSPU:        prj.CodeSPU.String,
		Resumes: slice.Map(resumes, func(idx int, src dao.PubProjectResume) domain.Resume {
			return repo.rsmToDomain(src)
		}),
		Difficulties: slice.Map(diff, func(idx int, src dao.PubProjectDifficulty) domain.Difficulty {
			return repo.diffToDomain(src)
		}),
		Questions: slice.Map(ques, func(idx int, src dao.PubProjectQuestion) domain.Question {
			return repo.queToDomain(src)
		}),
		Introductions: slice.Map(intrs, func(idx int, src dao.PubProjectIntroduction) domain.Introduction {
			return repo.intrToDomain(src)
		}),
		Combos: slice.Map(combos, func(idx int, src dao.PubProjectCombo) domain.Combo {
			return repo.comboToDomain(src)
		}),
	}
}

func (repo *CachedRepository) comboToDomain(c dao.PubProjectCombo) domain.Combo {
	return domain.Combo{
		Id:      c.Id,
		Title:   c.Title,
		Content: c.Content,
		Utime:   c.Utime,
		Status:  domain.ComboStatus(c.Status),
	}
}

func (repo *CachedRepository) intrToDomain(r dao.PubProjectIntroduction) domain.Introduction {
	return domain.Introduction{
		Id:       r.Id,
		Role:     r.Role,
		Content:  r.Content,
		Analysis: r.Analysis,
		Status:   domain.IntroductionStatus(r.Status),
		Utime:    time.UnixMilli(r.Utime),
	}
}

func (repo *CachedRepository) rsmToDomain(r dao.PubProjectResume) domain.Resume {
	return domain.Resume{
		Id:       r.Id,
		Role:     r.Role,
		Content:  r.Content,
		Analysis: r.Analysis,
		Status:   domain.ResumeStatus(r.Status),
		Utime:    time.UnixMilli(r.Utime),
	}
}

func (repo *CachedRepository) diffToDomain(d dao.PubProjectDifficulty) domain.Difficulty {
	return domain.Difficulty{
		Id:       d.Id,
		Title:    d.Title,
		Content:  d.Content,
		Analysis: d.Analysis,
		Status:   domain.DifficultyStatus(d.Status),
		Utime:    time.UnixMilli(d.Utime),
	}
}

func (repo *CachedRepository) queToDomain(d dao.PubProjectQuestion) domain.Question {
	return domain.Question{
		Id:       d.Id,
		Title:    d.Title,
		Analysis: d.Analysis,
		Status:   domain.QuestionStatus(d.Status),
		Answer:   d.Answer,
		Utime:    time.UnixMilli(d.Utime),
	}
}

func NewCachedRepository(dao dao.ProjectDAO) Repository {
	return &CachedRepository{dao: dao}
}
