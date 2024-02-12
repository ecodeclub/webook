package testioc

import (
	"github.com/ecodeclub/ecache"
	eredis "github.com/ecodeclub/ecache/redis"
	"github.com/redis/go-redis/v9"
)

var cache ecache.Cache

func InitCache() ecache.Cache {
	if cache != nil {
		return cache
	}
	cmd := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	return &ecache.NamespaceCache{
		C:         eredis.NewCache(cmd),
		Namespace: "webook:",
	}
}
