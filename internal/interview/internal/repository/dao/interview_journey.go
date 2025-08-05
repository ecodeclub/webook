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

	"github.com/ego-component/egorm"
)

// InterviewJourney 代表一个完整的面试旅程，从投递到最终结果
type InterviewJourney struct {
	ID          int64           `gorm:"type:BIGINT;primaryKey;autoIncrement;comment:'主键ID'"`
	Uid         int64           `gorm:"type:BIGINT;NOT NULL;index:idx_user_id;comment:'用户ID'"`
	CompanyID   sql.Null[int64] `gorm:"type:BIGINT;index:idx_company_id;comment:'关联的公司ID，可为空'"`
	CompanyName string          `gorm:"type:VARCHAR(255);NOT NULL;comment:'用户输入的公司名'"`
	JobInfo     string          `gorm:"type:TEXT;NOT NULL;comment:'用户输入的岗位信息（岗位名称+职责描述+任职要求）'"`
	ResumeURL   string          `gorm:"type:VARCHAR(255);NOT NULL;comment:'初始投递的简历在OSS中的存储URL'"`
	Stime       int64           `gorm:"type:BIGINT;NOT NULL;comment:'开始时间'"`
	Etime       int64           `gorm:"type:BIGINT;NOT NULL;default:0;comment:'结束时间'"`
	Status      string          `gorm:"type:ENUM('ACTIVE','SUCCEEDED','FAILED','ABANDONED');NOT NULL;default:'ACTIVE';comment:'面试历程状态'"`
	Ctime       int64
	Utime       int64
}

func (InterviewJourney) TableName() string {
	return "interview_journeys"
}

// InterviewJourneyDAO 定义面试历程的数据访问接口
type InterviewJourneyDAO interface {
	Create(ctx context.Context, journey InterviewJourney) (int64, error)
	Update(ctx context.Context, journey InterviewJourney) error
	First(ctx context.Context, id, uid int64) (InterviewJourney, error)
	FindByUID(ctx context.Context, uid int64, offset, limit int) ([]InterviewJourney, error)
	CountByUID(ctx context.Context, uid int64) (int64, error)
}

// GORMInterviewJourneyDAO 是 InterviewJourneyDAO 的GORM实现
type GORMInterviewJourneyDAO struct {
	db *egorm.Component
}

func NewGORMInterviewJourneyDAO(db *egorm.Component) InterviewJourneyDAO {
	return &GORMInterviewJourneyDAO{db: db}
}

func (g *GORMInterviewJourneyDAO) Create(ctx context.Context, journey InterviewJourney) (int64, error) {
	now := time.Now().UnixMilli()
	journey.Ctime, journey.Utime = now, now
	err := g.db.WithContext(ctx).Create(&journey).Error
	return journey.ID, err
}

func (g *GORMInterviewJourneyDAO) Update(ctx context.Context, journey InterviewJourney) error {
	return g.db.WithContext(ctx).Model(&journey).Where("id = ? AND uid = ?", journey.ID, journey.Uid).Updates(map[string]any{
		"company_id":   journey.CompanyID,
		"company_name": journey.CompanyName,
		"job_info":     journey.JobInfo,
		"resume_url":   journey.ResumeURL,
		"stime":        journey.Stime,
		"etime":        journey.Etime,
		"status":       journey.Status,
		"utime":        time.Now().UnixMilli(),
	}).Error
}

func (g *GORMInterviewJourneyDAO) First(ctx context.Context, id, uid int64) (InterviewJourney, error) {
	var journey InterviewJourney
	err := g.db.WithContext(ctx).Where("id = ? AND uid = ?", id, uid).First(&journey).Error
	return journey, err
}

func (g *GORMInterviewJourneyDAO) FindByUID(ctx context.Context, uid int64, offset, limit int) ([]InterviewJourney, error) {
	var journeys []InterviewJourney
	err := g.db.WithContext(ctx).Where("uid = ?", uid).
		Order("stime DESC").
		Offset(offset).
		Limit(limit).
		Find(&journeys).Error
	return journeys, err
}

func (g *GORMInterviewJourneyDAO) CountByUID(ctx context.Context, uid int64) (int64, error) {
	var count int64
	err := g.db.WithContext(ctx).Model(&InterviewJourney{}).Where("uid = ?", uid).Count(&count).Error
	return count, err
}
