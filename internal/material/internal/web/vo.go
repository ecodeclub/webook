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

// SubmitMaterialReq 提交素材请求
type SubmitMaterialReq struct {
	Material Material `json:"material"`
}

type Material struct {
	ID        int64  `json:"id"`
	Uid       int64  `json:"uid"`
	AudioURL  string `json:"audioURL"`
	ResumeURL string `json:"resumeURL"`
	Remark    string `json:"remark"`
	Status    string `json:"status"`
	Ctime     int64  `json:"ctime"`
	Utime     int64  `json:"utime"`
}

// ListMaterialsReq 分页查询用户所提交的素材
type ListMaterialsReq struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

type ListMaterialsResp struct {
	Total     int64      `json:"total"`
	Materials []Material `json:"materials"`
}

type AcceptMaterialReq struct {
	ID int64 `json:"id"`
}

type NotifyUserReq struct {
	ID   int64  `json:"id"`
	Date string `json:"date"`
}
