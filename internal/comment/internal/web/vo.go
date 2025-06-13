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

type CommentRequest struct {
	Comment Comment `json:"comment"`
}

type User struct {
	ID       int64
	Nickname string
	Avatar   string
}
type Comment struct {
	ID int64
	// 回复某个评论
	ParentID int64 `json:"parentID"`

	// 评论的具体内容
	Content string `json:"content"`

	// 评论的人
	User User `json:"user"`

	// 针对什么东西的评论
	// 注意，即便是回复某个评论，那么这两个字段依旧有值
	Biz   string `json:"biz"`
	BizID int64  `json:"bizID"`
	Utime int64  `json:"utime"`
}

type GetRepliesRequest struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
	// 根评论 ID
	Ancestor int64 `json:"ancestor"`
}

type CommentListRequest struct {
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
	Biz    string `json:"biz"`
	BizID  int64  `json:"bizID"`
}

type CommentList struct {
	Comments []Comment `json:"comments"`
	Total    int       `json:"total"`
}
