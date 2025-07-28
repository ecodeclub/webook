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

import "github.com/ecodeclub/ginx"

type CommentRequest struct {
	Comment Comment `json:"comment"`
}

type User struct {
	// 创建评论的时候只用id
	ID int64 `json:"id"`
	// 查询的时候要带上下面的冗余信息
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type Comment struct {
	ID int64 `json:"id"`

	// 评论的人
	User User `json:"user"`

	// 针对什么东西的评论
	// 注意，即便是回复某个评论，那么这两个字段依旧有值
	Biz   string `json:"biz"`
	BizID int64  `json:"bizId"`

	// 回复某个评论
	ParentID int64 `json:"parentId"`

	// 评论的具体内容
	Content string `json:"content"`

	Utime int64 `json:"utime"`

	// 当前评论的回复
	Replies []Comment `json:"replies"`
}

type CommentListRequest struct {
	Biz   string `json:"biz"`
	BizID int64  `json:"bizId"`

	// 上一页最小的评论ID，如果是第一页就传0
	MinID int64 `json:"minId"`
	Limit int   `json:"limit"`

	// 顺带返回的最大子评论数
	MaxSubCnt int `json:"maxSubCnt"`
}

type GetRepliesRequest struct {
	// 直接评论、根评论、始祖评论 ID
	AncestorID int64 `json:"ancestorId"`
	// 上一页最大的评论ID，如果是第一页就传0
	MaxID int64 `json:"maxId"`
	Limit int   `json:"limit"`
}

type CommentList ginx.DataList[Comment]
