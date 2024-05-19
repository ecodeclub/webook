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
)

type ActivityExecutor struct {
	memberEventProducer producer.MemberEventProducer
	creditEventProducer producer.CreditEventProducer
	endAtDate           time.Time
}

func NewActivityExecutor(
	memberEventProducer producer.MemberEventProducer,
	creditEventProducer producer.CreditEventProducer,
) *ActivityExecutor {
	return &ActivityExecutor{
		memberEventProducer: memberEventProducer,
		creditEventProducer: creditEventProducer,
		endAtDate:           time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC),
	}
}

func (s *ActivityExecutor) Execute(ctx context.Context, act domain.UserRegistrationActivity) error {
	return s.memberEventProducer.Produce(ctx, event.MemberEvent{
		Key:    fmt.Sprintf("user-registration-%d", act.Uid),
		Uid:    act.Uid,
		Days:   uint64(time.Until(s.endAtDate) / (24 * time.Hour)),
		Biz:    "user",
		BizId:  act.Uid,
		Action: "注册福利",
	})
}
