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

package biz

import (
	"github.com/ecodeclub/webook/internal/ai/internal/service/gpt/handler"
)

// GPTBizHandler 近似于标记接口，也就是用于区分专属于业务的，和通用的 Handler
type GPTBizHandler interface {
	handler.Handler
	// Biz 它处理的业务
	Biz() string
}
