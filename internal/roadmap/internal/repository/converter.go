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

package repository

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/roadmap/internal/domain"
	"github.com/ecodeclub/webook/internal/roadmap/internal/repository/dao"
)

// 公共的转换放过来这里
type converter struct {
}

func (converter) toDomain(r dao.Roadmap) domain.Roadmap {
	return domain.Roadmap{
		Id:    r.Id,
		Title: r.Title,
		Biz:   r.Biz.String,
		BizId: r.BizId.Int64,
		Utime: r.Utime,
	}
}

func (converter) edgesToDomain(edges []dao.Edge) []domain.Edge {
	return slice.Map(edges, func(idx int, edge dao.Edge) domain.Edge {
		return domain.Edge{
			Id: edge.Id,
			Src: domain.Node{
				Biz: domain.Biz{
					BizId: edge.SrcId,
					Biz:   edge.SrcBiz,
				},
			},
			Dst: domain.Node{
				Biz: domain.Biz{
					BizId: edge.DstId,
					Biz:   edge.DstBiz,
				},
			},
		}
	})
}
