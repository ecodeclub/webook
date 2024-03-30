package dao

import "github.com/ecodeclub/ekit/sqlx"

// Skill 代表的是个人技能
// 提供接口
// admin:
// - save，请求直接把整个 Skill 和 SkillLevel 都完整传过来
// - list
// - detail: 会对应把 SkillPreRequest 也拿过来
// C 端
// - list
type Base struct {
	Ctime int64
	Utime int64 `gorm:"index"`
}

type Skill struct {
	Id     int64
	Labels sqlx.JsonColumn[[]string] `gorm:"type:varchar(512)"`
	// Name 描述的是什么技能
	Name string `gorm:"unique"`
	// 技能本身的描述
	Desc string
	Base
}

func (Skill) TableName() string {
	return "skill"
}

type SkillLevel struct {
	Id  int64
	Sid int64
	// basic, intermediate, advanced
	// sid 和 level 构成唯一索引
	Level string
	// 在相应等级下，简历的写法
	Desc string
	Base
}
type PubSkillLevel SkillLevel

func (PubSkillLevel) TableName() string {
	return "pub_skill_level"
}

type PubSkill Skill

func (PubSkill) TableName() string {
	return "pub_skill"
}

type PubSKillPreRequest SkillPreRequest

func (PubSKillPreRequest) TableName() string {
	return "pub_skill_pre_request"
}
func (SkillLevel) TableName() string {
	return "skill_level"
}

// SkillPreRequest 是一个面试者需要准备好面试题，面试案例之后，才可以写到简历上的
// - save: 会把所有的 rid 和 rtype 传过来，删除原本的，而后插入新的
// 也就是说，Skill 里面的 Save 只是保存基本的信息，这里是新的保存关联关系的接口
type SkillPreRequest struct {
	Id int64
	// Skill 的 ID
	// 这个是冗余字段，方便查找
	Sid int64
	// Skill Level 的 ID
	Slid int64
	// 相关 id，
	Rid int64
	// 关联的类型，question-八股文，case-案例
	Rtype string
	Base
}

func (SkillPreRequest) TableName() string {
	return "skill_pre_request"
}
