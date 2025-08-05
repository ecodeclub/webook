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
	"time"

	"github.com/ego-component/egorm"
)

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

// InterviewRoundDAO 定义面试轮次的数据访问接口
type InterviewRoundDAO interface {
	Create(ctx context.Context, round InterviewRound) (int64, error)
	Update(ctx context.Context, round InterviewRound) error
	First(ctx context.Context, id, jid, uid int64) (InterviewRound, error)
	FindByJidAndUid(ctx context.Context, jid, uid int64) ([]InterviewRound, error)
}

type GORMInterviewRoundDAO struct {
	db *egorm.Component
}

func NewGORMInterviewRoundDAO(db *egorm.Component) InterviewRoundDAO {
	return &GORMInterviewRoundDAO{db: db}
}

func (g *GORMInterviewRoundDAO) Create(ctx context.Context, round InterviewRound) (int64, error) {
	now := time.Now().UnixMilli()
	round.Ctime, round.Utime = now, now
	err := g.db.WithContext(ctx).Create(&round).Error
	return round.ID, err
}

func (g *GORMInterviewRoundDAO) Update(ctx context.Context, round InterviewRound) error {
	return g.db.WithContext(ctx).Model(&round).
		Where("id = ? AND jid = ? AND uid = ?", round.ID, round.Jid, round.Uid).
		Updates(map[string]any{
			"round_number":   round.RoundNumber,
			"round_type":     round.RoundType,
			"interview_date": round.InterviewDate,
			"job_info":       round.JobInfo,
			"resume_url":     round.ResumeURL,
			"audio_url":      round.AudioURL,
			"self_result":    round.SelfResult,
			"self_summary":   round.SelfSummary,
			"result":         round.Result,
			"allow_sharing":  round.AllowSharing,
			"utime":          time.Now().UnixMilli(),
		}).Error
}

func (g *GORMInterviewRoundDAO) First(ctx context.Context, id, jid, uid int64) (InterviewRound, error) {
	var round InterviewRound
	err := g.db.WithContext(ctx).Where("id = ? AND jid = ? AND uid = ?", id, jid, uid).First(&round).Error
	return round, err
}

func (g *GORMInterviewRoundDAO) FindByJidAndUid(ctx context.Context, jid, uid int64) ([]InterviewRound, error) {
	var rounds []InterviewRound
	err := g.db.WithContext(ctx).Where("jid = ? AND uid = ?", jid, uid).Order("round_number ASC").Find(&rounds).Error
	return rounds, err
}
