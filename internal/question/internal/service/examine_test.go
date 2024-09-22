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

package service

import (
	"testing"

	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestLLMExamineService_findFirstZeroPosition(t *testing.T) {
	svc := &LLMExamineService{}
	testCases := []struct {
		name    string
		input   byte
		wantRes int
	}{
		{
			name:    "1分",
			input:   1,
			wantRes: 1,
		},
		{
			name:    "2分",
			input:   2,
			wantRes: 0,
		},
		{
			name:    "3分",
			input:   3,
			wantRes: 2,
		},
		{
			name:    "5分",
			input:   5,
			wantRes: 1,
		},
		{
			name:    "6分",
			input:   6,
			wantRes: 0,
		},
		{
			name:    "7分",
			input:   7,
			wantRes: 3,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := svc.findFirstZeroPosition(tc.input)
			assert.Equal(t, tc.wantRes, res)
		})
	}
}

func TestLLMExamineService_parseExamineResult(t *testing.T) {
	testCases := []struct {
		name      string
		llmResult string
		wantRes   domain.Result
	}{
		{
			name: "未通过",
			llmResult: `
#### 最终评分
  0分
#### 标准答案关键点
`,
			wantRes: domain.ResultFailed,
		},
		{
			name: "15K",
			llmResult: `
#### 最终评分
  1分
回答出来了第一个部分
#### 标准答案关键点
`,
			wantRes: domain.ResultBasic,
		},
	}

	svc := &LLMExamineService{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := svc.parseExamineResult(tc.llmResult)
			assert.Equal(t, tc.wantRes, res)
		})
	}
}
