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
	"fmt"
	"time"

	"github.com/ecodeclub/webook/internal/marketing/internal/domain"
	"github.com/ecodeclub/webook/internal/marketing/internal/event"
	"github.com/ecodeclub/webook/internal/marketing/internal/event/producer"
	"github.com/ecodeclub/webook/internal/marketing/internal/repository"
)

type ActivityExecutor struct {
	repo                repository.MarketingRepository
	memberEventProducer producer.MemberEventProducer
	creditEventProducer producer.CreditEventProducer
}

func NewActivityExecutor(
	repo repository.MarketingRepository,
	memberEventProducer producer.MemberEventProducer,
	creditEventProducer producer.CreditEventProducer,
) *ActivityExecutor {
	return &ActivityExecutor{
		repo:                repo,
		memberEventProducer: memberEventProducer,
		creditEventProducer: creditEventProducer,
	}
}

func (s *ActivityExecutor) Execute(ctx context.Context, act domain.UserRegistrationActivity) error {
	if err := s.awardRegistrationBonus(ctx, act); err != nil {
		return err
	}
	return s.awardInvitationBonus(ctx, act)
}

func (s *ActivityExecutor) awardRegistrationBonus(ctx context.Context, act domain.UserRegistrationActivity) error {
	endAtDate := time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC)
	if endAtDate.Before(time.Now()) {
		return nil
	}
	err := s.memberEventProducer.Produce(ctx, event.MemberEvent{
		Key:    fmt.Sprintf("user-registration-%d", act.Uid),
		Uid:    act.Uid,
		Days:   uint64(time.Until(endAtDate) / (24 * time.Hour)),
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
	if err != nil {
		return fmt.Errorf("查找邀请码失败: %w", err)
	}

	credits := uint64(300)
	_, err = s.repo.CreateInvitationRecord(ctx, domain.InvitationRecord{
		InviterId: c.Uid,
		InviteeId: act.Uid,
		Code:      c.Code,
		Attrs:     domain.InvitationRecordAttrs{Credits: credits},
	})
	if err != nil {
		return fmt.Errorf("创建邀请记录失败: %w", err)
	}

	return s.creditEventProducer.Produce(ctx, event.CreditIncreaseEvent{
		Key:    fmt.Sprintf("inviteeId-%d", act.Uid),
		Uid:    c.Uid,
		Amount: credits,
		Biz:    "user",
		BizId:  act.Uid,
		Action: "邀请奖励",
	})
}
