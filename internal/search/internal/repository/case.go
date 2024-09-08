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

	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/ecodeclub/webook/internal/search/internal/repository/dao"
)

type caseRepository struct {
	caseDao dao.CaseDAO
}

func NewCaseRepo(caseDao dao.CaseDAO) CaseRepo {
	return &caseRepository{
		caseDao: caseDao,
	}
}

func (c *caseRepository) SearchCase(ctx context.Context, offset, limit int, queryMetas []domain.QueryMeta) ([]domain.Case, error) {
	cases, err := c.caseDao.SearchCase(ctx, offset, limit, queryMetas)
	if err != nil {
		return nil, err
	}
	ans := make([]domain.Case, 0, len(cases))
	for _, ca := range cases {
		ans = append(ans, c.toDomain(ca))
	}
	return ans, err
}

func (*caseRepository) toDomain(p dao.Case) domain.Case {
	return domain.Case{
		Id:         p.Id,
		Uid:        p.Uid,
		Labels:     p.Labels,
		Title:      p.Title,
		Content:    p.Content,
		Keywords:   p.Keywords,
		GithubRepo: p.GithubRepo,
		GiteeRepo:  p.GiteeRepo,
		Shorthand:  p.Shorthand,
		Highlight:  p.Highlight,
		Guidance:   p.Guidance,
		Status:     domain.CaseStatus(p.Status),
		Ctime:      time.UnixMilli(p.Ctime),
		Utime:      time.UnixMilli(p.Utime),
	}
}
