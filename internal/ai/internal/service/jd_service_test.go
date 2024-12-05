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
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestJSONExpression 测试利用正则表达式提取 JSON 串
func TestJSONExpression(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "本身就是JSON",
			input: `{"abc": "bcd"}`,
			want:  `{"abc": "bcd"}`,
		},
		{
			name:  "有前缀后缀",
			input: "```json{\"abc\": \"bcd\"}```",
			want:  `{"abc": "bcd"}`,
		},
	}

	expr := regexp.MustCompile(jsonExpr)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			val := expr.FindString(tc.input)
			assert.Equal(t, tc.want, val)
		})
	}
}
