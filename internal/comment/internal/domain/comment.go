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

package domain

type User struct {
	ID       int64
	NickName string
	Avatar   string
}

type Comment struct {
	ID int64
	// 评论的人
	User User
	// 评论的对象
	Biz   string
	BizID int64

	// 当前评论要回复的父评论ID
	ParentID int64

	// 评论的具体内容
	Content string

	// 评论时间，因为评论本身是不允许修改的，所以这个时间基本上就是评论时间
	Utime int64

	// 当前评论的回复，只有再查询”始祖评论“的时候带上部分子回复。
	Replies []Comment
}
