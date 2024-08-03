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

// Interactive 汇总表
type Interactive struct {
	Id         int64  `gorm:"primaryKey,autoIncrement"`
	BizId      int64  `gorm:"uniqueIndex:biz_type_id"`
	Biz        string `gorm:"type:varchar(128);uniqueIndex:biz_type_id"`
	ViewCnt    int
	LikeCnt    int
	CollectCnt int
	Utime      int64
	Ctime      int64
}

// UserLikeBiz 点赞明细表
type UserLikeBiz struct {
	Id    int64  `gorm:"primaryKey,autoIncrement"`
	Uid   int64  `gorm:"uniqueIndex:uid_biz_type_id"`
	BizId int64  `gorm:"uniqueIndex:uid_biz_type_id"`
	Biz   string `gorm:"type:varchar(128);uniqueIndex:uid_biz_type_id"`
	Utime int64
	Ctime int64
}

// UserCollectionBiz 收藏明细表
type UserCollectionBiz struct {
	Id    int64  `gorm:"primaryKey,autoIncrement"`
	Uid   int64  `gorm:"uniqueIndex:uid_biz_type_id"`
	BizId int64  `gorm:"uniqueIndex:uid_biz_type_id"`
	Biz   string `gorm:"type:varchar(128);uniqueIndex:uid_biz_type_id"`

	// Cid 收藏夹id
	Cid   sql.NullInt64 `gorm:"index"`
	Utime int64
	Ctime int64
}

type Collection struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	// 在 Uid 和 Name 上创建唯一索引，确保用户不会创建同名收藏夹
	Uid   int64  `gorm:"uniqueIndex:uid_name"`
	Name  string `gorm:"type:varchar(256);uniqueIndex:uid_name"`
	Utime int64
	Ctime int64
}
