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

	"github.com/ecodeclub/webook/internal/member/internal/domain"
	"github.com/ecodeclub/webook/internal/member/internal/repository"
)

var (
	ErrUpdateMemberFailed     = repository.ErrUpdateMemberFailed
	ErrDuplicatedMemberRecord = repository.ErrDuplicatedMemberRecord
)

//go:generate mockgen -source=./service.go -package=membermocks --destination=../../mocks/member.mock.go -typed Service
type Service interface {
	GetMembershipInfo(ctx context.Context, uid int64) (domain.Member, error)
	ActivateMembership(ctx context.Context, member domain.Member) error
}

type service struct {
	repo repository.MemberRepository
}

func NewMemberService(repo repository.MemberRepository) Service {
	return &service{repo: repo}
}

func (s *service) GetMembershipInfo(ctx context.Context, uid int64) (domain.Member, error) {
	return s.repo.FindByUID(ctx, uid)
}

func (s *service) ActivateMembership(ctx context.Context, member domain.Member) error {
	return s.repo.Upsert(ctx, member)
}
