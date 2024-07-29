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

type Interactive struct {
	Biz        string
	BizId      int64
	ViewCnt    int
	LikeCnt    int
	CollectCnt int
	Liked      bool
	Collected  bool
}

type Collection struct {
	Id int64
	// 用户 ID
	Uid  int64
	Name string
}

type CollectionRecord struct {
	// 用于分发的
	Biz         string
	Case        int64
	Question    int64
	QuestionSet int64
}
