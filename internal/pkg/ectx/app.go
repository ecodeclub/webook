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

package ectx

import (
	"context"
)

const (
	appCtxKey = "app"
)

// AppFromCtx 这个只会检测 app 本身，并不会从 uid 里面推测
// 如果不存在，就返回0，表示这个是 webook 本体
func AppFromCtx(ctx context.Context) (uint, bool) {
	app := ctx.Value(appCtxKey)
	val, ok := app.(uint)
	return val, ok
}
