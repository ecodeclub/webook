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

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/comment/internal/domain"
	"github.com/ecodeclub/webook/internal/comment/internal/repository/dao"
	"golang.org/x/sync/errgroup"
)

type CommentRepository interface {
	// Create 创建直接评论（始祖评论），子评论及孙子评论
	Create(ctx context.Context, comment domain.Comment) (int64, error)
	// FindAncestors 查找某一业务下的所有直接评论（始祖评论）评论时间的倒序
	FindAncestors(ctx context.Context, biz string, bizID, minID int64, limit, maxSubCnt int) ([]domain.Comment, error)
	// CountAncestors 统计某一业务下所有直接评论（始祖评论）的数量
	CountAncestors(ctx context.Context, biz string, bizID int64) (int64, error)
	// FindDescendants 查找直接评论（始祖评论）所有后代即所有子评论，孙子评论，按照评论时间排序（即先评论的在前面）
	FindDescendants(ctx context.Context, ancestorID, maxID int64, limit int) ([]domain.Comment, error)
	// CountDescendants 统计直接评论（始祖评论）所有后代即所有子评论，孙子评论的数量
	CountDescendants(ctx context.Context, ancestorID int64) (int64, error)
}

type commentRepository struct {
	dao dao.CommentDAO
}

func NewCommentRepository(dao dao.CommentDAO) CommentRepository {
	return &commentRepository{dao: dao}
}

func (r *commentRepository) Create(ctx context.Context, comment domain.Comment) (int64, error) {
	return r.dao.Create(ctx, r.toEntity(comment))
}

func (r *commentRepository) toEntity(comment domain.Comment) dao.Comment {
	return dao.Comment{
		ID:       comment.ID,
		Uid:      comment.User.ID,
		Biz:      comment.Biz,
		BizID:    comment.BizID,
		ParentID: comment.ParentID,
		Content:  comment.Content,
	}
}

func (r *commentRepository) toDomain(comment dao.Comment) domain.Comment {
	return domain.Comment{
		ID: comment.ID,
		User: domain.User{
			ID: comment.Uid,
		},
		Biz:      comment.Biz,
		BizID:    comment.BizID,
		ParentID: comment.ParentID,
		Content:  comment.Content,
		Utime:    comment.Utime,
	}
}

func (r *commentRepository) FindAncestors(ctx context.Context, biz string, bizID, minID int64, limit, maxSubCnt int) ([]domain.Comment, error) {
	ancestors, err := r.dao.FindAncestors(ctx, biz, bizID, minID, limit)
	if err != nil {
		return nil, err
	}
	comments := slice.Map(ancestors, func(_ int, src dao.Comment) domain.Comment {
		return r.toDomain(src)
	})
	// 并发获取回复
	var eg errgroup.Group
	for i := range comments {
		eg.Go(func() error {
			children, err1 := r.dao.FindChildren(ctx, comments[i].ID, maxSubCnt)
			if err1 != nil {
				return err1
			}
			comments[i].Replies = slice.Map(children, func(_ int, src dao.Comment) domain.Comment {
				return r.toDomain(src)
			})
			return nil
		})
	}
	return comments, eg.Wait()
}

func (r *commentRepository) CountAncestors(ctx context.Context, biz string, bizID int64) (int64, error) {
	return r.dao.CountAncestors(ctx, biz, bizID)
}

func (r *commentRepository) FindDescendants(ctx context.Context, ancestorID, maxID int64, limit int) ([]domain.Comment, error) {
	found, err := r.dao.FindDescendants(ctx, ancestorID, maxID, limit)
	if err != nil {
		return nil, err
	}
	// 后裔评论不需要填充replies前端后组装。
	return slice.Map(found, func(_ int, src dao.Comment) domain.Comment {
		return r.toDomain(src)
	}), nil
}

func (r *commentRepository) CountDescendants(ctx context.Context, ancestorID int64) (int64, error) {
	return r.dao.CountDescendants(ctx, ancestorID)
}
