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

package ioc

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/gotomicro/ego/core/econf"
)

func InitES() *elasticsearch.TypedClient {
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

	// 健康检查
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = client.Ping().Do(ctx)
	if err != nil {
		panic(fmt.Errorf("ES 连接失败 %w", err))
	}
	return client
}
