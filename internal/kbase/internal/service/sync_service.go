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
	"fmt"
)

// Syncer 同步器接口
type Syncer interface {
	// Upsert 单条数据插入/更新同步
	Upsert(ctx context.Context, id int64) error

	// UpsertSince 批量数据插入/更新同步
	UpsertSince(ctx context.Context, since int64) error

	// Delete 单条数据删除同步
	Delete(ctx context.Context, id int64) error
}

type SyncService interface {
	// Upsert 单条数据插入/更新同步
	Upsert(ctx context.Context, biz string, bizID int64) error
	// UpsertSince 批量数据插入/更新同步
	UpsertSince(ctx context.Context, biz string, since int64) error
	// Delete 单条数据删除同步
	Delete(ctx context.Context, biz string, bizID int64) error
}

type syncService struct {
	syncers map[string]Syncer // 只读，不需要锁
}

// NewSyncService 创建同步服务
func NewSyncService(syncers map[string]Syncer) SyncService {
	return &syncService{
		syncers: syncers,
	}
}

func (s *syncService) Upsert(ctx context.Context, biz string, bizID int64) error {
	syncer, err := s.getSyncer(biz)
	if err != nil {
		return err
	}
	return syncer.Upsert(ctx, bizID)
}

// getSyncer 获取同步器（并发安全：map 只读）
func (s *syncService) getSyncer(biz string) (Syncer, error) {
	syncer, ok := s.syncers[biz]
	if !ok {
		return nil, fmt.Errorf("未知业务类型: %s", biz)
	}
	return syncer, nil
}

func (s *syncService) UpsertSince(ctx context.Context, biz string, since int64) error {
	syncer, err := s.getSyncer(biz)
	if err != nil {
		return err
	}
	return syncer.UpsertSince(ctx, since)
}

// Delete 删除文档
func (s *syncService) Delete(ctx context.Context, biz string, bizID int64) error {
	syncer, err := s.getSyncer(biz)
	if err != nil {
		return err
	}
	return syncer.Delete(ctx, bizID)
}
