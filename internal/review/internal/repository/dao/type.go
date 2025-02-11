package dao

import "github.com/ecodeclub/ekit/sqlx"

type Review struct {
	ID    int64  `gorm:"primaryKey;autoIncrement;column:id"`
	Uid   int64  `gorm:"column:uid"`
	Title string `gorm:"type=varchar(512)"`
	// 面试题目内容
	Desc             string
	Labels           sqlx.JsonColumn[[]string] `gorm:"type:varchar(512)"`
	JD               string                    `gorm:"column:jd;type:text"`
	JDAnalysis       string                    `gorm:"column:jd_analysis;type:text"`
	Questions        string                    `gorm:"column:questions;type:text"`
	QuestionAnalysis string                    `gorm:"column:question_analysis;type:text"`
	Resume           string                    `gorm:"column:resume;type:text"`
	Status           uint8                     `gorm:"type:tinyint(3);comment:0-未知 1-未发表 2-已发表"`
	Ctime            int64                     `gorm:"column:ctime"`
	Utime            int64                     `gorm:"column:utime"`
}

type PublishReview Review
