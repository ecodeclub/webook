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

package dao

import (
	"context"
	"time"

	"github.com/ego-component/egorm"
	"gorm.io/gorm/clause"
)

type AdminDAO interface {
	// 路线图的相关方法
	Save(ctx context.Context, r Roadmap) (int64, error)
	GetById(ctx context.Context, id int64) (Roadmap, error)
	List(ctx context.Context, offset int, limit int) ([]Roadmap, error)
	AllRoadmap(ctx context.Context) ([]Roadmap, error)

	// 旧版本边的操作
	GetEdgesByRid(ctx context.Context, rid int64) ([]Edge, error)
	AddEdge(ctx context.Context, edge Edge) error
	DeleteEdge(ctx context.Context, id int64) error

	// 新版本节点的操作
	SaveNode(ctx context.Context, node Node) (int64, error)
	DeleteNode(ctx context.Context, id int64) error
	NodeList(ctx context.Context, rid int64) ([]Node, error)
	CreateNodes(ctx context.Context, nodes []Node) ([]Node, error)

	// 新版本边的操作
	GetEdgesByRidV1(ctx context.Context, rid int64) (map[int64]Node, []EdgeV1, error)
	CreateEdgeV1s(ctx context.Context, edgev1List []EdgeV1) error
	SaveEdgeV1(ctx context.Context, edge EdgeV1) error
	DeleteEdgeV1(ctx context.Context, id int64) error
}

var _ AdminDAO = &GORMAdminDAO{}

type GORMAdminDAO struct {
	db *egorm.Component
}

func (dao *GORMAdminDAO) AllRoadmap(ctx context.Context) ([]Roadmap, error) {
	var res []Roadmap
	err := dao.db.WithContext(ctx).Order("id DESC").Find(&res).Error
	return res, err
}

func (dao *GORMAdminDAO) CreateNodes(ctx context.Context, nodes []Node) ([]Node, error) {
	now := time.Now().UnixMilli()
	for idx := range nodes {
		nodes[idx].Ctime = now
		nodes[idx].Utime = now
	}
	err := dao.db.WithContext(ctx).Create(&nodes).Error
	return nodes, err
}

func (dao *GORMAdminDAO) CreateEdgeV1s(ctx context.Context, edgev1List []EdgeV1) error {
	now := time.Now().UnixMilli()
	for idx := range edgev1List {
		edgev1List[idx].Ctime = now
		edgev1List[idx].Utime = now
	}
	return dao.db.WithContext(ctx).Create(&edgev1List).Error
}

func (dao *GORMAdminDAO) GetEdgesByRidV1(ctx context.Context, rid int64) (map[int64]Node, []EdgeV1, error) {
	var edges []EdgeV1
	err := dao.db.WithContext(ctx).Where("rid = ?", rid).
		Order("id desc").
		Find(&edges).Error
	if err != nil {
		return nil, nil, err
	}

	nodeIds := make(map[int64]struct{})
	for _, edge := range edges {
		nodeIds[edge.SrcNode] = struct{}{}
		nodeIds[edge.DstNode] = struct{}{}
	}

	var nodes []Node
	err = dao.db.WithContext(ctx).Where("id IN ?", keys(nodeIds)).Find(&nodes).Error
	if err != nil {
		return nil, nil, err
	}
	nodeMap := make(map[int64]Node, len(nodes))
	for _, node := range nodes {
		nodeMap[node.Id] = node
	}
	return nodeMap, edges, nil
}

func keys(m map[int64]struct{}) []int64 {
	ks := make([]int64, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func (dao *GORMAdminDAO) SaveNode(ctx context.Context, node Node) (int64, error) {
	now := time.Now().UnixMilli()
	node.Utime = now
	node.Ctime = now
	err := dao.db.WithContext(ctx).
		Clauses(
			clause.OnConflict{
				DoUpdates: clause.AssignmentColumns([]string{"biz", "rid", "ref_id", "attrs", "utime"}),
			},
		).
		Create(&node).Error
	return node.Id, err
}

func (dao *GORMAdminDAO) DeleteNode(ctx context.Context, id int64) error {
	return dao.db.
		WithContext(ctx).
		Where("id = ?", id).Delete(&Node{}).Error
}

// NodeList 获取本路线图的节点，和公共的节点
func (dao *GORMAdminDAO) NodeList(ctx context.Context, rid int64) ([]Node, error) {
	var nodes []Node
	err := dao.db.WithContext(ctx).
		// 获取当前路线图的节点，或者通用节点
		Where("rid = ? or rid = 0", rid).
		Order("id desc").
		Find(&nodes).Error
	return nodes, err
}

func (dao *GORMAdminDAO) SaveEdgeV1(ctx context.Context, edge EdgeV1) error {
	now := time.Now().UnixMilli()
	edge.Utime = now
	edge.Ctime = now
	return dao.db.WithContext(ctx).
		Clauses(
			clause.OnConflict{
				DoUpdates: clause.AssignmentColumns([]string{
					"src_node",
					"dst_node",
					"type",
					"attrs",
					"utime",
				}),
			},
		).
		Create(&edge).Error
}

func (dao *GORMAdminDAO) DeleteEdgeV1(ctx context.Context, id int64) error {
	return dao.db.WithContext(ctx).Where("id = ?", id).Delete(&EdgeV1{}).Error
}

func (dao *GORMAdminDAO) DeleteEdge(ctx context.Context, id int64) error {
	return dao.db.WithContext(ctx).Where("id = ?", id).Delete(&Edge{}).Error
}

func (dao *GORMAdminDAO) AddEdge(ctx context.Context, edge Edge) error {
	now := time.Now().UnixMilli()
	edge.Utime = now
	edge.Ctime = now
	return dao.db.WithContext(ctx).Create(&edge).Error
}

func (dao *GORMAdminDAO) GetEdgesByRid(ctx context.Context, rid int64) ([]Edge, error) {
	var res []Edge
	// 按照更新时间倒序排序
	err := dao.db.WithContext(ctx).Where("rid = ?", rid).Order("utime DESC").Find(&res).Error
	return res, err
}

func (dao *GORMAdminDAO) List(ctx context.Context, offset int, limit int) ([]Roadmap, error) {
	var res []Roadmap
	err := dao.db.WithContext(ctx).Order("id DESC").Offset(offset).Limit(limit).Find(&res).Error
	return res, err
}

func (dao *GORMAdminDAO) GetById(ctx context.Context, id int64) (Roadmap, error) {
	var r Roadmap
	err := dao.db.WithContext(ctx).Where("id = ?", id).First(&r).Error
	return r, err
}

func (dao *GORMAdminDAO) Save(ctx context.Context, r Roadmap) (int64, error) {
	now := time.Now().UnixMilli()
	r.Ctime = now
	r.Utime = now
	err := dao.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoUpdates: clause.AssignmentColumns([]string{"title", "biz", "biz_id", "utime"}),
		}).Create(&r).Error
	return r.Id, err
}

func NewGORMAdminDAO(db *egorm.Component) AdminDAO {
	return &GORMAdminDAO{
		db: db,
	}
}
