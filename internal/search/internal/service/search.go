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
	"errors"
	"strings"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/ecodeclub/webook/internal/search/internal/repository"
	"golang.org/x/sync/errgroup"
)

type SearchService interface {
	// Search 出于长远考虑，这里你用 expr 来代表搜索的表达式，后期我们会考虑支持类似 github 那种复杂的搜索表达式
	Search(ctx context.Context, offset, limit int, expr string) (*domain.SearchResult, error)
}

type searchSvc struct {
	searchHandlers map[string]SearchHandler
}

func (s *searchSvc) Search(ctx context.Context, offset, limit int, expr string) (*domain.SearchResult, error) {
	biz, keywords, err := s.parseExpr(expr)
	if err != nil {
		return nil, err
	}
	var eg errgroup.Group
	res := &domain.SearchResult{}
	if biz == "all" {
		for _, handler := range s.searchHandlers {
			bizHandler := handler
			eg.Go(func() error {
				return bizHandler.search(ctx, keywords, offset, limit, res)
			})
		}
		if err = eg.Wait(); err != nil {
			return nil, err
		}
	} else {
		bizhandler, ok := s.searchHandlers[biz]
		if !ok {
			return nil, errors.New("无相关的业务处理方式")
		}
		err = bizhandler.search(ctx, keywords, offset, limit, res)
		if err != nil {
			return nil, err
		}
	}
	return res, nil

}
func (s *searchSvc) parseExpr(expr string) (string, string, error) {
	searchParams := strings.SplitN(expr, ":", 3)
	if len(searchParams) == 3 {
		typ := searchParams[0]
		if typ != "biz" {
			return "", "", errors.New("参数错误")
		}
		biz := searchParams[1]
		keywords := searchParams[2]
		return biz, keywords, nil
	}
	return "", "", errors.New("参数错误")
}

func NewSearchSvc(
	questionRepo repository.QuestionRepo,
	questionSetRepo repository.QuestionSetRepo,
	skillRepo repository.SkillRepo,
	caseRepo repository.CaseRepo,
) SearchService {
	searchHandlers := map[string]SearchHandler{
		"skill":       NewSkillHandler(skillRepo),
		"case":        NewCaseHandler(caseRepo),
		"questionSet": NewQuestionSetHandler(questionSetRepo),
		"question":    NewQuestionHandler(questionRepo),
	}
	return &searchSvc{
		searchHandlers: searchHandlers,
	}
}
