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
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ego-component/egorm"
	"gorm.io/gorm"
)

var (
	ErrInvalidParentID = errors.New("父评论ID非法")
)

// Comment 表示针对某一资源的评论
type Comment struct {
	ID int64 `gorm:"autoIncrement,primaryKey;comment:'评论自增ID'"`

	Uid int64 `gorm:"not null;index;comment:'评论者'"`

	// 评论的对象
	Biz   string `gorm:"type:varchar(256);not null;index:biz_biz_id,priority:1;comment:'业务名称'"`
	BizID int64  `gorm:"type:bigint;not null;index:idx_biz_biz_id,priority:2;comment:'业务内唯一ID'"`

	Content string `gorm:"type:text;not null;comment:'评论的具体内容'"`

	// 这两个字段都可以为 NULL。如果是 NULL 就代表它自身就是一个根评论
	AncestorID sql.Null[int64] `gorm:"type:bigint;index:idx_ancestor_id;comment:'始祖评论ID，0表示对业务资源的直接评论，因其引发的后续所有评论都是其后裔（非0）。'"`
	ParentID   sql.Null[int64] `gorm:"type:bigint;index:idx_parent_id;comment:'父评论ID，0表示对业务资源的直接评论，非0表示要回复的评论的ID'"`

	// 外键用于级联删除后裔评论（子评论、子孙评论）
	ParentComment *Comment `gorm:"ForeignKey:ParentID;AssociationForeignKey:ID;constraint:OnDelete:CASCADE"`

	Utime int64
	Ctime int64
}

func (Comment) TableName() string {
	return "comments"
}

type CommentDAO interface {
	// Create 创建直接评论（始祖评论），子评论及孙子评论
	Create(ctx context.Context, comment Comment) (int64, error)
	// FindAncestors 查找某一业务下的所有直接评论（始祖评论），按评论时间的倒序排序
	FindAncestors(ctx context.Context, biz string, bizID, minID int64, limit int) ([]Comment, error)
	// FindChildren 查找子评论
	FindChildren(ctx context.Context, parentID int64, limit int) ([]Comment, error)
	// CountAncestors 统计某一业务下所有直接评论（始祖评论）的数量
	CountAncestors(ctx context.Context, biz string, bizID int64) (int64, error)
	// FindDescendants 查找直接评论（始祖评论）所有后代即所有子评论，孙子评论，按照评论时间排序（即先评论的在前面）
	FindDescendants(ctx context.Context, ancestorID, maxID int64, limit int) ([]Comment, error)
	// CountDescendants 统计直接评论（始祖评论）所有后代即所有子评论，孙子评论的数量
	CountDescendants(ctx context.Context, ancestorID int64) (int64, error)
	// FindByID 根据评论ID查找评论
	FindByID(ctx context.Context, id int64) (Comment, error)
	// Delete 根据ID删除评论及其后裔评论
	Delete(ctx context.Context, id int64) error
}

type commentDAO struct {
	db *egorm.Component
}

func NewCommentGORMDAO(db *egorm.Component) CommentDAO {
	return &commentDAO{db: db}
}

func (g *commentDAO) Create(ctx context.Context, c Comment) (int64, error) {
	now := time.Now().UnixMilli()
	c.Ctime, c.Utime = now, now
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ancestorID int64
		// 当前评论非根评论（始祖评论）
		if c.ParentID.Valid {
			// 找到父评论
			var parent Comment
			if err := tx.First(&parent, "id = ?", c.ParentID.V).Error; err != nil {
				return fmt.Errorf("%w: %w", ErrInvalidParentID, err)
			}
			// 如果父评论是根评论（始祖评论），那始祖评论ID就是父评论ID，否则与父评论的始祖评论ID相同
			if !parent.ParentID.Valid {
				ancestorID = parent.ID
			} else {
				ancestorID = parent.AncestorID.V
			}
		}
		c.AncestorID = sql.Null[int64]{V: ancestorID, Valid: ancestorID != 0}
		if err := tx.Create(&c).Error; err != nil {
			return err
		}
		return nil
	})
	return c.ID, err
}

func (g *commentDAO) FindAncestors(ctx context.Context, biz string, bizID, minID int64, limit int) ([]Comment, error) {
	var res []Comment
	err := g.db.WithContext(ctx).
		Where("id < ? AND biz = ? AND biz_id = ?", minID, biz, bizID).
		// 直接评论、根评论、始祖评论
		Where("ancestor_id IS NULL AND parent_id IS NULL").
		Order("id DESC").
		Limit(limit).
		Find(&res).Error
	return res, err
}

func (g *commentDAO) FindChildren(ctx context.Context, parentID int64, limit int) ([]Comment, error) {
	var res []Comment
	err := g.db.WithContext(ctx).
		Where("parent_id", parentID).
		Order("id ASC").
		Limit(limit).
		Find(&res).Error
	return res, err
}

func (g *commentDAO) CountAncestors(ctx context.Context, biz string, bizID int64) (int64, error) {
	var count int64
	err := g.db.WithContext(ctx).Model(&Comment{}).
		Where("biz = ? AND biz_id = ?", biz, bizID).
		Where("ancestor_id IS NULL AND parent_id IS NULL").
		Count(&count).Error
	return count, err
}

func (g *commentDAO) FindDescendants(ctx context.Context, ancestorID, maxID int64, limit int) ([]Comment, error) {
	var res []Comment
	err := g.db.WithContext(ctx).
		Where("id > ? AND ancestor_id = ?", maxID, ancestorID).
		Order("id ASC").
		Limit(limit).
		Find(&res).Error
	return res, err
}

func (g *commentDAO) CountDescendants(ctx context.Context, ancestorID int64) (int64, error) {
	var count int64
	err := g.db.WithContext(ctx).Model(&Comment{}).
		Where("ancestor_id = ?", ancestorID).
		Count(&count).Error
	return count, err
}

func (g *commentDAO) FindByID(ctx context.Context, id int64) (Comment, error) {
	var c Comment
	err := g.db.WithContext(ctx).First(&c, id).Error
	return c, err
}

func (g *commentDAO) Delete(ctx context.Context, id int64) error {
	return g.db.WithContext(ctx).Where("id = ?", id).Delete(&Comment{}).Error
}
