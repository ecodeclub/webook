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

import "database/sql"

// Roadmap 后续要考虑引入制作库，线上库
type Roadmap struct {
	Id    int64 `gorm:"primaryKey,autoIncrement"`
	Title string
	// 关联的 ID，不一定都有
	// 例如说这个是专门给题集用的，这里就是代表题集的 ID
	// 唯一索引确保一个业务不会有两个业务图
	BizId sql.NullInt64  `gorm:"uniqueIndex:biz"`
	Biz   sql.NullString `gorm:"type:varchar(128);uniqueIndex:biz"`

	Ctime int64
	Utime int64
}

func (r Roadmap) TableName() string {
	return "roadmaps"
}

type Edge struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`

	// 理论上来说 Edge 中的 Rid, Src, Dst 构成一个唯一索引。
	// 但是因为都是内部在操作，所以没太大必要真的建立这个唯一索引
	// Roadmap 的 ID
	Rid int64 `gorm:"index"`

	// 源头
	SrcId  int64  `gorm:"index:src"`
	SrcBiz string `gorm:"type:varchar(128);index:src"`

	// 目标
	DstId  int64  `gorm:"index:dst"`
	DstBiz string `gorm:"type:varchar(128);index:dst"`

	Utime int64
	Ctime int64
}

func (e Edge) TableName() string {
	return "roadmap_edges"
}

type EdgeV1 struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`

	// 理论上来说 Edge 中的 Rid, Src, Dst 构成一个唯一索引。
	// 但是因为都是内部在操作，所以没太大必要真的建立这个唯一索引
	// Roadmap 的 ID
	Rid int64 `gorm:"index"`

	// 源头
	SrcNode int64 `gorm:"index:src_node"`

	// 目标
	DstNode int64 `gorm:"index:dst_node"`

	Type  string
	Attrs string

	Utime int64
	Ctime int64
}
type Node struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	// plainText, link
	Biz string

	// 关联id
	RefId int64
	Attrs string
	Rid   int64 `gorm:"index"`
	Utime int64
	Ctime int64
}

func (e Node) TableName() string {
	return "roadmap_nodes"
}
func (e EdgeV1) TableName() string {
	return "roadmap_edges_v1"
}
