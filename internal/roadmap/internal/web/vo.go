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

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/roadmap/internal/domain"
)

type AddEdgeReq struct {
	// roadmap 的 ID
	Rid  int64
	Edge Edge
}

type Page struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

type RoadmapListResp struct {
	Total int       `json:"total,omitempty"`
	Maps  []Roadmap `json:"maps,omitempty"`
}

type Roadmap struct {
	Id       int64  `json:"id"`
	Title    string `json:"title"`
	Biz      string `json:"biz"`
	BizId    int64  `json:"bizId"`
	BizTitle string `json:"bizTitle"`
	Utime    int64  `json:"utime"`
	Edges    []Edge `json:"edges"`
}

func newRoadmapWithBiz(r domain.Roadmap,
	bizMap map[string]map[int64]domain.Biz) Roadmap {
	rm := newRoadmap(r)
	rm.BizTitle = bizMap[r.Biz][r.BizId].Title
	rm.Edges = slice.Map(r.Edges, func(idx int, edge domain.Edge) Edge {
		src := newNode(edge.Src)
		src.Title = bizMap[src.Biz][src.BizId].Title
		dst := newNode(edge.Dst)
		dst.Title = bizMap[dst.Biz][dst.BizId].Title
		return Edge{
			Id:    edge.Id,
			Type:  edge.Type,
			Attrs: edge.Attrs,
			Src:   src,
			Dst:   dst,
		}
	})
	return rm
}

func newRoadmap(r domain.Roadmap) Roadmap {
	return Roadmap{
		Id:    r.Id,
		Title: r.Title,
		Biz:   r.Biz,
		BizId: r.BizId,
		Utime: r.Utime,
	}
}

func (r Roadmap) toDomain() domain.Roadmap {
	return domain.Roadmap{
		Id:    r.Id,
		Title: r.Title,
		Biz:   r.Biz,
		BizId: r.BizId,
		Utime: r.Utime,
	}
}

type IdReq struct {
	Id int64 `json:"id,omitempty"`
}

type Node struct {
	ID    int64  `json:"id"`
	Rid   int64  `json:"rid"`
	Attrs string `json:"attrs"`
	BizId int64  `json:"bizId"`
	Biz   string `json:"biz"`
	Title string `json:"title"`
}

func (n Node) toDomain() domain.Node {
	return domain.Node{
		ID:    n.ID,
		Rid:   n.Rid,
		Attrs: n.Attrs,
		Biz: domain.Biz{
			BizId: n.BizId,
			Biz:   n.Biz,
		},
	}
}

func newNode(node domain.Node) Node {
	return Node{
		ID:    node.ID,
		Rid:   node.Rid,
		Attrs: node.Attrs,
		BizId: node.BizId,

		Biz:   node.Biz.Biz,
		Title: node.Title,
	}
}

type Edge struct {
	Id    int64  `json:"id"`
	Type  string `json:"type"`
	Attrs string `json:"attrs"`
	Src   Node   `json:"src"`
	Dst   Node   `json:"dst"`
}

func (e Edge) toDomain() domain.Edge {
	return domain.Edge{
		Id:    e.Id,
		Type:  e.Type,
		Attrs: e.Attrs,
		Src:   e.Src.toDomain(),
		Dst:   e.Dst.toDomain(),
	}
}

type Biz struct {
	Biz   string `json:"biz"`
	BizId int64  `json:"bizId"`
}
