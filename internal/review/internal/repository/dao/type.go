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

import "github.com/ecodeclub/ekit/sqlx"

type Review struct {
	ID    int64  `gorm:"primaryKey;autoIncrement;column:id"`
	Uid   int64  `gorm:"column:uid"`
	Title string `gorm:"type=varchar(512)"`
	// 面试题目内容
	Desc    string
	Labels  sqlx.JsonColumn[[]string] `gorm:"type:varchar(512)"`
	JD      string                    `gorm:"column:jd;type:text"`
	Content string                    `gorm:"column:content;type:text;not null;comment:'markdown 语法'"`
	Resume  string                    `gorm:"column:resume;type:text"`
	Status  uint8                     `gorm:"type:tinyint(3);comment:0-未知 1-未发表 2-已发表"`
	Cid     int64                     `gorm:"column:cid"`
	Ctime   int64                     `gorm:"column:ctime"`
	Utime   int64                     `gorm:"column:utime"`
}

type PublishReview Review
