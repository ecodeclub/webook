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
	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/project/internal/domain"
	"github.com/ecodeclub/webook/internal/project/internal/repository/dao"
	"golang.org/x/sync/errgroup"
)

// ProjectAdminRepository 管理员操作的
type ProjectAdminRepository interface {
	Save(ctx context.Context, prj domain.Project) (int64, error)
	List(ctx context.Context, offset int, limit int) ([]domain.Project, error)
	Count(ctx context.Context) (int64, error)
	Detail(ctx context.Context, id int64) (domain.Project, error)
	Sync(ctx context.Context, prj domain.Project) (int64, error)

	ResumeSave(ctx context.Context, pid int64, resume domain.Resume) (int64, error)
	ResumeDetail(ctx context.Context, id int64) (domain.Resume, error)
	ResumePublish(ctx context.Context, pid int64, resume domain.Resume) (int64, error)

	DifficultySave(ctx context.Context, pid int64, diff domain.Difficulty) (int64, error)
	DifficultyDetail(ctx context.Context, id int64) (domain.Difficulty, error)
	DifficultyPublish(ctx context.Context, pid int64, diff domain.Difficulty) (int64, error)

	QuestionSave(ctx context.Context, pid int64, que domain.Question) (int64, error)
	QuestionDetail(ctx context.Context, id int64) (domain.Question, error)
	QuestionSync(ctx context.Context, pid int64, que domain.Question) (int64, error)

	IntroductionSave(ctx context.Context, pid int64, intr domain.Introduction) (int64, error)
	IntroductionDetail(ctx context.Context, id int64) (domain.Introduction, error)
	IntroductionSync(ctx context.Context, pid int64, intr domain.Introduction) (int64, error)
	ComboSave(ctx context.Context, pid int64, c domain.Combo) (int64, error)
	ComboDetail(ctx context.Context, cid int64) (domain.Combo, error)
	ComboSync(ctx context.Context, pid int64, c domain.Combo) (int64, error)
}

var _ ProjectAdminRepository = (*projectAdminRepository)(nil)

type projectAdminRepository struct {
	dao dao.ProjectAdminDAO
}

func (repo *projectAdminRepository) ComboSync(ctx context.Context, pid int64, c domain.Combo) (int64, error) {
	entity := repo.comboToEntity(c)
	entity.Pid = pid
	return repo.dao.ComboSync(ctx, entity)
}

func (repo *projectAdminRepository) ComboDetail(ctx context.Context, cid int64) (domain.Combo, error) {
	c, err := repo.dao.ComboById(ctx, cid)
	return repo.comboToDomain(c), err
}

func (repo *projectAdminRepository) ComboSave(ctx context.Context, pid int64, c domain.Combo) (int64, error) {
	entity := repo.comboToEntity(c)
	entity.Pid = pid
	return repo.dao.ComboSave(ctx, entity)
}

func (repo *projectAdminRepository) ResumePublish(ctx context.Context, pid int64, resume domain.Resume) (int64, error) {
	entity := repo.rsmToEntity(resume)
	entity.Pid = pid
	return repo.dao.ResumeSync(ctx, entity)
}

func (repo *projectAdminRepository) QuestionSync(ctx context.Context, pid int64, que domain.Question) (int64, error) {
	entity := repo.queToEntity(que)
	entity.Pid = pid
	return repo.dao.QuestionSync(ctx, entity)
}

func (repo *projectAdminRepository) DifficultyPublish(ctx context.Context, pid int64, diff domain.Difficulty) (int64, error) {
	entity := repo.diffToEntity(diff)
	entity.Pid = pid
	return repo.dao.DifficultySync(ctx, entity)
}

func (repo *projectAdminRepository) IntroductionSync(ctx context.Context, pid int64, intr domain.Introduction) (int64, error) {
	entity := repo.intrToEntity(intr)
	entity.Pid = pid
	return repo.dao.IntroductionSync(ctx, entity)
}

func (repo *projectAdminRepository) IntroductionDetail(ctx context.Context, id int64) (domain.Introduction, error) {
	res, err := repo.dao.IntroductionById(ctx, id)
	return repo.intrToDomain(res), err
}

func (repo *projectAdminRepository) IntroductionSave(ctx context.Context, pid int64, intr domain.Introduction) (int64, error) {
	entity := repo.intrToEntity(intr)
	entity.Pid = pid
	return repo.dao.IntroductionSave(ctx, entity)
}

