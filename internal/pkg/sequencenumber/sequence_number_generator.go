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

package sequencenumber

import (
	"fmt"
	"time"

	"github.com/lithammer/shortuuid/v4"
)

// TimestampGenerateFunc 定义生成时间戳的函数类型
type TimestampGenerateFunc func(time.Time) int64

// ShortUUIDGenerateFunc 定义生成ShortUUID的函数类型
type ShortUUIDGenerateFunc func() string

// Generator 包含时间和UUID生成函数
type Generator struct {
	timestampGenFunc TimestampGenerateFunc
	shortUUIDGenFunc ShortUUIDGenerateFunc
}

// NewGeneratorWith 创建一个Generator实例
func NewGeneratorWith(timestampGen TimestampGenerateFunc, uuidGen ShortUUIDGenerateFunc) *Generator {
	return &Generator{
		timestampGenFunc: timestampGen,
		shortUUIDGenFunc: uuidGen,
	}
}

// NewGenerator 创建一个Generator实例
func NewGenerator() *Generator {
	return NewGeneratorWith(func(t time.Time) int64 { return t.UnixMilli() }, func() string { return shortuuid.New() })
}

// Generate 使用ID生成序列号，生成 32 位长度的字符串
func (s *Generator) Generate(id int64) (string, error) {
	timestamp := s.timestampGenFunc(time.Now())
	lastFour := fmt.Sprintf("%04d", id%10000)
	uuid := s.shortUUIDGenFunc()
	// timestamp 的16进制编码 + 用户后四位 + (uuid 凑够位数) == 32 位
	return fmt.Sprintf("%d%s%s", timestamp, lastFour, uuid)[:32], nil
}
