package service

import (
	"context"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/ecodeclub/webook/internal/search/internal/repository"
	"golang.org/x/sync/errgroup"
)

type SearchHandler interface {
	search(ctx context.Context, keywords []string) (domain.SearchResult, error)
}

type caseHandler struct {
	caseRepo repository.CaseRepo
}

func (c *caseHandler) search(ctx context.Context, keywords []string) (domain.SearchResult, error) {
	cases, err := c.caseRepo.SearchCase(ctx, keywords)
	if err != nil {
		return domain.SearchResult{}, err
	}
	return domain.SearchResult{
		Cases: cases,
	}, nil
}

func NewCaseHandler(caseRepo repository.CaseRepo) SearchHandler {
	return &caseHandler{
		caseRepo: caseRepo,
	}
}

type questionHandler struct {
	questionRepo repository.QuestionRepo
}

func (q *questionHandler) search(ctx context.Context, keywords []string) (domain.SearchResult, error) {
	ques, err := q.questionRepo.SearchQuestion(ctx, keywords)
	if err != nil {
		return domain.SearchResult{}, err
	}
	return domain.SearchResult{
		Questions: ques,
	}, nil
}

func NewQuestionHandler(questionRepo repository.QuestionRepo) SearchHandler {
	return &questionHandler{
		questionRepo: questionRepo,
	}
}

type questionSetHandler struct {
	questionRepo    repository.QuestionRepo
	questionSetRepo repository.QuestionSetRepo
}

func (q *questionSetHandler) search(ctx context.Context, keywords []string) (domain.SearchResult, error) {
	ques, err := q.questionRepo.SearchQuestion(ctx, keywords)
	if err != nil {
		return domain.SearchResult{}, err
	}
	qids := slice.Map(ques, func(idx int, src domain.Question) int64 {
		return src.ID
	})
	questionSets, err := q.questionSetRepo.SearchQuestionSet(ctx, qids, keywords)
	return domain.SearchResult{
		QuestionSet: questionSets,
	}, err
}

func NewQuestionSetHandler(questionRepo repository.QuestionRepo, questionSetRepo repository.QuestionSetRepo) SearchHandler {
	return &questionSetHandler{
		questionRepo:    questionRepo,
		questionSetRepo: questionSetRepo,
	}
}

type skillHandler struct {
	caseRepo     repository.CaseRepo
	questionRepo repository.QuestionRepo
	skillRepo    repository.SkillRepo
}

func NewSkillHandler(
	caseRepo repository.CaseRepo,
	questionRepo repository.QuestionRepo,
	skillRepo repository.SkillRepo) SearchHandler {
	return &skillHandler{
		caseRepo:     caseRepo,
		questionRepo: questionRepo,
		skillRepo:    skillRepo,
	}
}
func (s *skillHandler) search(ctx context.Context, keywords []string) (domain.SearchResult, error) {
	var eg errgroup.Group
	var cases, questions []int64
	eg.Go(func() error {
		cas, err := s.caseRepo.SearchCase(ctx, keywords)
		if err != nil {
			return err
		}
		cases = slice.Map(cas, func(idx int, src domain.Case) int64 {
			return src.Id
		})
		return nil
	})
	eg.Go(func() error {
		ques, err := s.questionRepo.SearchQuestion(ctx, keywords)
		if err != nil {
			return err
		}
		questions = slice.Map(ques, func(idx int, src domain.Question) int64 {
			return src.ID
		})
		return nil
	})
	if err := eg.Wait(); err != nil {
		return domain.SearchResult{}, err
	}
	skills, err := s.skillRepo.SearchSkill(ctx, questions, cases, keywords)
	return domain.SearchResult{
		Skills: skills,
	}, err
}
