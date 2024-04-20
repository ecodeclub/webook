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

package domain

type Credit struct {
	Uid               int64
	TotalAmount       uint64
	LockedTotalAmount uint64
	Logs              []CreditLog
}

type CreditLog struct {
	ID           int64
	Key          string
	ChangeAmount int64
	Biz          string
	BizId        int64
	Desc         string
}
