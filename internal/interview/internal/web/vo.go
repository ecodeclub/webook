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

type SaveReq struct {
	Journey Journey `json:"journey"`
}

type Journey struct {
	ID          int64  `json:"id"`
	CompanyID   int64  `json:"companyId"`
	CompanyName string `json:"companyName"`
	JobInfo     string `json:"jobInfo"`
	ResumeURL   string `json:"resumeURL"`
	Status      string `json:"status"`
	Stime       int64  `json:"stime"`
	Etime       int64  `json:"etime"`

	Rounds []Round `json:"rounds,omitzero"` // 仅在详情页中填充
}

type Round struct {
	ID            int64  `json:"id"`
	RoundNumber   int    `json:"roundNumber"`
	RoundType     string `json:"roundType"`
	InterviewDate int64  `json:"interviewDate"`
	JobInfo       string `json:"jobInfo"`
	ResumeURL     string `json:"resumeURL"`
	AudioURL      string `json:"audioURL"`
	SelfResult    bool   `json:"selfResult"`
	SelfSummary   string `json:"selfSummary"`
	Result        string `json:"result"`
	AllowSharing  bool   `json:"allowSharing"`
}

type DetailReq struct {
	ID int64 `json:"id"`
}

type ListReq struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}
