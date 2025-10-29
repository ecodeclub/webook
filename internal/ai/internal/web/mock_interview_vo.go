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

type CreateMockInterviewReq struct {
	Title string `json:"title"`
}

type StreamMockInterviewReq struct {
	InterviewID string `json:"interviewId"`
	Content     string `json:"content"`
	AudioURL    string `json:"audioUrl"`
	ConfigID    int64  `json:"configId"`
}

type COSTempCredentialsResp struct {
	TmpSecretId  string `json:"tmpSecretId"`
	TmpSecretKey string `json:"tmpSecretKey"`
	SessionToken string `json:"sessionToken"`
	StartTime    int64  `json:"startTime"`
	ExpiredTime  int64  `json:"expiredTime"`
	Bucket       string `json:"bucket"`
	Region       string `json:"region"`
}
