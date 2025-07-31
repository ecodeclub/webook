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
	UserNotFoundError   = ErrorCode{Code: 418001, Msg: "用户未找到"}
	PhoneNotLinkedError = ErrorCode{Code: 418002, Msg: "用户未绑定手机号"}

	SystemError               = ErrorCode{Code: 518001, Msg: "系统错误"}
	ReceieveNotifyFailedError = ErrorCode{Code: 518002, Msg: "用户接收通知失败"}
)

type ErrorCode struct {
	Code int
	Msg  string
}
