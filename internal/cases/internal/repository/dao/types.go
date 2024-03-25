package dao

import "github.com/ecodeclub/ekit/sqlx"

type Case struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	// 作者
	Uid    int64                     `gorm:"index"`
	Labels sqlx.JsonColumn[[]string] `gorm:"type:varchar(512)"`
	// Case 标题
	Title string `gorm:"type=varchar(512)"`
	// Case 内容
	Content string
	// 代码仓库地址
	CodeRepo string
	// 关键字，辅助记忆，提取重点
	Keywords string
	// 速记，口诀
	Shorthand string
	// 亮点
	Highlight string
	// 引导点
	Guidance string
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
