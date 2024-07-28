package dao

import "github.com/ecodeclub/ekit/sqlx"

type Skill struct {
	Id     int64
	Labels sqlx.JsonColumn[[]string] `gorm:"type:varchar(512)"`
	// Name 描述的是什么技能
	Name string `gorm:"unique"`
	// 技能本身的描述
	Desc  string
	Ctime int64
	Utime int64 `gorm:"index"`
}

func (Skill) TableName() string {
	return "skill"
}

type SkillLevel struct {
	Id  int64
	Sid int64 `gorm:"uniqueIndex:sid_level"`
	// basic, intermediate, advanced
	// sid 和 level 构成唯一索引
	Level string `gorm:"uniqueIndex:sid_level;type:varchar(64)"`
	// 在相应等级下，简历的写法
	Desc  string
	Ctime int64
	Utime int64 `gorm:"index"`
}

func (SkillLevel) TableName() string {
	return "skill_level"
}

// SkillRef 是一个面试者需要准备好面试题，面试案例之后，才可以写到简历上的
// - save: 会把所有的 rid 和 rtype 传过来，删除原本的，而后插入新的
// 也就是说，Skill 里面的 Save 只是保存基本的信息，这里是新的保存关联关系的接口
type SkillRef struct {
	Id int64
	// Skill 的 ID
	// 这个是冗余字段，方便查找
	Sid int64 `gorm:"index"`
	// Skill Level 的 ID
	Slid int64 `gorm:"index"`
	// 相关 id，
	Rid int64
	// 关联的类型，question-八股文，case-案例
	Rtype string
	Ctime int64
	Utime int64 `gorm:"index"`
}

const (
	LevelBasic        = "basic"
	LevelIntermediate = "intermediate"
	LevelAdvanced     = "advanced"
	RTypeQuestion     = "question"
	RTypeCase         = "case"
	RTypeQuestionSet  = "question_set"
)

func (SkillRef) TableName() string {
	return "skill_refs"
}
