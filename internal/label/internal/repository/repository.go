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
	"github.com/ecodeclub/webook/internal/label/internal/domain"
	"github.com/ecodeclub/webook/internal/label/internal/repository/dao"
)

type LabelRepository interface {
	UidLabels(ctx context.Context, uid int64) ([]domain.Label, error)
	CreateLabel(ctx context.Context, uid int64, name string) (int64, error)
}

type CachedLabelRepository struct {
	dao dao.LabelDAO
}

func (repo *CachedLabelRepository) CreateLabel(ctx context.Context, uid int64, name string) (int64, error) {
	return repo.dao.CreateLabel(ctx, dao.Label{
		Uid:  uid,
		Name: name,
	})
}

func (repo *CachedLabelRepository) UidLabels(ctx context.Context, uid int64) ([]domain.Label, error) {
	labels, err := repo.dao.UidLabels(ctx, uid)
	return slice.Map(labels, func(idx int, src dao.Label) domain.Label {
		return domain.Label{
			Id:   src.Id,
			Uid:  src.Uid,
			Name: src.Name,
		}
	}), err
}

func NewCachedLabelRepository(dao dao.LabelDAO) LabelRepository {
	return &CachedLabelRepository{dao: dao}
}
