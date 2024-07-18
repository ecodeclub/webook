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

package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/ecodeclub/webook/internal/marketing/internal/domain"
	"github.com/ecodeclub/webook/internal/marketing/internal/event"
	"github.com/ecodeclub/webook/internal/marketing/internal/event/producer"
	"github.com/ecodeclub/webook/internal/marketing/internal/repository"
	"github.com/gotomicro/ego/core/elog"
)

type ActivityExecutor struct {
	repo                repository.MarketingRepository
	memberEventProducer producer.MemberEventProducer
	creditEventProducer producer.CreditEventProducer
	logger              *elog.Component
	creditsAwarded      uint64
}

func NewActivityExecutor(
	repo repository.MarketingRepository,
	memberEventProducer producer.MemberEventProducer,
	creditEventProducer producer.CreditEventProducer,
	creditsAwarded uint64,
) *ActivityExecutor {
	return &ActivityExecutor{
		repo:                repo,
		memberEventProducer: memberEventProducer,
		creditEventProducer: creditEventProducer,
		logger:              elog.DefaultLogger,
		creditsAwarded:      creditsAwarded,
	}
}

func (s *ActivityExecutor) Execute(ctx context.Context, act domain.UserRegistrationActivity) error {
	if err := s.awardRegistrationBonus(ctx, act); err != nil {
		return err
	}
	return s.awardInvitationBonus(ctx, act)
}

func (s *ActivityExecutor) awardRegistrationBonus(ctx context.Context, act domain.UserRegistrationActivity) error {
	err := s.memberEventProducer.Produce(ctx, event.MemberEvent{
		Key:    fmt.Sprintf("user-registration-%d", act.Uid),
		Uid:    act.Uid,
		Days:   7,
		Biz:    "user",
		BizId:  act.Uid,
		Action: "注册福利",
	})
	if err != nil {
		return fmt.Errorf("为注册者发放注册福利失败: %w", err)
	}
	return nil
}

func (s *ActivityExecutor) awardInvitationBonus(ctx context.Context, act domain.UserRegistrationActivity) error {
	if act.InvitationCode == "" {
		return nil
	}

	c, err := s.repo.FindInvitationCodeByCode(ctx, act.InvitationCode)
	if errors.Is(err, repository.ErrInvitationCodeNotFound) {
		s.logger.Warn("未找到邀请码", elog.String("invitationCode", act.InvitationCode))
		return nil
	}
	if err != nil {
		return fmt.Errorf("查找邀请码失败: %w", err)
	}

	_, err = s.repo.CreateInvitationRecord(ctx, domain.InvitationRecord{
		InviterId: c.Uid,
		InviteeId: act.Uid,
		Code:      c.Code,
		Attrs:     domain.InvitationRecordAttrs{Credits: s.creditsAwarded},
	})
	if err != nil {
		return fmt.Errorf("创建邀请记录失败: %w", err)
	}

	return s.creditEventProducer.Produce(ctx, event.CreditIncreaseEvent{
		Key:    fmt.Sprintf("inviteeId-%d", act.Uid),
		Uid:    c.Uid,
		Amount: s.creditsAwarded,
		Biz:    "user",
		BizId:  act.Uid,
		Action: "邀请奖励",
	})
}
