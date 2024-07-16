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

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
)

type ConfigRepository interface {
	GetConfig(ctx context.Context, biz string) (domain.BizConfig, error)
}

// CachedConfigRepository 这个是一定要搞缓存的
// 后续性能瓶颈了再说
type CachedConfigRepository struct {
	dao dao.ConfigDAO
}

func NewCachedConfigRepository(dao dao.ConfigDAO) ConfigRepository {
	return &CachedConfigRepository{dao: dao}
}

func (repo *CachedConfigRepository) GetConfig(ctx context.Context, biz string) (domain.BizConfig, error) {
	res, err := repo.dao.GetConfig(ctx, biz)
	if err != nil {
		return domain.BizConfig{}, err
	}
	return domain.BizConfig{
		MaxInput:       res.MaxInput,
		PromptTemplate: res.PromptTemplate,
		KnowledgeId:    res.KnowledgeId,
	}, nil
}
