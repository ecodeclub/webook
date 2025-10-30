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

	chatv1 "github.com/ecodeclub/webook/api/proto/gen/chat/v1"
	"github.com/gotomicro/ego/core/econf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitGrpcClient() (chatv1.ServiceClient, error) {
	type Config struct {
		Addr string `yaml:"addr"`
	}
	var cfg Config
	err := econf.UnmarshalKey("grpc.aiGateway", &cfg)
	if err != nil {
		return nil, fmt.Errorf("读取 grpc.aiGateway 配置失败: %w", err)
	}
	conn, err := grpc.NewClient(cfg.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接 grpc 服务失败: %w", err)
	}
	return chatv1.NewServiceClient(conn), nil
}
