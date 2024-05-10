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

package events

import (
	"context"

	"github.com/ecodeclub/webook/internal/interactive/internal/service"
)

type Event struct {
	Biz   string `json:"biz,omitempty"`
	BizId int64  `json:"biz_id,omitempty"`
	// 取值是
	// like, collect, view 三个
	Action string `json:"action,omitempty"`
	Uid    int64  `json:"uid,omitempty"`
}
type handleFunc func(ctx context.Context, svc service.InteractiveService, evt Event) error

func likeHandle(ctx context.Context, svc service.InteractiveService, evt Event) error {
	return svc.Like(ctx, evt.Biz, evt.BizId, evt.Uid)
}
func collectHandle(ctx context.Context, svc service.InteractiveService, evt Event) error {
	return svc.Collect(ctx, evt.Biz, evt.BizId, evt.Uid)
}
func viewHandle(ctx context.Context, svc service.InteractiveService, evt Event) error {
	return svc.IncrReadCnt(ctx, evt.Biz, evt.BizId)
}
