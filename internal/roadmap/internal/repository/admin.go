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
	"context"
	"fmt"
	"time"

	"github.com/gotomicro/ego/core/elog"

	"github.com/ecodeclub/ekit/sqlx"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/roadmap/internal/domain"
	"github.com/ecodeclub/webook/internal/roadmap/internal/repository/dao"
	"golang.org/x/sync/errgroup"
)

type AdminRepository interface {
	Save(ctx context.Context, r domain.Roadmap) (int64, error)
	List(ctx context.Context, offset int, limit int) (int64, []domain.Roadmap, error)
	// ListSince 分页查找Utime大于等于since的路线图，返回结果包含边信息
	ListSince(ctx context.Context, since int64, offset, limit int) ([]domain.Roadmap, error)
	GetById(ctx context.Context, id int64) (domain.Roadmap, error)
	Delete(ctx context.Context, id int64) error

	AddEdge(ctx context.Context, rid int64, edge domain.Edge) error
	DeleteEdge(ctx context.Context, id int64) error
	SanitizeData()
	SaveNodes(ctx context.Context, nodes []domain.Node) error

	SaveNode(ctx context.Context, node domain.Node) (int64, error)
	DeleteNode(ctx context.Context, id int64) error
	NodeList(ctx context.Context, rid int64) ([]domain.Node, error)
	SaveEdgeV1(ctx context.Context, rid int64, edge domain.Edge) error
	DeleteEdgeV1(ctx context.Context, id int64) error
}

var _ AdminRepository = &CachedAdminRepository{}

// CachedAdminRepository 虽然还没缓存，但是将来肯定要有缓存的
type CachedAdminRepository struct {
	converter
	dao    dao.AdminDAO
	logger *elog.Component
}

func (repo *CachedAdminRepository) SaveNodes(ctx context.Context, nodes []domain.Node) error {
	return repo.dao.SaveNodes(ctx, slice.Map(nodes, func(idx int, src domain.Node) dao.Node {
		return repo.toEntityNode(src)
	}))
}

func (repo *CachedAdminRepository) Delete(ctx context.Context, id int64) error {
	return repo.dao.Delete(ctx, id)
}

func (repo *CachedAdminRepository) SanitizeData() {
	go repo.sanitizeData()
}

func (repo *CachedAdminRepository) sanitizeData() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	roadMaps, err := repo.dao.AllRoadmap(ctx)
	cancel()
	if err != nil {
		repo.logger.Error("获取路线图失败", elog.FieldErr(err))
	}
	for _, roadMap := range roadMaps {
		// 开始清洗每个路线图
		rctx, rcancel := context.WithTimeout(context.Background(), 100000*time.Second)
		err = repo.sanitizeRoadmap(rctx, roadMap.Id)
		rcancel()
		if err == nil {
			repo.logger.Info(fmt.Sprintf("清洗路线图 %d成功", roadMap.Id))
		} else {
			repo.logger.Error(fmt.Sprintf("清洗路线图 %d失败", roadMap.Id), elog.FieldErr(err))
		}
	}
}

func (repo *CachedAdminRepository) sanitizeRoadmap(ctx context.Context, rid int64) error {
	edges, err := repo.dao.GetEdgesByRid(ctx, rid)
	if err != nil {
		return err
	}
	// 获取node
	nodeMap := make(map[string]dao.Node, len(edges)*2)
	for _, edge := range edges {
		dstkey := repo.getkey(edge.DstBiz, edge.DstId)
		if _, ok := nodeMap[dstkey]; !ok {
			nodeMap[dstkey] = dao.Node{
				Biz:   edge.DstBiz,
				Rid:   rid,
				RefId: edge.DstId,
			}
		}
		srckey := repo.getkey(edge.SrcBiz, edge.SrcId)
		if _, ok := nodeMap[srckey]; !ok {
			nodeMap[srckey] = dao.Node{
				Biz:   edge.SrcBiz,
				Rid:   rid,
				RefId: edge.SrcId,
			}
		}
	}
	nodes, err := repo.dao.CreateNodes(ctx, repo.getValues(nodeMap))
	if err != nil {
		return err
	}
	nodeMap = make(map[string]dao.Node, len(edges)*2)
	for _, node := range nodes {
		key := repo.getkey(node.Biz, node.RefId)
		nodeMap[key] = node
	}

	// 获取edgev1
	edgev1List := make([]dao.EdgeV1, 0, len(edges))
	for _, edge := range edges {
		srckey := repo.getkey(edge.SrcBiz, edge.SrcId)
		dstkey := repo.getkey(edge.DstBiz, edge.DstId)
		srcNode := nodeMap[srckey]
		dstNode := nodeMap[dstkey]
		edgev1List = append(edgev1List, dao.EdgeV1{
			Rid:     rid,
			SrcNode: srcNode.Id,
			DstNode: dstNode.Id,
		})
	}
	return repo.dao.CreateEdgeV1s(ctx, edgev1List)
}
func (repo *CachedAdminRepository) getkey(biz string, id int64) string {
	return fmt.Sprintf("%s_%d", biz, id)
}

