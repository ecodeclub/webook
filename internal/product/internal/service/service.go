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

	"github.com/ecodeclub/webook/internal/product/internal/domain"
	"github.com/ecodeclub/webook/internal/product/internal/repository"
)

//go:generate mockgen -source=./service.go -package=productmocks -destination=../../mocks/product.mock.go -typed Service
type Service interface {
	FindSKUBySN(ctx context.Context, sn string) (domain.SPU, error)
}

func NewService(repo repository.ProductRepository) Service {
	return &service{repo: repo}
}

type service struct {
	repo repository.ProductRepository
}

func (s *service) FindSKUBySN(ctx context.Context, sn string) (domain.SPU, error) {
	return s.repo.FindSKUBySN(ctx, sn)
}
