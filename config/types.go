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

package config

type WebookConfig struct {
	DB           DBConfig
	EmailConf    EmailConfig
	EmailVfyConf EmailVfyConfig
}

type DBConfig struct {
	DSN string
}

type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
}

type EmailVfyConfig struct {
	Issuer      string
	Key         string
	AbsoluteURL string // 跳转到前端的绝对URL
}
