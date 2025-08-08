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

// JourneyStatus 定义了面试历程的有效状态，使用自定义类型以获得类型安全。
type JourneyStatus string

// 定义面试历程状态的枚举常量
const (
	StatusActive    JourneyStatus = "ACTIVE"
	StatusSucceeded JourneyStatus = "SUCCEEDED"
	StatusFailed    JourneyStatus = "FAILED"
	StatusAbandoned JourneyStatus = "ABANDONED"
)

// IsValid 检查给定的状态字符串是否为有效的 JourneyStatus 枚举值。
// Service层在接收到外部输入时，可以使用此方法进行校验。
func (s JourneyStatus) IsValid() bool {
	switch s {
	case StatusActive, StatusSucceeded, StatusFailed, StatusAbandoned:
		return true
	default:
		return false
	}
}

func (s JourneyStatus) String() string {
	return string(s)
}

func (s JourneyStatus) IsActive() bool {
	return s == StatusActive
}

// InterviewJourney 是面试历程的领域模型，也是聚合根。
// 它代表一个完整的业务概念，并聚合了与之相关的面试轮次。
type InterviewJourney struct {
	ID          int64
	Uid         int64
	CompanyID   int64
	CompanyName string
	JobInfo     string
	ResumeURL   string
	Status      JourneyStatus
	Stime       int64
	Etime       int64

	// 聚合关系：一个面试历程包含多个面试轮次
	Rounds []InterviewRound
}

func (i InterviewJourney) IsValid() bool {
	if i.Uid == 0 ||
		i.CompanyName == "" ||
		i.JobInfo == "" ||
		i.ResumeURL == "" ||
		i.Stime == 0 ||
		!i.Status.IsValid() {
		return false
	}
	return true
}

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
func (r InterviewRound) IsShared() bool {
	return r.AllowSharing
}

func (r InterviewRound) IsValid() bool {
	if r.Uid == 0 ||
		r.RoundNumber == 0 ||
		r.RoundType == "" ||
		r.InterviewDate == 0 ||
		r.JobInfo == "" ||
		r.ResumeURL == "" ||
		!r.Result.IsValid() {
		return false
	}
	return true
}
