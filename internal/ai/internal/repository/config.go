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

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
)

type ConfigRepository interface {
	GetConfig(ctx context.Context, biz string) (domain.BizConfig, error)
	Save(ctx context.Context, cfg domain.BizConfig) (int64, error)
	List(ctx context.Context) ([]domain.BizConfig, error)
	GetById(ctx context.Context, id int64) (domain.BizConfig, error)
}

// CachedConfigRepository 这个是一定要搞缓存的
// 后续性能瓶颈了再说
type CachedConfigRepository struct {
	dao dao.ConfigDAO
}

func NewCachedConfigRepository(dao dao.ConfigDAO) ConfigRepository {
	return &CachedConfigRepository{dao: dao}
}

// Save 保存配置
func (r *CachedConfigRepository) Save(ctx context.Context, cfg domain.BizConfig) (int64, error) {
	return r.dao.Save(ctx, dao.BizConfig{
		Id:             cfg.Id,
		Biz:            cfg.Biz,
		MaxInput:       cfg.MaxInput,
		Model:          cfg.Model,
		Price:          cfg.Price,
		Temperature:    cfg.Temperature,
		TopP:           cfg.TopP,
		SystemPrompt:   cfg.SystemPrompt,
		PromptTemplate: cfg.PromptTemplate,
		KnowledgeId:    cfg.KnowledgeId,
	})
}
func (r *CachedConfigRepository) List(ctx context.Context) ([]domain.BizConfig, error) {
	configs, err := r.dao.List(ctx)
	if err != nil {
		return nil, err
	}
	return slice.Map(configs, func(idx int, src dao.BizConfig) domain.BizConfig {
		return r.toDomain(src)
	}), nil
}

func (r *CachedConfigRepository) GetById(ctx context.Context, id int64) (domain.BizConfig, error) {
	cfg, err := r.dao.GetById(ctx, id)
	if err != nil {
		return domain.BizConfig{}, err
	}

	return r.toDomain(cfg), nil
}
func (repo *CachedConfigRepository) GetConfig(ctx context.Context, biz string) (domain.BizConfig, error) {
	res, err := repo.dao.GetConfig(ctx, biz)
	if err != nil {
		return domain.BizConfig{}, err
	}
	return repo.toDomain(res), nil
}

func (repo *CachedConfigRepository) toDomain(src dao.BizConfig) domain.BizConfig {
	return domain.BizConfig{
		Id:             src.Id,
		Biz:            src.Biz,
		Model:          src.Model,
		Price:          src.Price,
		Temperature:    src.Temperature,
		TopP:           src.TopP,
		SystemPrompt:   src.SystemPrompt,
		MaxInput:       src.MaxInput,
		KnowledgeId:    src.KnowledgeId,
		PromptTemplate: src.PromptTemplate,
		Utime:          src.Utime,
	}
}
