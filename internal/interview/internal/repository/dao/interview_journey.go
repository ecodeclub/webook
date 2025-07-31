package dao

import (
	"database/sql"
)

// InterviewJourneyStatus 定义面试历程的状态类型
type InterviewJourneyStatus string

const (
	// JourneyStatusActive 表示面试历程正在进行中
	JourneyStatusActive    InterviewJourneyStatus = "ACTIVE"
	JourneyStatusSucceeded InterviewJourneyStatus = "SUCCEEDED"
	JourneyStatusFailed    InterviewJourneyStatus = "FAILED"
	JourneyStatusAbandoned InterviewJourneyStatus = "ABANDONED"
)

// InterviewJourney 代表一个完整的面试历程，从投递到最终结果
type InterviewJourney struct {
	ID  int64 `gorm:"type:BIGINT;primaryKey;autoIncrement;comment:'主键ID'"`
	Uid int64 `gorm:"type:BIGINT;NOT NULL;index:idx_user_id;comment:'用户ID'"`

	CompanyID   sql.Null[int64] `gorm:"type:BIGINT;index:idx_target_company_id;comment:'关联的公司ID，可为空'"`
	CompanyName string          `gorm:"type:VARCHAR(255);NOT NULL;comment:'用户输入的公司名'"`

	// TargetJobID          sql.Null[int64] `gorm:"type:BIGINT;index:idx_target_job_id;comment:'关联的目标岗位ID，可为空'"`
	JobInfo string `gorm:"type:TEXT;NOT NULL;comment:'用户输入的岗位信息（岗位名称+职责描述+任职要求）'"`

	// 这边既可以引用现有的简历，也可以上传创建一个简历记录
	// InitialResumeID     sql.Null[int64] `gorm:"type:BIGINT;NOT NULL;comment:'初始投递的简历ID，可为空'"`
	// ResumeVersionID     int64
	ResumeURL string `gorm:"type:VARCHAR(255);NOT NULL;comment:'初始投递的简历在OSS中的URL'"`

	Status InterviewJourneyStatus `gorm:"type:ENUM('ACTIVE','SUCCEEDED','FAILED','ABANDONED');NOT NULL;default:'ACTIVE';comment:'面试历程状态'"`

	Stime int64
	Etime int64

	Ctime int64
	Utime int64
}

func (InterviewJourney) TableName() string {
	return "interview_journeys"
}

// InterviewRound 代表面试历程中的一个具体轮次（如：一面、二面、HR面）
type InterviewRound struct {
	ID int64 `gorm:"type:BIGINT;primaryKey;autoIncrement;comment:'主键ID'"`

	Jid int64 `gorm:"type:BIGINT;NOT NULL;index:idx_journey_id;comment:'所属面试历程ID'"`

	RoundNumber int `gorm:"type:INT;NOT NULL;default:1;comment:'轮数编号，例如 1, 2, 3'"`
	// 同事-虚线leader-leader-manager-CTO-CEO-HR
	RoundType string `gorm:"type:VARCHAR(255);comment:'轮数类型，例如 同事面'"`

	InterviewDate int64 `gorm:"NOT NULL;comment:'面试时间'"`

	JobInfo string `gorm:"type:TEXT;NOT NULL;comment:'本轮实际面试的岗位信息（岗位名称+职责描述+任职要求）'"`

	// 简历路径？还是ID？简历管理功能？需要实现一个通用文件上传下载功能吗？

	ResumeURL string `gorm:"type:VARCHAR(255);NOT NULL;comment:'本轮投递的简历在OSS中的URL'"`

	AudioURL string `gorm:"type:VARCHAR(1024);comment:'本轮面试录音在OSS中的URL'"`

	SelfResult  bool   `gorm:"type:BOOLEAN;NOT NULL;comment:'自我评估结果：true->已通过, false->未通过'"`
	SelfSummary string `gorm:"type:TEXT;comment:'自我复盘总结'"`

	Result       sql.Null[bool] `gorm:"type:ENUM('','','');comment:'官方结果：true->已通过, false->未通过, null->等待中'"`
	AllowSharing bool           `gorm:"type:BOOLEAN;NOT NULL;default:false;comment:'用户是否允许公开分享此轮次信息'"`

	Ctime int64
	Utime int64
}

func (InterviewRound) TableName() string {
	return "interview_rounds"
}

/*
- 修改tapd Comment， github上的文件定义与tapd中的不符，“模拟一个用户，在题目下发表、删除评论” 当前实现不支持修改和删除，是否要支持？
- 素材 （备注，AutoURL，ResumeURL,status = "init, accepted"）, List,Accept,"notify"
- interview 模块
- Comment模块，添加删除接口
*/