func (repo *projectAdminRepository) Sync(ctx context.Context, prj domain.Project) (int64, error) {
	return repo.dao.Sync(ctx, repo.prjToEntity(prj))
}

func (repo *projectAdminRepository) QuestionDetail(ctx context.Context, id int64) (domain.Question, error) {
	res, err := repo.dao.QuestionById(ctx, id)
	return repo.queToDomain(res), err
}

func (repo *projectAdminRepository) QuestionSave(ctx context.Context, pid int64, que domain.Question) (int64, error) {
	entity := repo.queToEntity(que)
	entity.Pid = pid
	return repo.dao.QuestionSave(ctx, entity)
}

func (repo *projectAdminRepository) DifficultyDetail(ctx context.Context,
	id int64) (domain.Difficulty, error) {
	res, err := repo.dao.DifficultyById(ctx, id)
	return repo.diffToDomain(res), err
}

func (repo *projectAdminRepository) DifficultySave(ctx context.Context, pid int64, diff domain.Difficulty) (int64, error) {
	entity := repo.diffToEntity(diff)
	entity.Pid = pid
	return repo.dao.DifficultySave(ctx, entity)
}

func (repo *projectAdminRepository) ResumeDetail(ctx context.Context, id int64) (domain.Resume, error) {
	res, err := repo.dao.ResumeById(ctx, id)
	return repo.rsmToDomain(res), err
}

func (repo *projectAdminRepository) ResumeSave(ctx context.Context, pid int64, resume domain.Resume) (int64, error) {
	entity := repo.rsmToEntity(resume)
	entity.Pid = pid
	return repo.dao.ResumeSave(ctx, entity)
}

