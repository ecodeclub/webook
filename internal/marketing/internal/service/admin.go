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

	"github.com/ecodeclub/webook/internal/marketing/internal/domain"
	"github.com/ecodeclub/webook/internal/marketing/internal/repository"
)

type RedemptionCodeAdminService interface {
	GenerateRedemptionCodes(ctx context.Context, codes []domain.RedemptionCode) error
}

type adminService struct {
	repo repository.MarketingRepository
}

func NewAdminService(repo repository.MarketingRepository) RedemptionCodeAdminService {
	return &adminService{repo: repo}
}

func (a *adminService) GenerateRedemptionCodes(ctx context.Context, codes []domain.RedemptionCode) error {
	_, err := a.repo.CreateRedemptionCodes(ctx, codes)
	return err
}
