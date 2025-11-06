package testioc

import (
	"context"
	"fmt"
	"time"

	"github.com/ecodeclub/ekit/retry"
	"github.com/elastic/go-elasticsearch/v9"

	"github.com/gotomicro/ego/core/econf"
)

func InitES() *elasticsearch.TypedClient {
	econf.Set("es.url", "http://127.0.0.1:9200")
	econf.Set("es.sniff", false)
	type Config struct {
		Url      string `yaml:"url"`
		Sniff    bool   `yaml:"sniff"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	}
	var cfg Config
	err := econf.UnmarshalKey("es", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 ES 配置失败 %w", err))
	}

	esCfg := elasticsearch.Config{
		Addresses: []string{cfg.Url},
		Username:  cfg.Username,
		Password:  cfg.Password,
	}

	client, err := elasticsearch.NewTypedClient(esCfg)
	if err != nil {
		panic(err)
	}
	const maxInterval = 10 * time.Second
	const maxRetries = 10
	const timeout = 5 * time.Second
	strategy, err := retry.NewExponentialBackoffRetryStrategy(time.Second, maxInterval, maxRetries)
	if err != nil {
		panic(err)
	}
	for {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		_, err = client.Ping().Do(ctx)
		cancel()
		if err == nil {
			break
		}
		next, ok := strategy.Next()
		if !ok {
			panic("WaitForDBSetup 重试失败......")
		}
		time.Sleep(next)
	}
	return client
}
