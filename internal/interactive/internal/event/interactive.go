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

package event

import (
	"context"

	"github.com/ecodeclub/webook/internal/interactive/internal/service"
)

type Event struct {
	Biz   string `json:"biz,omitempty"`
	BizId int64  `json:"bizId,omitempty"`
	// 取值是
	// like, collect, view 三个
	Action string `json:"action,omitempty"`
	Uid    int64  `json:"uid,omitempty"`
}
type handleFunc func(ctx context.Context, svc service.Service, evt Event) error
