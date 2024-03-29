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

	"github.com/ecodeclub/webook/internal/credit/internal/domain"
	"github.com/ecodeclub/webook/internal/credit/internal/repository"
)

type Service interface {
	GetByUID(ctx context.Context, uid int64) (domain.Credit, error)
}

func NewService(repo repository.CreditRepository) Service {
	return &service{repo: repo}
}

type service struct {
	repo repository.CreditRepository
}

func (s *service) GetByUID(ctx context.Context, uid int64) (domain.Credit, error) {
	return domain.Credit{}, nil
}
