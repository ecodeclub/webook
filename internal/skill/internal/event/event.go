package event

import (
	"encoding/json"

	"github.com/ecodeclub/webook/internal/skill/internal/domain"
)

type SkillEvent struct {
	Biz   string `json:"biz"`
	BizID int    `json:"bizID"`
	Data  string `json:"data"`
}

type SkillLevel struct {
	ID        int64   `json:"id"`
	Desc      string  `json:"desc"`
	Ctime     int64   `json:"ctime"`
	Utime     int64   `json:"utime"`
	Questions []int64 `json:"questions"`
	Cases     []int64 `json:"cases"`
}

type Skill struct {
	ID           int64      `json:"id"`
	Labels       []string   `json:"labels"`
	Name         string     `json:"name"`
	Desc         string     `json:"desc"`
	Basic        SkillLevel `json:"basic"`
	Intermediate SkillLevel `json:"intermediate"`
	Advanced     SkillLevel `json:"advanced"`
	Ctime        int64      `json:"ctime"`
	Utime        int64      `json:"utime"`
}

func newSkillLevel(l domain.SkillLevel) SkillLevel {
	return SkillLevel{
		ID:        l.Id,
		Desc:      l.Desc,
		Ctime:     l.Ctime.UnixMilli(),
		Utime:     l.Utime.UnixMilli(),
		Questions: l.Questions,
		Cases:     l.Cases,
	}
}
func newSkill(s domain.Skill) Skill {
	return Skill{
		ID:           s.ID,
		Labels:       s.Labels,
		Name:         s.Name,
		Desc:         s.Desc,
		Basic:        newSkillLevel(s.Basic),
		Intermediate: newSkillLevel(s.Intermediate),
		Advanced:     newSkillLevel(s.Advanced),
		Ctime:        s.Ctime.UnixMilli(),
		Utime:        s.Utime.UnixMilli(),
	}
}
func NewSkillEvent(s domain.Skill) SkillEvent {
	qByte, _ := json.Marshal(s)
	return SkillEvent{
		Biz:   "skill",
		BizID: int(s.ID),
		Data:  string(qByte),
	}
}
