package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/ecodeclub/webook/internal/search/internal/repository"
)

type SearchHandler interface {
	// 不加锁 res
	search(ctx context.Context, keywords string, res *domain.SearchResult) error
}

type caseHandler struct {
	caseRepo repository.CaseRepo
}

func (c *caseHandler) search(ctx context.Context, keywords string, res *domain.SearchResult) error {
	cases, err := c.caseRepo.SearchCase(ctx, keywords)
	if err != nil {
		return err
	}
	res.SetCases(cases)
	return nil
}

func NewCaseHandler(caseRepo repository.CaseRepo) SearchHandler {
	return &caseHandler{
		caseRepo: caseRepo,
	}
}

type questionHandler struct {
	questionRepo repository.QuestionRepo
}

func (q *questionHandler) search(ctx context.Context, keywords string, res *domain.SearchResult) error {
	ques, err := q.questionRepo.SearchQuestion(ctx, keywords)
	if err != nil {
		return err
	}
	res.SetQuestions(ques)
	return nil
}

func NewQuestionHandler(questionRepo repository.QuestionRepo) SearchHandler {
	return &questionHandler{
		questionRepo: questionRepo,
	}
}

type questionSetHandler struct {
	questionSetRepo repository.QuestionSetRepo
}

func (q *questionSetHandler) search(ctx context.Context, keywords string, res *domain.SearchResult) error {
	questionSets, err := q.questionSetRepo.SearchQuestionSet(ctx, keywords)
	if err != nil {
		return err
	}
	res.SetQuestionSet(questionSets)
	return nil
}

func NewQuestionSetHandler(questionSetRepo repository.QuestionSetRepo) SearchHandler {
	return &questionSetHandler{
		questionSetRepo: questionSetRepo,
	}
}

type skillHandler struct {
	skillRepo repository.SkillRepo
}

func NewSkillHandler(
	skillRepo repository.SkillRepo) SearchHandler {
	return &skillHandler{
		skillRepo: skillRepo,
	}
}
func (s *skillHandler) search(ctx context.Context, keywords string, res *domain.SearchResult) error {
	skills, err := s.skillRepo.SearchSkill(ctx, keywords)
	if err != nil {
		return err
	}
	res.SetSkills(skills)
	return nil
}
