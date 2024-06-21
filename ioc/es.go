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
	"fmt"
	"time"

	"github.com/gotomicro/ego/core/econf"
	"github.com/olivere/elastic/v7"
)

func InitES() *elastic.Client {
	type Config struct {
		Url   string `yaml:"url"`
		Sniff bool   `yaml:"sniff"`
	}
	var cfg Config
	err := econf.UnmarshalKey("es", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 ES 配置失败 %w", err))
	}
	const timeout = 10 * time.Second
	opts := []elastic.ClientOptionFunc{
		elastic.SetURL(cfg.Url),
		elastic.SetSniff(cfg.Sniff),
		elastic.SetHealthcheckTimeoutStartup(timeout),
	}
	client, err := elastic.NewClient(opts...)
	if err != nil {
		panic(err)
	}
	return client
}
