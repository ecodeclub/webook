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
	"github.com/ecodeclub/webook/internal/marketing/internal/event/producer"
	"github.com/ecodeclub/webook/internal/marketing/internal/repository"
	orderexe "github.com/ecodeclub/webook/internal/marketing/internal/service/activity/order"
	"github.com/ecodeclub/webook/internal/marketing/internal/service/activity/user"
	"github.com/ecodeclub/webook/internal/order"
	"github.com/ecodeclub/webook/internal/product"
	"golang.org/x/sync/errgroup"
)

var (
	ErrRedemptionNotFound = repository.ErrRedemptionNotFound
	ErrRedemptionCodeUsed = repository.ErrRedemptionCodeUsed
)

type Service interface {
	ExecuteOrderCompletedActivity(ctx context.Context, act domain.OrderCompletedActivity) error
	ExecuteUserRegistrationActivity(ctx context.Context, act domain.UserRegistrationActivity) error
	RedeemRedemptionCode(ctx context.Context, uid int64, code string) error
	ListRedemptionCodes(ctx context.Context, uid int64, offset, list int) ([]domain.RedemptionCode, int64, error)
	GenerateInvitationCode(ctx context.Context, uid int64) (domain.InvitationCode, error)
}

type service struct {
	repo repository.MarketingRepository

	productSvc              product.Service
	eventKeyGenerator       func() string
	invitationCodeGenerator func(id int64) string
	orderActivityExecutor   *orderexe.ActivityExecutor
	userActivityExecutor    *user.ActivityExecutor
}

func NewService(
	repo repository.MarketingRepository,
	orderSvc order.Service,
	productSvc product.Service,
	codeGenerator func(id int64) string,
	eventKeyGenerator func() string,
	memberEventProducer producer.MemberEventProducer,
	creditEventProducer producer.CreditEventProducer,
	permissionEventProducer producer.PermissionEventProducer,
) Service {

	return &service{
		repo:                    repo,
		productSvc:              productSvc,
		eventKeyGenerator:       eventKeyGenerator,
		invitationCodeGenerator: codeGenerator,
		orderActivityExecutor:   orderexe.NewOrderActivityExecutor(repo, orderSvc, codeGenerator, memberEventProducer, creditEventProducer, permissionEventProducer),
		userActivityExecutor:    user.NewActivityExecutor(repo, memberEventProducer, creditEventProducer, 500),
	}
}

func (s *service) ExecuteOrderCompletedActivity(ctx context.Context, act domain.OrderCompletedActivity) error {
	return s.orderActivityExecutor.Execute(ctx, act)
}

func (s *service) ExecuteUserRegistrationActivity(ctx context.Context, act domain.UserRegistrationActivity) error {
	return s.userActivityExecutor.Execute(ctx, act)
}

func (s *service) RedeemRedemptionCode(ctx context.Context, uid int64, code string) error {
	r, err := s.repo.SetUnusedRedemptionCodeStatusUsed(ctx, uid, code)
	if err != nil {
		return err
	}
	return s.orderActivityExecutor.Redeem(ctx, uid, r)
}

func (s *service) ListRedemptionCodes(ctx context.Context, uid int64, offset, list int) ([]domain.RedemptionCode, int64, error) {
	var (
		eg    errgroup.Group
		codes []domain.RedemptionCode
		total int64
	)
	eg.Go(func() error {
		var err error
		codes, err = s.repo.FindRedemptionCodesByUID(ctx, uid, offset, list)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.TotalRedemptionCodes(ctx, uid)
		return err
	})

	return codes, total, eg.Wait()
}

func (s *service) GenerateInvitationCode(ctx context.Context, uid int64) (domain.InvitationCode, error) {
	c, err := s.repo.CreateInvitationCode(ctx, domain.InvitationCode{
		Uid:  uid,
		Code: s.invitationCodeGenerator(uid),
	})
	if err != nil {
		return domain.InvitationCode{}, err
	}
	return c, nil
}
