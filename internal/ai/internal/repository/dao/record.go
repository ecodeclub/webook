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
	"time"

	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ego-component/egorm"
	"gorm.io/gorm/clause"
)

type GPTRecordDAO interface {
	Save(ctx context.Context, r GPTRecord) (int64, error)
}

type GORMGPTLogDAO struct {
	db *egorm.Component
}

func NewGORMGPTLogDAO(db *egorm.Component) GPTRecordDAO {
	return &GORMGPTLogDAO{db: db}
}

func (g *GORMGPTLogDAO) Save(ctx context.Context, record GPTRecord) (int64, error) {
	now := time.Now().UnixMilli()
	record.Ctime = now
	record.Utime = now
	err := g.db.WithContext(ctx).Model(&GPTRecord{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "utime"}),
		}).Create(&record).Error
	return record.Id, err
}

func (g *GORMGPTLogDAO) FirstLog(ctx context.Context, id int64) (*GPTRecord, error) {
	logModel := &GPTRecord{}
	err := g.db.WithContext(ctx).Model(&GPTRecord{}).Where("id = ?", id).First(logModel).Error
	return logModel, err
}

type GPTRecord struct {
	Id             int64                     `gorm:"primaryKey;autoIncrement;comment:积分流水表自增ID"`
	Tid            string                    `gorm:"type:varchar(256);not null;uniqueIndex:unq_tid;comment:一次请求的Tid只能有一次"`
	Uid            int64                     `gorm:"not null;index:idx_user_id;comment:用户ID"`
	Biz            string                    `gorm:"type:varchar(256);not null;comment:业务类型名"`
	Tokens         int64                     `gorm:"type:int;default:0;comment:扣费token数"`
	Amount         int64                     `gorm:"type:int;default:0;comment:具体扣费的换算的钱，分为单位"`
	Status         uint8                     `gorm:"type:tinyint unsigned;not null;default:1;comment:调用状态 1=成功, 2=失败"`
	Input          sqlx.JsonColumn[[]string] `gorm:"type:text;comment:调用请求的参数"`
	KnowledgeId    string                    `gorm:"type:varchar(256);not null;comment:使用的知识库 ID"`
	PromptTemplate sql.NullString            `gorm:"type:text;comment:PromptTemplate 模板，加上请求参数构成一个完整的 prompt"`
	Answer         sql.NullString            `gorm:"type:text;comment:gpt的回答"`
	Ctime          int64
	Utime          int64
}

func (l GPTRecord) TableName() string {
	return "gpt_records"
}
