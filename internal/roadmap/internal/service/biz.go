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
	"sync"

	"github.com/ecodeclub/ekit/mapx"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/roadmap/internal/domain"
	"golang.org/x/sync/errgroup"
)

// BizService 作为一个聚合服务，下沉到这里以减轻 web 的逻辑负担
type BizService interface {
	// GetBizs bizs 和 ids 的长度必须一样
	// 返回值是 biz-id-Biz 的结构
	GetBizs(ctx context.Context, bizs []string, ids []int64) (map[string]map[int64]domain.Biz, error)
}

var _ BizService = &ConcurrentBizService{}

// ConcurrentBizService 强调并发
type ConcurrentBizService struct {
	queSvc    baguwen.Service
	queSetSvc baguwen.QuestionSetService
}

func (svc *ConcurrentBizService) GetBizs(ctx context.Context, bizs []string, ids []int64) (map[string]map[int64]domain.Biz, error) {
	// 先按照 biz 分组
	// biz 不会有很多
	bizIdMap := mapx.NewMultiBuiltinMap[string, int64](4)
	for i := 0; i < len(bizs); i++ {
		// 这里不对长度做检测，调用者负责确保长度一致
		_ = bizIdMap.Put(bizs[i], ids[i])
	}

	var eg errgroup.Group
	keys := bizIdMap.Keys()
	var lock sync.Mutex
	res := make(map[string]map[int64]domain.Biz, len(keys))
	for _, key := range keys {
		bizIds, ok := bizIdMap.Get(key)
		if !ok {
			continue
		}

		// 1.22 之后可以去掉
		key := key
		eg.Go(func() error {
			bizMap, err := svc.GetBizsByIds(ctx, key, bizIds)
			if err == nil {
				lock.Lock()
				res[key] = bizMap
				lock.Unlock()
			}
			return err
		})
	}
	err := eg.Wait()
	return res, err
}

// GetBizsByIds 将来可能需要暴露出去，暂时保留定义为公共接口
func (svc *ConcurrentBizService) GetBizsByIds(ctx context.Context, biz string, ids []int64) (map[int64]domain.Biz, error) {
	// biz 不是很多，所以可以用 switch
	// 后续可以重构为策略模式
	switch biz {
	case domain.BizQuestion:
		return svc.getQuestions(ctx, ids)
	case domain.BizQuestionSet:
		return svc.getQuestionSet(ctx, ids)
	default:
		return nil, fmt.Errorf("不支持的 Biz: %s", biz)
	}
}

func (svc *ConcurrentBizService) getQuestions(ctx context.Context, ids []int64) (map[int64]domain.Biz, error) {
	ques, err := svc.queSvc.GetPubByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	res := make(map[int64]domain.Biz, len(ques))
	for _, que := range ques {
		res[que.Id] = domain.Biz{
			Biz:   domain.BizQuestion,
			BizId: que.Id,
			Title: que.Title,
		}
	}
	return res, nil
}

func (svc *ConcurrentBizService) getQuestionSet(ctx context.Context, ids []int64) (map[int64]domain.Biz, error) {
	qs, err := svc.queSetSvc.GetByIds(ctx, ids)
	if err != nil {
		return nil, err
	}
	res := make(map[int64]domain.Biz, len(qs))
	for _, q := range qs {
		res[q.Id] = domain.Biz{
			Biz:   domain.BizQuestionSet,
			BizId: q.Id,
			Title: q.Title,
		}
	}
	return res, nil
}

func NewConcurrentBizService(queSvc baguwen.Service, queSetSvc baguwen.QuestionSetService) BizService {
	return &ConcurrentBizService{queSvc: queSvc, queSetSvc: queSetSvc}
}
