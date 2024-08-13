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

package web

import "github.com/ecodeclub/webook/internal/question/internal/domain"

type ExamineReq struct {
	Qid   int64  `json:"qid"`
	Input string `json:"input"`
}

type ExamineResult struct {
	Qid    int64
	Result uint8 `json:"result"`
	// 原始回答，源自 AI
	RawResult string `json:"rawResult"`

	// 使用的 token 数量
	Tokens int64 `json:"tokens"`
	// 花费的金额
	Amount int64 `json:"amount"`
}

func newExamineResult(r domain.ExamineResult) ExamineResult {
	return ExamineResult{
		Qid:       r.Qid,
		Result:    r.Result.ToUint8(),
		RawResult: r.RawResult,
		Amount:    r.Amount,
	}
}

type CorrectReq struct {
	Qid int64 `json:"qid"`
	// 修正结果，对应 domain.Result
	Result uint8 `json:"result"`
}
