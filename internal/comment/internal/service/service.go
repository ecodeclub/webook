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

package service

import (
	"context"
	"math"

	"github.com/ecodeclub/webook/internal/comment/internal/domain"
	"github.com/ecodeclub/webook/internal/comment/internal/repository"
	"github.com/ecodeclub/webook/internal/user"
	"golang.org/x/sync/errgroup"
)

type CommentService interface {
	// Create  创建直接评论（始祖评论），子评论及孙子评论
	Create(ctx context.Context, comment domain.Comment) (int64, error)
	// List 查找某一业务下的所有直接评论（始祖评论），按评论时间的倒序排序
	List(ctx context.Context, biz string, bizID, minID int64, limit int) ([]domain.Comment, int64, error)
	// Replies 查找直接评论（始祖评论）所有后代即所有子评论，孙子评论，按照评论时间倒序排序（即后评论的在前面）
	Replies(ctx context.Context, ancestorID, minID int64, limit int) ([]domain.Comment, int64, error)
	// Delete 根据ID删除评论及其后裔评论
	Delete(ctx context.Context, id, uid int64) error
}

type commentService struct {
	userSvc user.UserService
	repo    repository.CommentRepository
}

func NewCommentService(userSvc user.UserService, repo repository.CommentRepository) CommentService {
	return &commentService{userSvc: userSvc, repo: repo}
}

func (s *commentService) Create(ctx context.Context, comment domain.Comment) (int64, error) {
	return s.repo.Create(ctx, comment)
}

func (s *commentService) List(ctx context.Context, biz string, bizID, minID int64, limit int) ([]domain.Comment, int64, error) {
	var (
		eg       errgroup.Group
		comments []domain.Comment
		total    int64
	)

	if minID <= 0 {
		minID = math.MaxInt64
	}

	eg.Go(func() error {
		var err error
		comments, err = s.repo.FindAncestors(ctx, biz, bizID, minID, limit)
		if err != nil {
			return err
		}
		return s.setUserInfo(ctx, comments)
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.CountAncestors(ctx, biz, bizID)
		return err
	})

	return comments, total, eg.Wait()
}

func (s *commentService) setUserInfo(ctx context.Context, comments []domain.Comment) error {
	if len(comments) == 0 {
		return nil
	}

	// 获取用户id集合
	uids := make([]int64, 0, len(comments)*2)
	for i := range comments {
		uids = append(uids, comments[i].User.ID)
	}

	// 批量查询用户信息
	profiles, err := s.userSvc.BatchProfile(ctx, uids)
	if err != nil {
		return err
	}
	// 构建映射
	userInfoMap := make(map[int64]domain.User, len(profiles))
	for _, p := range profiles {
		userInfoMap[p.Id] = domain.User{
			ID:       p.Id,
			NickName: p.Nickname,
			Avatar:   p.Avatar,
		}
	}
	// 直接覆盖用户信息
	for i := range comments {
		if u, ok := userInfoMap[comments[i].User.ID]; ok {
			comments[i].User = u
		}
	}
	return nil
}

func (s *commentService) Replies(ctx context.Context, ancestorID, minID int64, limit int) ([]domain.Comment, int64, error) {
	var (
		eg      errgroup.Group
		replies []domain.Comment
		total   int64
	)

	if minID <= 0 {
		minID = math.MaxInt64
	}

	eg.Go(func() error {
		var err error
		replies, err = s.repo.FindDescendants(ctx, ancestorID, minID, limit)
		if err != nil {
			return err
		}
		return s.setUserInfo(ctx, replies)
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.CountDescendants(ctx, ancestorID)
		return err
	})

	return replies, total, eg.Wait()
}

func (s *commentService) Delete(ctx context.Context, id, uid int64) error {
	return s.repo.Delete(ctx, id, uid)
}
