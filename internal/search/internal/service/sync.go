package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/ecodeclub/webook/internal/search/internal/repository"
)

type SyncSvc interface {
	InputSkill(ctx context.Context, skill domain.Skill) error
	InputCase(ctx context.Context, ca domain.Case) error
	InputQuestion(ctx context.Context, que domain.Question) error
	InputQuestionSet(ctx context.Context, queSet domain.QuestionSet) error
}
type syncSvc struct {
	questionRepo    repository.QuestionRepo
	questionSetRepo repository.QuestionSetRepo
	skillRepo       repository.SkillRepo
	caseRepo        repository.CaseRepo
}

func NewSyncSvc(
	questionRepo repository.QuestionRepo,
	questionSetRepo repository.QuestionSetRepo,
	skillRepo repository.SkillRepo,
	caseRepo repository.CaseRepo) SyncSvc {
	return &syncSvc{
		questionRepo:    questionRepo,
		questionSetRepo: questionSetRepo,
		skillRepo:       skillRepo,
		caseRepo:        caseRepo,
	}
}

func (s *syncSvc) InputSkill(ctx context.Context, skill domain.Skill) error {
	return s.skillRepo.InputSKill(ctx, skill)
}

func (s *syncSvc) InputCase(ctx context.Context, ca domain.Case) error {
	return s.caseRepo.InputCase(ctx, ca)
}

func (s *syncSvc) InputQuestion(ctx context.Context, que domain.Question) error {
	return s.questionRepo.InputQuestion(ctx, que)
}

func (s *syncSvc) InputQuestionSet(ctx context.Context, queSet domain.QuestionSet) error {
	return s.questionSetRepo.InputQuestionSet(ctx, queSet)
}
