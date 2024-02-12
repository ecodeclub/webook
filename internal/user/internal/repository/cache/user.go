package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/webook/internal/user/internal/domain"
	"github.com/redis/go-redis/v9"
)

// ErrKeyNotExist 因为我们目前还是只有一个实现，所以可以保持用别名
var ErrKeyNotExist = redis.Nil

//go:generate mockgen -source=./user.go -package=cachemocks -destination=mocks/user.mock.go UserCache
type UserCache interface {
	Delete(ctx context.Context, id int64) error
	Get(ctx context.Context, id int64) (domain.User, error)
	Set(ctx context.Context, u domain.User) error
}

type UserECache struct {
	cache ecache.Cache
	// 过期时间
	expiration time.Duration
}

// NewUserECache 注意缓存前缀
func NewUserECache(c ecache.Cache) UserCache {
	return &UserECache{
		cache: &ecache.NamespaceCache{
			Namespace: "user:",
			C:         c,
		},
		expiration: time.Minute * 15,
	}
}
func (cache *UserECache) Delete(ctx context.Context, id int64) error {
	_, err := cache.cache.Delete(ctx, cache.key(id))
	return err
}

func (cache *UserECache) Get(ctx context.Context, id int64) (domain.User, error) {
	key := cache.key(id)
	var u domain.User
	err := cache.cache.Get(ctx, key).JSONScan(&u)
	// 反序列化回来
	return u, err
}

func (cache *UserECache) Set(ctx context.Context, u domain.User) error {
	data, err := json.Marshal(u)
	if err != nil {
		return err
	}
	key := cache.key(u.Id)
	return cache.cache.Set(ctx, key, data, cache.expiration)
}

func (cache *UserECache) key(id int64) string {
	return fmt.Sprintf("webook:user:info:%d", id)
}
