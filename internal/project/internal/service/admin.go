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
	"time"

	"github.com/lithammer/shortuuid/v4"

	"github.com/ecodeclub/webook/internal/project/internal/event"
	"github.com/gotomicro/ego/core/elog"

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
	adminRepo repository.ProjectAdminRepository
	repo      repository.Repository
	producer  event.SyncProjectToSearchEventProducer
	logger    *elog.Component
}

func (svc *projectAdminService) ResumePublish(ctx context.Context, pid int64, resume domain.Resume) (int64, error) {
	resume.Status = domain.ResumeStatusPublished
	id, err := svc.adminRepo.ResumePublish(ctx, pid, resume)
	if err == nil {
		// 同步数据
		svc.syncToSearch(pid)
	}
	return id, err
}

func (svc *projectAdminService) QuestionPublish(ctx context.Context, pid int64, que domain.Question) (int64, error) {
	que.Status = domain.QuestionStatusPublished
	id, err := svc.adminRepo.QuestionSync(ctx, pid, que)
	if err == nil {
		// 同步数据
		svc.syncToSearch(pid)
	}
	return id, err
}

func (svc *projectAdminService) DifficultyPublish(ctx context.Context, pid int64, diff domain.Difficulty) (int64, error) {
	diff.Status = domain.DifficultyStatusPublished
	id, err := svc.adminRepo.DifficultyPublish(ctx, pid, diff)
	if err == nil {
		// 同步数据
		svc.syncToSearch(pid)
	}
	return id, err
}

func (svc *projectAdminService) IntroductionPublish(ctx context.Context, pid int64, intr domain.Introduction) (int64, error) {
	intr.Status = domain.IntroductionStatusPublished
	id, err := svc.adminRepo.IntroductionSync(ctx, pid, intr)
	if err == nil {
		// 同步数据
		svc.syncToSearch(pid)
	}
	return id, err
}

func (svc *projectAdminService) IntroductionDetail(ctx context.Context, id int64) (domain.Introduction, error) {
	return svc.adminRepo.IntroductionDetail(ctx, id)
}

func (svc *projectAdminService) IntroductionSave(ctx context.Context, pid int64, intr domain.Introduction) (int64, error) {
	intr.Status = domain.IntroductionStatusUnpublished
	return svc.adminRepo.IntroductionSave(ctx, pid, intr)
}

func (svc *projectAdminService) Publish(ctx context.Context, prj domain.Project) (int64, error) {
	prj.Status = domain.ProjectStatusPublished
	if prj.Id == 0 {
		sn := shortuuid.New()
		prj.SN = sn
	}
	id, err := svc.adminRepo.Sync(ctx, prj)
	if err == nil {
		// 同步数据，这边后续读写分离之后，可能会有问题
		svc.syncToSearch(id)
	}
	return id, err
}

func (svc *projectAdminService) QuestionDetail(ctx context.Context, id int64) (domain.Question, error) {
	return svc.adminRepo.QuestionDetail(ctx, id)
}

func (svc *projectAdminService) QuestionSave(ctx context.Context, pid int64, que domain.Question) (int64, error) {
	que.Status = domain.QuestionStatusUnpublished
	return svc.adminRepo.QuestionSave(ctx, pid, que)
}

func (svc *projectAdminService) DifficultyDetail(ctx context.Context, id int64) (domain.Difficulty, error) {
	return svc.adminRepo.DifficultyDetail(ctx, id)
}

func (svc *projectAdminService) DifficultySave(ctx context.Context, pid int64, diff domain.Difficulty) (int64, error) {
	diff.Status = domain.DifficultyStatusUnpublished
	return svc.adminRepo.DifficultySave(ctx, pid, diff)
}

func (svc *projectAdminService) ResumeDetail(ctx context.Context, id int64) (domain.Resume, error) {
	return svc.adminRepo.ResumeDetail(ctx, id)
}

func (svc *projectAdminService) ResumeSave(ctx context.Context, pid int64, resume domain.Resume) (int64, error) {
	resume.Status = domain.ResumeStatusUnpublished
	return svc.adminRepo.ResumeSave(ctx, pid, resume)
}

func (svc *projectAdminService) Detail(ctx context.Context, id int64) (domain.Project, error) {
	return svc.adminRepo.Detail(ctx, id)
}

func (svc *projectAdminService) Count(ctx context.Context) (int64, error) {
	return svc.adminRepo.Count(ctx)
}

func (svc *projectAdminService) List(ctx context.Context, offset int, limit int) ([]domain.Project, error) {
	return svc.adminRepo.List(ctx, offset, limit)
}

func (svc *projectAdminService) Save(ctx context.Context,
	prj domain.Project) (int64, error) {
	sn := shortuuid.New()
	prj.SN = sn
	prj.Status = domain.ProjectStatusUnpublished
	return svc.adminRepo.Save(ctx, prj)
}

func (svc *projectAdminService) syncToSearch(id int64) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	prj, err := svc.repo.Detail(ctx, id)
	if err != nil {
		svc.logger.Error("准备同步数据，查询项目详情失败",
			elog.Int64("id", id),
			elog.FieldErr(err))
		return
	}
	evt := event.NewSyncProjectToSearchEvent(prj)
	err = svc.producer.Produce(ctx, evt)
	if err != nil {
		svc.logger.Error("同步数据到搜索失败",
			elog.Int64("id", id),
			elog.FieldErr(err))
	}
}

func NewProjectAdminService(
	adminRepo repository.ProjectAdminRepository,
	producer event.SyncProjectToSearchEventProducer,
	repo repository.Repository) ProjectAdminService {
	return &projectAdminService{
		adminRepo: adminRepo,
		producer:  producer,
		repo:      repo,
		logger:    elog.DefaultLogger,
	}
}
