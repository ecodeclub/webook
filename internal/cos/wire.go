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

//go:build wireinject

package cos

import (
	"github.com/ecodeclub/webook/internal/cos/internal/web"
	"github.com/google/wire"
)

func InitHandler(cfg Config) *Handler {
	wire.Build(initHandler)
	return new(Handler)
}

func initHandler(cfg Config) *Handler {
	return web.NewHandler(cfg.SecretID, cfg.SecretKey, cfg.AppID, cfg.Bucket, cfg.Region)
}

type Handler = web.Handler

type Config struct {
	SecretKey string `json:"secretKey"`
	SecretID  string `json:"secretID"`
	AppID     string `json:"appID"`
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
}
