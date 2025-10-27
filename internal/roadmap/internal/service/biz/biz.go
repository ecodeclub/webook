package biz

import (
	"context"
	"errors"
	"sync"

	"github.com/ecodeclub/ekit/mapx"
	"github.com/ecodeclub/webook/internal/roadmap/internal/domain"
	"golang.org/x/sync/errgroup"
)

// ConcurrentBizService 强调并发
type ConcurrentBizService struct {
	BizStrategyMap map[string]Strategy
}

func NewConcurrentBizService(bizStrategtMap map[string]Strategy) Service {
	return &ConcurrentBizService{
		BizStrategyMap: bizStrategtMap,
	}
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
			return nil
		})
	}
	err := eg.Wait()
	return res, err
}

// GetBizsByIds 将来可能需要暴露出去，暂时保留定义为公共接口
func (svc *ConcurrentBizService) GetBizsByIds(ctx context.Context, biz string, ids []int64) (map[int64]domain.Biz, error) {
	// biz 不是很多，所以可以用 switch
	// 后续可以重构为策略模式
	strategy, ok := svc.BizStrategyMap[biz]
	if !ok {
		return nil, errors.New("biz not exist")
	}
	return strategy.GetBizsByIds(ctx, ids)
}
