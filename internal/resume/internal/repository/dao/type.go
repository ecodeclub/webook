package dao

// 简历上的项目
type ResumeProject struct {
	ID int64 `gorm:"primaryKey,autoIncrement"`
	// 项目开始时间
	StartTime int64 `gorm:"not null;comment:开始时间"`
	// 项目的结束时间
	EndTime int64  `gorm:"not null;comment:结束时间"`
	Uid     string `gorm:"not null;index"`
	// 项目名称
	Name string `gorm:"not null"`
	// 项目背景，项目介绍
	Introduction string `gorm:"not null"`
	Core         bool   `gorm:"not null"`
	Utime        int64
	Ctime        int64
}

// 贡献
type Contribution struct {
	ID        int64  `gorm:"primaryKey,autoIncrement"`
	Type      string `gorm:"type:varchar(255);not null"`
	Desc      string `gorm:"type:text"`
	ProjectID int64  `gorm:"index"`
	Utime     int64
	Ctime     int64
}

// 难点
type Difficulty struct {
	ID        int64  `gorm:"primaryKey,autoIncrement"`
	Desc      string `gorm:"type:text"`
	CaseID    int64  `gorm:"not null"`
	ProjectID int64  `gorm:"index"`
	// 枚举 15k 20k ...
	Level uint8 `gorm:"not null;default:0"`
	Utime     int64
	Ctime     int64
}

type RefCase struct {
	ID             int64 `gorm:"primaryKey,autoIncrement"`
	ContributionID int64 `gorm:"uniqueIndex:contribution_case_idx;not null"`
	CaseID         int64 `gorm:"uniqueIndex:contribution_case_idx;not null"`
	// 是否为亮点 0-否 1-是
	Highlight bool `gorm:"not null;default:false"`
	// 枚举 15k 20k ...
	Level uint8 `gorm:"not null;default:0"`
	Utime int64
	Ctime int64
}
