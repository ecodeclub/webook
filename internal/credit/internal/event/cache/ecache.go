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
	"time"

	"github.com/ecodeclub/ecache"
)

type creditECache struct {
	ec ecache.Cache
}

func NewCreditECache(ec ecache.Cache) CreditCache {
	return &creditECache{
		ec: &ecache.NamespaceCache{
			Namespace: "credit:",
			C:         ec,
		},
	}
}

func (q *creditECache) GetEventKey(ctx context.Context, key string) (string, error) {
	return q.ec.Get(ctx, q.eventKey(key)).AsString()
}

func (q *creditECache) SetNXEventKey(ctx context.Context, key string) (bool, error) {
	return q.ec.SetNX(ctx, q.eventKey(key), 1, 24*time.Hour)
}

func (q *creditECache) DelEventKey(ctx context.Context, key string) (int64, error) {
	return q.ec.Delete(ctx, q.eventKey(key))
}

// 注意 Namespace 设置
func (q *creditECache) eventKey(key string) string {
	return "increase:" + key
}
