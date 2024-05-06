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

package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/project/internal/domain"
	"github.com/ecodeclub/webook/internal/project/internal/repository"
)

type ProjectAdminService interface {
	Save(ctx context.Context, prj domain.Project) (int64, error)
	Publish(ctx context.Context, prj domain.Project) (int64, error)

	List(ctx context.Context, offset int, limit int) ([]domain.Project, error)
	Count(ctx context.Context) (int64, error)
	Detail(ctx context.Context, id int64) (domain.Project, error)

	ResumeSave(ctx context.Context, pid int64, resume domain.Resume) (int64, error)
	ResumeDetail(ctx context.Context, id int64) (domain.Resume, error)
	ResumePublish(ctx context.Context, pid int64, resume domain.Resume) (int64, error)

	DifficultySave(ctx context.Context, pid int64, diff domain.Difficulty) (int64, error)
	DifficultyDetail(ctx context.Context, id int64) (domain.Difficulty, error)
	DifficultyPublish(ctx context.Context, pid int64, diff domain.Difficulty) (int64, error)

	QuestionSave(ctx context.Context, pid int64, que domain.Question) (int64, error)
	QuestionDetail(ctx context.Context, id int64) (domain.Question, error)
	QuestionPublish(ctx context.Context, pid int64, que domain.Question) (int64, error)

	IntroductionSave(ctx context.Context, pid int64, intr domain.Introduction) (int64, error)
	IntroductionDetail(ctx context.Context, id int64) (domain.Introduction, error)
	IntroductionPublish(ctx context.Context, pid int64, intr domain.Introduction) (int64, error)
}

type projectAdminService struct {
	repo repository.ProjectAdminRepository
}

func (svc *projectAdminService) ResumePublish(ctx context.Context, pid int64, resume domain.Resume) (int64, error) {
	resume.Status = domain.ResumeStatusPublished
	return svc.repo.ResumePublish(ctx, pid, resume)
}

func (svc *projectAdminService) QuestionPublish(ctx context.Context, pid int64, que domain.Question) (int64, error) {
	que.Status = domain.QuestionStatusPublished
	return svc.repo.QuestionSync(ctx, pid, que)
}

func (svc *projectAdminService) DifficultyPublish(ctx context.Context, pid int64, diff domain.Difficulty) (int64, error) {
	diff.Status = domain.DifficultyStatusPublished
	return svc.repo.DifficultyPublish(ctx, pid, diff)
}

func (svc *projectAdminService) IntroductionPublish(ctx context.Context, pid int64, intr domain.Introduction) (int64, error) {
	intr.Status = domain.IntroductionStatusPublished
	return svc.repo.IntroductionSync(ctx, pid, intr)
}

func (svc *projectAdminService) IntroductionDetail(ctx context.Context, id int64) (domain.Introduction, error) {
	return svc.repo.IntroductionDetail(ctx, id)
}

func (svc *projectAdminService) IntroductionSave(ctx context.Context, pid int64, intr domain.Introduction) (int64, error) {
	intr.Status = domain.IntroductionStatusUnpublished
	return svc.repo.IntroductionSave(ctx, pid, intr)
}

func (svc *projectAdminService) Publish(ctx context.Context, prj domain.Project) (int64, error) {
	prj.Status = domain.ProjectStatusPublished
	return svc.repo.Sync(ctx, prj)
}

func (svc *projectAdminService) QuestionDetail(ctx context.Context, id int64) (domain.Question, error) {
	return svc.repo.QuestionDetail(ctx, id)
}

func (svc *projectAdminService) QuestionSave(ctx context.Context, pid int64, que domain.Question) (int64, error) {
	que.Status = domain.QuestionStatusUnpublished
	return svc.repo.QuestionSave(ctx, pid, que)
}

func (svc *projectAdminService) DifficultyDetail(ctx context.Context, id int64) (domain.Difficulty, error) {
	return svc.repo.DifficultyDetail(ctx, id)
}

func (svc *projectAdminService) DifficultySave(ctx context.Context, pid int64, diff domain.Difficulty) (int64, error) {
	diff.Status = domain.DifficultyStatusUnpublished
	return svc.repo.DifficultySave(ctx, pid, diff)
}

func (svc *projectAdminService) ResumeDetail(ctx context.Context, id int64) (domain.Resume, error) {
	return svc.repo.ResumeDetail(ctx, id)
}

func (svc *projectAdminService) ResumeSave(ctx context.Context, pid int64, resume domain.Resume) (int64, error) {
	resume.Status = domain.ResumeStatusUnpublished
	return svc.repo.ResumeSave(ctx, pid, resume)
}

func (svc *projectAdminService) Detail(ctx context.Context, id int64) (domain.Project, error) {
	return svc.repo.Detail(ctx, id)
}

func (svc *projectAdminService) Count(ctx context.Context) (int64, error) {
	return svc.repo.Count(ctx)
}

func (svc *projectAdminService) List(ctx context.Context, offset int, limit int) ([]domain.Project, error) {
	return svc.repo.List(ctx, offset, limit)
}

func (svc *projectAdminService) Save(ctx context.Context,
	prj domain.Project) (int64, error) {
	prj.Status = domain.ProjectStatusUnpublished
	return svc.repo.Save(ctx, prj)
}

func NewProjectAdminService(repo repository.ProjectAdminRepository) ProjectAdminService {
	return &projectAdminService{repo: repo}
}
