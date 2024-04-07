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
	"errors"

	"github.com/ecodeclub/webook/internal/user/internal/event"
	"github.com/lithammer/shortuuid/v4"

	"github.com/ecodeclub/webook/internal/user/internal/domain"
	"github.com/ecodeclub/webook/internal/user/internal/repository"
	"github.com/gotomicro/ego/core/elog"
)

//go:generate mockgen -source=./user.go -package=svcmocks -destination=mocks/user.mock.go UserService
type UserService interface {
	Profile(ctx context.Context, id int64) (domain.User, error)
	// FindOrCreateByWechat 查找或者初始化
	// 随着业务增长，这边可以考虑拆分出去作为一个新的 Service
	FindOrCreateByWechat(ctx context.Context, info domain.WechatInfo) (domain.User, error)

	// UpdateNonSensitiveInfo 更新非敏感数据
	// 你可以在这里进一步补充究竟哪些数据会被更新
	UpdateNonSensitiveInfo(ctx context.Context, user domain.User) error
}

type userService struct {
	repo     repository.UserRepository
	producer *event.RegistrationEventProducer
	logger   *elog.Component
}

func NewUserService(repo repository.UserRepository, p *event.RegistrationEventProducer) UserService {
	return &userService{
		repo:     repo,
		producer: p,
		logger:   elog.DefaultLogger,
	}
}

func (svc *userService) UpdateNonSensitiveInfo(ctx context.Context, user domain.User) error {
	// 不让修改序列号
	user.SN = ""
	return svc.repo.Update(ctx, user)
}

func (svc *userService) FindOrCreateByWechat(ctx context.Context,
	info domain.WechatInfo) (domain.User, error) {
	// 类似于手机号的过程，大部分人只是扫码登录，也就是数据在我们这里是有的
	u, err := svc.repo.FindByWechat(ctx, info.OpenId)
	if !errors.Is(err, repository.ErrUserNotFound) {
		return u, err
	}
	sn := shortuuid.New()
	id, err := svc.repo.Create(ctx, domain.User{
		WechatInfo: info,
		SN:         sn,
		Nickname:   sn[:4],
	})

	if err != nil {
		return domain.User{}, err
	}

	// 发送注册成功消息
	evt := event.RegistrationEvent{Uid: id}
	if e := svc.producer.Produce(ctx, evt); e != nil {
		svc.logger.Error("发送注册成功消息失败",
			elog.FieldErr(e),
			elog.FieldKey("event"),
			elog.FieldValueAny(evt),
		)
	}

	return domain.User{
		Id:         id,
		WechatInfo: info,
	}, nil
}

func (svc *userService) Profile(ctx context.Context,
	id int64) (domain.User, error) {
	// 在系统内部，基本上都是用 ID 的。
	// 有些人的系统比较复杂，有一个 GUID（global unique ID）
	return svc.repo.FindById(ctx, id)
}
