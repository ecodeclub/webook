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

	"github.com/ego-component/egorm"
)

type ConfigDAO interface {
	GetConfig(ctx context.Context, biz string) (BizConfig, error)
}

type GORMConfigDAO struct {
	db *egorm.Component
}

func NewGORMConfigDAO(db *egorm.Component) ConfigDAO {
	return &GORMConfigDAO{db: db}
}

func (dao *GORMConfigDAO) GetConfig(ctx context.Context, biz string) (BizConfig, error) {
	var res BizConfig
	err := dao.db.WithContext(ctx).Where("biz = ?", biz).First(&res).Error
	return res, err
}

type BizConfig struct {
	Id          int64  `gorm:"primaryKey;autoIncrement;comment:AI biz 配置表ID"`
	Biz         string `gorm:"type:varchar(256);uniqueIndex;not null;comment:业务类型名"`
	MaxInput    int    `gorm:"comment:最大输入长度"`
	Model       string `gorm:"type:varchar(256)"`
	Price       int64
	Temperature float64
	TopP        float64
	// 系统 prompt
	SystemPrompt   string
	PromptTemplate string
	KnowledgeId    string `gorm:"type:varchar(256);not null;comment:使用的知识库 ID"`
	// 其它字段按需添加
	Ctime int64
	Utime int64
}

func (c BizConfig) TableName() string {
	return "ai_biz_configs"
}