func (repo *CachedAdminRepository) getValues(nodeMap map[string]dao.Node) []dao.Node {
	nodes := make([]dao.Node, 0, len(nodeMap))
	for _, v := range nodeMap {
		nodes = append(nodes, v)
	}
	return nodes
}

func (repo *CachedAdminRepository) SaveNode(ctx context.Context, node domain.Node) (int64, error) {
	return repo.dao.SaveNode(ctx, repo.toEntityNode(node))
}

func (repo *CachedAdminRepository) DeleteNode(ctx context.Context, id int64) error {
	return repo.dao.DeleteNode(ctx, id)
}

func (repo *CachedAdminRepository) NodeList(ctx context.Context, rid int64) ([]domain.Node, error) {
	nodes, err := repo.dao.NodeList(ctx, rid)
	if err != nil {
		return nil, err
	}
	return slice.Map(nodes, func(idx int, src dao.Node) domain.Node {
		return domain.Node{
			ID:    src.Id,
			Biz:   domain.Biz{Biz: src.Biz, BizId: src.RefId},
			Rid:   src.Rid,
			Attrs: src.Attrs,
		}
	}), nil
}

func (repo *CachedAdminRepository) SaveEdgeV1(ctx context.Context, rid int64, edge domain.Edge) error {
	return repo.dao.SaveEdgeV1(ctx, dao.EdgeV1{
		Id:      edge.Id,
		Rid:     rid,
		SrcNode: edge.Src.ID,
		DstNode: edge.Dst.ID,
		Type:    edge.Type,
		Attrs:   edge.Attrs,
	})
}

func (repo *CachedAdminRepository) DeleteEdgeV1(ctx context.Context, id int64) error {

	return repo.dao.DeleteEdgeV1(ctx, id)
}

func (repo *CachedAdminRepository) DeleteEdge(ctx context.Context, id int64) error {
	return repo.dao.DeleteEdge(ctx, id)
}

func (repo *CachedAdminRepository) AddEdge(ctx context.Context, rid int64, edge domain.Edge) error {
	return repo.dao.AddEdge(ctx, dao.Edge{
		Rid:    rid,
		SrcId:  edge.Src.BizId,
		SrcBiz: edge.Src.Biz.Biz,
		DstId:  edge.Dst.BizId,
		DstBiz: edge.Dst.Biz.Biz,
	})
}

func (repo *CachedAdminRepository) GetById(ctx context.Context, id int64) (domain.Roadmap, error) {
	var (
		eg      errgroup.Group
		r       dao.Roadmap
		edges   []dao.EdgeV1
		nodeMap map[int64]dao.Node
	)
	eg.Go(func() error {
		var err error
		r, err = repo.dao.GetById(ctx, id)
		return err
	})

	eg.Go(func() error {
		var err error
		nodeMap, edges, err = repo.dao.GetEdgesByRidV1(ctx, id)
		return err
	})
	err := eg.Wait()
	if err != nil {
		return domain.Roadmap{}, err
	}
	res := repo.toDomain(r)
	res.Edges = repo.edgesToDomain(edges, nodeMap)
	return res, nil
}

func (repo *CachedAdminRepository) List(ctx context.Context, offset int, limit int) (int64, []domain.Roadmap, error) {
	count, rs, err := repo.dao.List(ctx, offset, limit)
	return count, slice.Map(rs, func(idx int, src dao.Roadmap) domain.Roadmap {
		return repo.toDomain(src)
	}), err
}

func (repo *CachedAdminRepository) ListSince(ctx context.Context, since int64, offset, limit int) ([]domain.Roadmap, error) {
	rs, err := repo.dao.ListSince(ctx, since, offset, limit)
	if err != nil {
		return nil, err
	}

	if len(rs) == 0 {
		return []domain.Roadmap{}, nil
	}

	rids := slice.Map(rs, func(idx int, src dao.Roadmap) int64 {
		return src.Id
	})

	nodeMap, edgeMap, err := repo.dao.GetEdgesByRidsV1(ctx, rids)
	if err != nil {
		return nil, err
	}

	return slice.Map(rs, func(idx int, src dao.Roadmap) domain.Roadmap {
		rd := repo.toDomain(src)
		rd.Edges = repo.edgesToDomain(edgeMap[src.Id], nodeMap)
		return rd
	}), err
}

func (repo *CachedAdminRepository) Save(ctx context.Context, r domain.Roadmap) (int64, error) {
	return repo.dao.Save(ctx, repo.toEntity(r))
}

func (repo *CachedAdminRepository) toEntity(r domain.Roadmap) dao.Roadmap {
	return dao.Roadmap{
		Id:    r.Id,
		Title: r.Title,
		Biz:   sqlx.NewNullString(r.Biz.Biz),
		BizId: sqlx.NewNullInt64(r.BizId),
	}
}

func (repo *CachedAdminRepository) toEntityNode(node domain.Node) dao.Node {
	return dao.Node{
		Id:    node.ID,
		Biz:   node.Biz.Biz,
		Rid:   node.Rid,
		RefId: node.Biz.BizId,
		Attrs: node.Attrs,
	}
}

func NewCachedAdminRepository(dao dao.AdminDAO) AdminRepository {
	return &CachedAdminRepository{
		dao:    dao,
		logger: elog.DefaultLogger,
	}
}
