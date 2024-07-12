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

type ExamineResult struct {
	Qid    int64
	Result Result
	// 原始回答，源自 AI
	RawResult string

	// 使用的 token 数量
	Tokens int
	// 花费的金额
	Amount int64
	Tid    string
}

type Result uint8

func (r Result) ToUint8() uint8 {
	return uint8(r)
}

const (
	// ResultFailed 完全没通过，或者完全没有考过，我们不需要区别这两种状态
	ResultFailed Result = iota
	// ResultBasic 只回答出来了 15K 的部分
	ResultBasic
	// ResultIntermediate 回答了 25K 部分
	ResultIntermediate
	// ResultAdvanced 回答出来了 35K 部分
	ResultAdvanced
)
