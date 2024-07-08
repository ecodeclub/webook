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

type Roadmap struct {
	Id    int64
	Title string

	Biz   string
	BizId int64

	Edges []Edge
	Utime int64
}

func (r Roadmap) Bizs() ([]string, []int64) {
	// SRC + DST，所以乘以 2，而后加上本体的 biz
	bizs := make([]string, 0, len(r.Edges)*2+1)
	bizIds := make([]int64, 0, len(r.Edges)*2+1)
	for _, edge := range r.Edges {
		bizs = append(bizs, edge.Src.Biz.Biz, edge.Dst.Biz.Biz)
		bizIds = append(bizIds, edge.Src.BizId, edge.Dst.BizId)
	}
	// 加上本身的
	bizs = append(bizs, r.Biz)
	bizIds = append(bizIds, r.BizId)
	return bizs, bizIds
}

type Node struct {
	Biz
}

type Edge struct {
	Id  int64
	Src Node
	Dst Node
}

type Biz struct {
	Biz   string
	BizId int64
	Title string
}

const (
	BizQuestion    = "question"
	BizQuestionSet = "questionSet"
)
