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

package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/ecodeclub/ecache"
)

type InvitationCodeECache struct {
	ec         ecache.Cache
	expiration time.Duration
}

func NewInvitationCodeECache(ec ecache.Cache, expiration time.Duration) InvitationCodeCache {
	return &InvitationCodeECache{
		ec: &ecache.NamespaceCache{
			Namespace: "marketing:invitation-code:",
			C:         ec,
		},
		expiration: expiration,
	}
}

func (q *InvitationCodeECache) GetInvitationCode(ctx context.Context, uid int64) (string, error) {
	return q.ec.Get(ctx, q.codeKey(uid)).AsString()
}

func (q *InvitationCodeECache) SetInvitationCode(ctx context.Context, uid int64, code string) error {
	return q.ec.Set(ctx, q.codeKey(uid), code, q.expiration)
}

// 注意 Namespace 设置
func (q *InvitationCodeECache) codeKey(uid int64) string {
	return fmt.Sprintf("user:%d", uid)
}
