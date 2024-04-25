package service

import (
	"context"
	"errors"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/ecodeclub/webook/internal/search/internal/repository"
	"golang.org/x/sync/errgroup"
)

type SearchSvc interface {
	Search(ctx context.Context, keywords []string) (domain.SearchResult, error)
	SearchWithBiz(ctx context.Context, biz string, keywords []string) (domain.SearchResult, error)
}

type searchSvc struct {
	questionRepo    repository.QuestionRepo
	questionSetRepo repository.QuestionSetRepo
	skillRepo       repository.SkillRepo
	caseRepo        repository.CaseRepo
	searchHandlers  map[string]SearchHandler
}

func NewSearchSvc(
	questionRepo repository.QuestionRepo,
	questionSetRepo repository.QuestionSetRepo,
	skillRepo repository.SkillRepo,
	caseRepo repository.CaseRepo,
) SearchSvc {
	searchHandlers := map[string]SearchHandler{
		"skill":       NewSkillHandler(caseRepo, questionRepo, skillRepo),
		"case":        NewCaseHandler(caseRepo),
		"questionSet": NewQuestionSetHandler(questionRepo, questionSetRepo),
		"question":    NewQuestionHandler(questionRepo),
	}
	return &searchSvc{
		questionRepo:    questionRepo,
		questionSetRepo: questionSetRepo,
		skillRepo:       skillRepo,
		caseRepo:        caseRepo,
		searchHandlers:  searchHandlers,
	}
}

func (s *searchSvc) Search(ctx context.Context, keywords []string) (domain.SearchResult, error) {
	var eg errgroup.Group
	var cases []domain.Case
	var ques []domain.Question
	eg.Go(func() error {
		var err error
		cases, err = s.caseRepo.SearchCase(ctx, keywords)
		return err
	})
	eg.Go(func() error {
		var err error
		ques, err = s.questionRepo.SearchQuestion(ctx, keywords)
		return err
	})
	if err := eg.Wait(); err != nil {
		return domain.SearchResult{}, err
	}
	caseIds := slice.Map(cases, func(idx int, src domain.Case) int64 {
		return src.Id
	})
	questionIds := slice.Map(ques, func(idx int, src domain.Question) int64 {
		return src.ID
	})
	var questionSets []domain.QuestionSet
	var skills []domain.Skill
	eg.Go(func() error {
		var err error
		questionSets, err = s.questionSetRepo.SearchQuestionSet(ctx, questionIds, keywords)
		return err
	})

	eg.Go(func() error {
		var err error
		skills, err = s.skillRepo.SearchSkill(ctx, questionIds, caseIds, keywords)
		return err
	})
	if err := eg.Wait(); err != nil {
		return domain.SearchResult{}, err
	}
	return domain.SearchResult{
		Cases:       cases,
		Questions:   ques,
		QuestionSet: questionSets,
		Skills:      skills,
	}, nil

}

func (s *searchSvc) SearchWithBiz(ctx context.Context, biz string, keywords []string) (domain.SearchResult, error) {
	handler, ok := s.searchHandlers[biz]
	if !ok {
		return domain.SearchResult{}, errors.New("未找到相关业务的搜索方法")
	}
	return handler.search(ctx, keywords)
}
