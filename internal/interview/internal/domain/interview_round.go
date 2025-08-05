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

// RoundResult 定义了面试轮次的官方结果。
type RoundResult string

// 定义面试轮次结果的枚举常量
const (
	ResultPending  RoundResult = "PENDING"
	ResultApproved RoundResult = "APPROVED"
	ResultRejected RoundResult = "REJECTED"
)

// IsValid 检查给定的结果字符串是否为有效的 RoundResult 枚举值。
func (r RoundResult) IsValid() bool {
	switch r {
	case ResultPending, ResultApproved, ResultRejected:
		return true
	default:
		return false
	}
}

func (r RoundResult) String() string {
	return string(r)
}

// InterviewRound 是面试轮次的领域模型。
// 它的业务一致性由其所属的 InterviewJourney 聚合根来维护。
type InterviewRound struct {
	ID            int64
	Jid           int64
	Uid           int64
	RoundNumber   int
	RoundType     string
	InterviewDate int64
	JobInfo       string
	ResumeURL     string
	AudioURL      string
	SelfResult    bool
	SelfSummary   string
	Result        RoundResult
	AllowSharing  bool
}

// IsShared 检查本轮面试是否已授权公开。
func (r *InterviewRound) IsShared() bool {
	return r.AllowSharing
}
