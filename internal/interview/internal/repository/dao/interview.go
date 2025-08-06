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
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// InterviewRound 代表面试旅程中的一个具体轮次（如：一面、二面、HR面）
type InterviewRound struct {
	ID            int64  `gorm:"type:BIGINT;primaryKey;autoIncrement;comment:'主键ID'"`
	Uid           int64  `gorm:"type:BIGINT;NOT NULL;index:idx_user_id;comment:'用户ID'"`
	Jid           int64  `gorm:"type:BIGINT;NOT NULL;index:idx_journey_id;unique:unq_journey_id_round_number,priority:1;comment:'所属面试历程ID'"`
	RoundNumber   int    `gorm:"type:INT;NOT NULL;default:1;unique:unq_journey_id_round_number,priority:2;comment:'轮数编号，例如 1, 2, 3'"`
	RoundType     string `gorm:"type:VARCHAR(255);comment:'轮数类型，允许为NULL，可以填入例如——同事、虚线leader、leader、manager、CTO、CEO、HR'"`
	InterviewDate int64  `gorm:"NOT NULL;comment:'面试时间'"`
	JobInfo       string `gorm:"type:TEXT;NOT NULL;comment:'本轮实际面试的岗位信息（岗位名称+职责描述+任职要求）'"`
	ResumeURL     string `gorm:"type:VARCHAR(255);NOT NULL;comment:'本轮投递的简历在OSS中的存储URL'"`
	AudioURL      string `gorm:"type:VARCHAR(1024);comment:'本轮面试录音在OSS中的存储URL'"`
	SelfResult    bool   `gorm:"type:BOOLEAN;NOT NULL;comment:'自我评估结果：true->已通过, false->未通过'"`
	SelfSummary   string `gorm:"type:TEXT;comment:'自我复盘总结'"`
	Result        string `gorm:"type:ENUM('PENDING','APPROVED','REJECTED');NOT NULL;comment:'官方结果：等待中、已通过、未通过'"`
	AllowSharing  bool   `gorm:"type:BOOLEAN;NOT NULL;default:false;comment:'授权公开本轮面试信息'"`

	Ctime int64
	Utime int64
}

func (InterviewRound) TableName() string {
	return "interview_rounds"
}

// InterviewDAO 定义面试历程的数据访问接口
type InterviewDAO interface {
	Save(ctx context.Context, journey InterviewJourney, rounds []InterviewRound) (int64, error)
	Find(ctx context.Context, id, uid int64) (InterviewJourney, []InterviewRound, error)

	FindJourneysByUID(ctx context.Context, uid int64, offset, limit int) ([]InterviewJourney, error)
	CountJourneyByUID(ctx context.Context, uid int64) (int64, error)

	FindRoundByID(ctx context.Context, id, jid, uid int64) (InterviewRound, error)
	FindRoundsByJidAndUid(ctx context.Context, jid, uid int64) ([]InterviewRound, error)
}

// GORMInterviewDAO 是 InterviewDAO 的GORM实现
type GORMInterviewDAO struct {
	db *egorm.Component
}

func NewGORMInterviewDAO(db *egorm.Component) InterviewDAO {
	return &GORMInterviewDAO{db: db}
}

func (g *GORMInterviewDAO) Save(ctx context.Context, journey InterviewJourney, rounds []InterviewRound) (int64, error) {
	// 为 journey 和 rounds 统一设置时间
	now := time.Now().UnixMilli()
	journey.Utime = now
	if journey.ID == 0 {
		journey.Ctime = now
	}
	for i := range rounds {
		rounds[i].Utime = now
		if rounds[i].ID == 0 {
			rounds[i].Ctime = now
		}
	}

	var jid int64
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 步骤1: 保存 Journey 主体，使用 Clauses(OnConflict)
		if err := tx.Clauses(clause.OnConflict{
			// 冲突目标是主键 id
			Columns: []clause.Column{{Name: "id"}},
			// 冲突时，更新所有可变字段
			DoUpdates: clause.AssignmentColumns([]string{
				"company_id",
				"company_name",
				"job_info",
				"resume_url",
				"stime",
				"etime",
				"status",
				"utime",
			}),
		}).Create(&journey).Error; err != nil {
			return err
		}
		jid = journey.ID

		if len(rounds) == 0 {
			return nil
		}

		// 步骤2: 遍历并保存所有 Round，逻辑保持一致
		for i := range rounds {
			rounds[i].Jid = jid // 确保关联ID正确
		}
		return tx.Clauses(clause.OnConflict{
			// 冲突的目标是联合唯一索引
			Columns: []clause.Column{{Name: "jid"}, {Name: "round_number"}},
			// 在冲突时，需要更新除了唯一键和创建时间之外的所有字段
			DoUpdates: clause.AssignmentColumns([]string{
				"round_type",
				"interview_date",
				"job_info",
				"resume_url",
				"audio_url",
				"self_result",
				"self_summary",
				"result",
				"allow_sharing",
				"utime",
			}),
		}).Create(&rounds).Error
	})
	return jid, err
}

func (g *GORMInterviewDAO) Find(ctx context.Context, id, uid int64) (InterviewJourney, []InterviewRound, error) {
	var journey InterviewJourney
	var rounds []InterviewRound
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND uid = ?", id, uid).First(&journey).Error; err != nil {
			return err
		}
		return tx.Where("jid = ? AND uid = ?", id, uid).Order("round_number ASC").Find(&rounds).Error
	})
	return journey, rounds, err
}

func (g *GORMInterviewDAO) FindJourneysByUID(ctx context.Context, uid int64, offset, limit int) ([]InterviewJourney, error) {
	var journeys []InterviewJourney
	err := g.db.WithContext(ctx).Where("uid = ?", uid).
		Order("utime DESC").
		Offset(offset).
		Limit(limit).
		Find(&journeys).Error
	return journeys, err
}

func (g *GORMInterviewDAO) CountJourneyByUID(ctx context.Context, uid int64) (int64, error) {
	var count int64
	err := g.db.WithContext(ctx).Model(&InterviewJourney{}).Where("uid = ?", uid).Count(&count).Error
	return count, err
}

func (g *GORMInterviewDAO) FindRoundByID(ctx context.Context, id, jid, uid int64) (InterviewRound, error) {
	var round InterviewRound
	err := g.db.WithContext(ctx).Where("id = ? AND jid = ? AND uid = ?", id, jid, uid).First(&round).Error
	return round, err
}

func (g *GORMInterviewDAO) FindRoundsByJidAndUid(ctx context.Context, jid, uid int64) ([]InterviewRound, error) {
	var rounds []InterviewRound
	err := g.db.WithContext(ctx).Where("jid = ? AND uid = ?", jid, uid).Order("round_number ASC").Find(&rounds).Error
	return rounds, err
}
