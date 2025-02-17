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

//go:generate mockgen -source=./user.go -package=usermocks -typed=true -destination=../../mocks/user.mock.go UserService
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
	producer event.RegistrationEventProducer
	logger   *elog.Component
}

func NewUserService(repo repository.UserRepository, p event.RegistrationEventProducer) UserService {
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
	u, err := svc.repo.FindByWechat(ctx, info.UnionId)
	defer func() {
		// 尝试更新 openid 或者 Mini open id
		if (info.MiniOpenId != "" && u.WechatInfo.MiniOpenId == "") ||
			(info.OpenId != "" && u.WechatInfo.OpenId == "") {
			u.WechatInfo.MiniOpenId = info.MiniOpenId
			u.WechatInfo.OpenId = info.OpenId
			err1 := svc.repo.Update(ctx, u)
			if err1 != nil {
				svc.logger.Error("尝试更新微信信息失败", elog.FieldErr(err1))
			}
		}
	}()
	if !errors.Is(err, repository.ErrUserNotFound) {
		return u, err
	}
	sn := shortuuid.New()
	u = domain.User{
		WechatInfo: info,
		SN:         sn,
		Nickname:   sn[:4],
	}
	id, err := svc.repo.Create(ctx, u)

	if err != nil {
		return domain.User{}, err
	}
	// 发送注册成功消息
	evt := event.RegistrationEvent{Uid: id, InvitationCode: info.InvitationCode}
	if e := svc.producer.Produce(ctx, evt); e != nil {
		svc.logger.Error("发送注册成功消息失败",
			elog.FieldErr(e),
			elog.FieldKey("event"),
			elog.FieldValueAny(evt),
		)
	}
	u.Id = id
	return u, nil
}

func (svc *userService) Profile(ctx context.Context,
	id int64) (domain.User, error) {
	// 在系统内部，基本上都是用 ID 的。
	// 有些人的系统比较复杂，有一个 GUID（global unique ID）
	return svc.repo.FindById(ctx, id)
}
