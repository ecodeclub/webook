package dao

import (
	"context"
	"database/sql"
	"time"

	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ego-component/egorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MockInterviewDAO interface {
	SaveInterview(ctx context.Context, mi MockInterview) (int64, error)
	FindInterviews(ctx context.Context, uid int64, limit, offset int) ([]MockInterview, error)
	CountInterviews(ctx context.Context, uid int64) (int64, error)

	SaveQuestion(ctx context.Context, q MockInterviewQuestion) (int64, error)
	FindQuestions(ctx context.Context, interviewID, uid int64, limit, offset int) ([]MockInterviewQuestion, error)
	CountQuestions(ctx context.Context, interviewID int64, uid int64) (int64, error)
}

type GORMMockInterviewDAO struct {
	db *egorm.Component
}

func NewMockInterviewDAO(db *egorm.Component) MockInterviewDAO {
	return &GORMMockInterviewDAO{
		db: db,
	}
}

func (d *GORMMockInterviewDAO) SaveInterview(ctx context.Context, mi MockInterview) (int64, error) {
	now := time.Now().UnixMilli()
	mi.Ctime = now
	mi.Utime = now
	err := d.db.WithContext(ctx).Model(&MockInterview{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "chat_sn"}},
			DoUpdates: clause.AssignmentColumns([]string{"uid", "title", "evaluation", "utime"}),
		}).
		Create(&mi).Error
	return mi.ID, err
}

func (d *GORMMockInterviewDAO) FindInterviews(ctx context.Context, uid int64, limit, offset int) ([]MockInterview, error) {
	var res []MockInterview
	db := d.db.WithContext(ctx).Model(&MockInterview{})
	if uid != 0 {
		db = db.Where("uid = ?", uid)
	}
	err := db.Order("ctime DESC").Limit(limit).Offset(offset).Find(&res).Error
	return res, err
}

func (d *GORMMockInterviewDAO) CountInterviews(ctx context.Context, uid int64) (int64, error) {
	var count int64
	db := d.db.WithContext(ctx).Model(&MockInterview{})
	if uid != 0 {
		db = db.Where("uid = ?", uid)
	}
	err := db.Count(&count).Error
	return count, err
}

func (d *GORMMockInterviewDAO) SaveQuestion(ctx context.Context, q MockInterviewQuestion) (int64, error) {
	now := time.Now().UnixMilli()
	var retID int64
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var mi MockInterview
		if err := tx.Where("chat_sn = ?", q.ChatSN).First(&mi).Error; err != nil {
			return err
		}
		q.InterviewID = mi.ID
		q.Ctime = now
		q.Utime = now
		if err := tx.Create(&q).Error; err != nil {
			return err
		}
		retID = q.ID
		return nil
	})
	return retID, err
}

func (d *GORMMockInterviewDAO) FindQuestions(ctx context.Context, interviewID, uid int64, limit, offset int) ([]MockInterviewQuestion, error) {
	var res []MockInterviewQuestion
	db := d.db.WithContext(ctx).Model(&MockInterviewQuestion{})
	if uid != 0 {
		db = db.Where("interview_id = ? AND uid = ?", interviewID, uid)
	} else {
		db = db.Where("interview_id = ?", interviewID)
	}

	err := db.Order("ctime DESC").Limit(limit).Offset(offset).Find(&res).Error
	return res, err
}

func (d *GORMMockInterviewDAO) CountQuestions(ctx context.Context, interviewID int64, uid int64) (int64, error) {
	var count int64
	db := d.db.WithContext(ctx).Model(&MockInterviewQuestion{})
	if uid != 0 {
		db = db.Where("interview_id = ? AND uid = ?", interviewID, uid)
	} else {
		db = db.Where("interview_id = ?", interviewID)
	}
	err := db.Count(&count).Error
	return count, err
}

type MockInterview struct {
	ID         int64                           `gorm:"primaryKey;autoIncrement;comment:自增ID"`
	ChatSN     string                          `gorm:"type:varchar(255);not null;uniqueIndex:uk_chat_sn;comment:外部会话SN"`
	Uid        int64                           `gorm:"not null;index:idx_uid;comment:用户UID"`
	Title      string                          `gorm:"type:varchar(255);not null;comment:面试名称"`
	Evaluation sqlx.JsonColumn[map[string]any] `gorm:"type:json;comment:本场面试总体评价JSON"`
	Ctime      int64                           `gorm:"not null;comment:创建时间"`
	Utime      int64                           `gorm:"not null;comment:更新时间"`
}

func (MockInterview) TableName() string { return "mock_interviews" }

type MockInterviewQuestion struct {
	ID          int64                           `gorm:"primaryKey;autoIncrement;comment:自增ID"`
	InterviewID int64                           `gorm:"not null;index:idx_miq_iid;comment:模拟面试ID"`
	ChatSN      string                          `gorm:"type:varchar(255);not null;index:idx_miq_chat_sn;comment:外部会话SN"`
	Uid         int64                           `gorm:"not null;index:idx_miq_uid;comment:用户UID"`
	Biz         string                          `gorm:"type:varchar(64);not null;index:idx_miq_biz_id,priority:1;comment:业务类型(question/generated*)"`
	BizID       sql.NullInt64                   `gorm:"index:idx_miq_biz_id,priority:2;comment:业务ID，自由生成的题目可以不填写"`
	Title       sql.NullString                  `gorm:"type:varchar(255);comment:题目名称，题库中的题目不要填写而是通过biz和biz_id建立关联"`
	Answer      sqlx.JsonColumn[map[string]any] `gorm:"type:json;comment:用户回答JSON"`
	Evaluation  sqlx.JsonColumn[map[string]any] `gorm:"type:json;comment:对回答的评价JSON"`
	Ctime       int64                           `gorm:"not null;comment:创建时间"`
	Utime       int64                           `gorm:"not null;comment:更新时间"`
}

func (MockInterviewQuestion) TableName() string { return "mock_interview_questions" }
