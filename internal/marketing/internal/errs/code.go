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

package errs

var (
	RedemptionCodeUsedError   = ErrorCode{Code: 412001, Msg: "兑换码已使用"}
	RedemptionCodeNotFoundErr = ErrorCode{Code: 412002, Msg: "兑换码不存在"}

	SystemError = ErrorCode{Code: 512001, Msg: "系统错误"}
)

type ErrorCode struct {
	Code int
	Msg  string
}
