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

	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/ecodeclub/webook/internal/search/internal/repository"
)

type SearchHandler interface {
	// 不加锁 res
	search(ctx context.Context, keywords string, offset, limit int, res *domain.SearchResult) error
}

type caseHandler struct {
	caseRepo repository.CaseRepo
}

func (c *caseHandler) search(ctx context.Context, keywords string, offset, limit int, res *domain.SearchResult) error {
	cases, err := c.caseRepo.SearchCase(ctx, offset, limit, keywords)
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

func (q *questionHandler) search(ctx context.Context, keywords string, offset, limit int, res *domain.SearchResult) error {
	ques, err := q.questionRepo.SearchQuestion(ctx, offset, limit, keywords)
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

func (q *questionSetHandler) search(ctx context.Context, keywords string, offset, limit int, res *domain.SearchResult) error {
	questionSets, err := q.questionSetRepo.SearchQuestionSet(ctx, offset, limit, keywords)
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
func (s *skillHandler) search(ctx context.Context, keywords string, offset, limit int, res *domain.SearchResult) error {
	skills, err := s.skillRepo.SearchSkill(ctx, offset, limit, keywords)
	if err != nil {
		return err
	}
	res.SetSkills(skills)
	return nil
}
