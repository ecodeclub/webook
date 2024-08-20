package dao

import "github.com/ecodeclub/ekit/sqlx"

type Case struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	// 作者
	Uid          int64 `gorm:"index"`
	Introduction string
	Labels       sqlx.JsonColumn[[]string] `gorm:"type:varchar(512)"`
	// Case 标题
	Title string `gorm:"type=varchar(512)"`
	// Case 内容
	Content string
	// 代码仓库地址
	GithubRepo string
	GiteeRepo  string
	// 关键字，辅助记忆，提取重点
	Keywords string
	// 速记，口诀
	Shorthand string
	// 亮点
	Highlight string
	// 引导点
	Guidance string
	Status   uint8  `gorm:"type:tinyint(3);comment:0-未知 1-未发表 2-已发表"`
	Biz      string `gorm:"type=varchar(256);index:biz;not null;default:'baguwen';"`
	BizId    int64  `gorm:"index:biz;not null;default:0;"`
	Ctime    int64
	Utime    int64 `gorm:"index"`
}

func (Case) TableName() string {
	return "cases"
}

type PublishCase Case

func (PublishCase) TableName() string {
	return "publish_cases"
}

type CaseSet struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	// 所有者
	Uid int64 `gorm:"index"`
	// 题集标题
	Title string
	// 题集描述
	Description string

	Biz   string `gorm:"type=varchar(256);index:biz;not null;default:'baguwen';"`
	BizId int64  `gorm:"index:biz;not null;default:0;"`

	Ctime int64
	Utime int64 `gorm:"index"`
}

// CaseSetCase 案例集和案例的关联关系
type CaseSetCase struct {
	Id    int64 `gorm:"primaryKey,autoIncrement"`
	CSID  int64 `gorm:"column:cs_id;uniqueIndex:csid_cid"`
	CID   int64 `gorm:"column:cid;uniqueIndex:csid_cid"`
	Ctime int64
	Utime int64 `gorm:"index"`
}