func (repo *projectAdminRepository) Detail(ctx context.Context, id int64) (domain.Project, error) {
	var (
		eg      errgroup.Group
		prj     dao.Project
		resumes []dao.ProjectResume
		diffs   []dao.ProjectDifficulty
		ques    []dao.ProjectQuestion
		intrs   []dao.ProjectIntroduction
		combos  []dao.ProjectCombo
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

func (repo *projectAdminRepository) Count(ctx context.Context) (int64, error) {
	return repo.dao.Count(ctx)
}

func (repo *projectAdminRepository) List(ctx context.Context, offset int, limit int) ([]domain.Project, error) {
	res, err := repo.dao.List(ctx, offset, limit)
	return slice.Map(res, func(idx int, src dao.Project) domain.Project {
		return repo.prjToDomain(src, nil, nil, nil, nil, nil)
	}), err
}

func (repo *projectAdminRepository) Save(ctx context.Context, prj domain.Project) (int64, error) {
	return repo.dao.Save(ctx, repo.prjToEntity(prj))
}

func (repo *projectAdminRepository) diffToEntity(d domain.Difficulty) dao.ProjectDifficulty {
	return dao.ProjectDifficulty{
		Id:       d.Id,
		Title:    d.Title,
		Content:  d.Content,
		Analysis: d.Analysis,
		Status:   d.Status.ToUint8(),
		Utime:    d.Utime.UnixMilli(),
	}
}

func (repo *projectAdminRepository) intrToEntity(r domain.Introduction) dao.ProjectIntroduction {
	return dao.ProjectIntroduction{
		Id:       r.Id,
		Role:     r.Role,
		Content:  r.Content,
		Analysis: r.Analysis,
		Status:   r.Status.ToUint8(),
		Utime:    r.Utime.UnixMilli(),
	}
}

func (repo *projectAdminRepository) rsmToEntity(r domain.Resume) dao.ProjectResume {
	return dao.ProjectResume{
		Id:       r.Id,
		Role:     r.Role,
		Content:  r.Content,
		Analysis: r.Analysis,
		Status:   r.Status.ToUint8(),
		Utime:    r.Utime.UnixMilli(),
	}
}

func (repo *projectAdminRepository) prjToEntity(prj domain.Project) dao.Project {
	return dao.Project{
		Id:             prj.Id,
		SN:             prj.SN,
		Title:          prj.Title,
		Status:         prj.Status.ToUint8(),
		Overview:       prj.Overview,
		SystemDesign:   prj.SystemDesign,
		GithubRepo:     prj.GithubRepo,
		GiteeRepo:      prj.GiteeRepo,
		RefQuestionSet: prj.RefQuestionSet,
		Labels:         sqlx.JsonColumn[[]string]{Val: prj.Labels, Valid: true},
		Desc:           prj.Desc,
		Utime:          prj.Utime,
	}
}

func (repo *projectAdminRepository) queToEntity(d domain.Question) dao.ProjectQuestion {
	return dao.ProjectQuestion{
		Id:       d.Id,
		Title:    d.Title,
		Analysis: d.Analysis,
		Status:   d.Status.ToUint8(),
		Answer:   d.Answer,
		Utime:    d.Utime.UnixMilli(),
	}
}

func NewProjectAdminRepository(dao dao.ProjectAdminDAO) ProjectAdminRepository {
	return &projectAdminRepository{dao: dao}
}

func (repo *projectAdminRepository) prjToDomain(prj dao.Project,
	resumes []dao.ProjectResume,
	diff []dao.ProjectDifficulty,
	ques []dao.ProjectQuestion,
	intrs []dao.ProjectIntroduction,
	combos []dao.ProjectCombo,
) domain.Project {
	return domain.Project{
		Id:             prj.Id,
		SN:             prj.SN,
		Title:          prj.Title,
		Status:         domain.ProjectStatus(prj.Status),
		Labels:         prj.Labels.Val,
		Desc:           prj.Desc,
		Overview:       prj.Overview,
		SystemDesign:   prj.SystemDesign,
		GithubRepo:     prj.GithubRepo,
		GiteeRepo:      prj.GiteeRepo,
		RefQuestionSet: prj.RefQuestionSet,
		Utime:          prj.Utime,
		Resumes: slice.Map(resumes, func(idx int, src dao.ProjectResume) domain.Resume {
			return repo.rsmToDomain(src)
		}),
		Difficulties: slice.Map(diff, func(idx int, src dao.ProjectDifficulty) domain.Difficulty {
			return repo.diffToDomain(src)
		}),
		Questions: slice.Map(ques, func(idx int, src dao.ProjectQuestion) domain.Question {
			return repo.queToDomain(src)
		}),
		Introductions: slice.Map(intrs, func(idx int, src dao.ProjectIntroduction) domain.Introduction {
			return repo.intrToDomain(src)
		}),
		Combos: slice.Map(combos, func(idx int, src dao.ProjectCombo) domain.Combo {
			return repo.comboToDomain(src)
		}),
	}
}

func (repo *projectAdminRepository) intrToDomain(r dao.ProjectIntroduction) domain.Introduction {
	return domain.Introduction{
		Id:       r.Id,
		Role:     r.Role,
		Content:  r.Content,
		Analysis: r.Analysis,
		Status:   domain.IntroductionStatus(r.Status),
		Utime:    time.UnixMilli(r.Utime),
	}
}

func (repo *projectAdminRepository) rsmToDomain(r dao.ProjectResume) domain.Resume {
	return domain.Resume{
		Id:       r.Id,
		Role:     r.Role,
		Content:  r.Content,
		Analysis: r.Analysis,
		Status:   domain.ResumeStatus(r.Status),
		Utime:    time.UnixMilli(r.Utime),
	}
}

func (repo *projectAdminRepository) diffToDomain(d dao.ProjectDifficulty) domain.Difficulty {
	return domain.Difficulty{
		Id:       d.Id,
		Title:    d.Title,
		Content:  d.Content,
		Analysis: d.Analysis,
		Status:   domain.DifficultyStatus(d.Status),
		Utime:    time.UnixMilli(d.Utime),
	}
}

func (repo *projectAdminRepository) queToDomain(d dao.ProjectQuestion) domain.Question {
	return domain.Question{
		Id:       d.Id,
		Title:    d.Title,
		Analysis: d.Analysis,
		Status:   domain.QuestionStatus(d.Status),
		Answer:   d.Answer,
		Utime:    time.UnixMilli(d.Utime),
	}
}

func (repo *projectAdminRepository) comboToDomain(c dao.ProjectCombo) domain.Combo {
	return domain.Combo{
		Id:      c.Id,
		Title:   c.Title,
		Content: c.Content,
		Utime:   c.Utime,
		Status:  domain.ComboStatus(c.Status),
	}
}

func (repo *projectAdminRepository) comboToEntity(c domain.Combo) dao.ProjectCombo {
	return dao.ProjectCombo{
		Id:      c.Id,
		Title:   c.Title,
		Content: c.Content,
		Utime:   c.Utime,
		Status:  c.Status.ToUint8(),
	}
}
