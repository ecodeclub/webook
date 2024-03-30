package web

import (
	"time"

	"github.com/ecodeclub/webook/internal/skill/internal/domain"
)

type SaveReq struct {
	Skill Skill `json:"skill,omitempty"`
}
type SaveRequestReq struct {
	Sid      int64             `json:"sid"`
	Slid     int64             `json:"slid"`
	Requests []SkillPreRequest `json:"requests,omitempty"`
}

type Skill struct {
	ID     int64        `json:"id,omitempty"`
	Labels []string     `json:"labels,omitempty"`
	Name   string       `json:"name,omitempty"`
	Desc   string       `json:"desc,omitempty"`
	Levels []SkillLevel `json:"levels,omitempty"`
	Utime  string       `json:"utime,omitempty"`
}

type SkillLevel struct {
	Id       int64             `json:"id,omitempty"`
	Level    string            `json:"level,omitempty"`
	Desc     string            `json:"desc,omitempty"`
	Utime    string            `json:"utime,omitempty"`
	Requests []SkillPreRequest `json:"requests,omitempty"`
}

type SkillPreRequest struct {
	Id    int64  `json:"id,omitempty"`
	Rid   int64  `json:"rid,omitempty"`
	Rtype string `json:"rtype,omitempty"`
	Utime string `json:"utime,omitempty"`
}
type Sid struct {
	Sid int64 `json:"sid"`
}
type Page struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

type SkillList struct {
	Skills []Skill `json:"skills,omitempty"`
	Total  int64   `json:"total,omitempty"`
}

func (s Skill) toDomain() domain.Skill {
	skill := domain.Skill{
		ID:     s.ID,
		Labels: s.Labels,
		Name:   s.Name,
		Desc:   s.Desc,
	}
	levels := make([]domain.SkillLevel, 0, len(s.Levels))
	for _, level := range s.Levels {
		levels = append(levels, level.toDomain())
	}
	skill.Levels = levels
	return skill
}

func (s SkillLevel) toDomain() domain.SkillLevel {
	return domain.SkillLevel{
		Id:    s.Id,
		Level: s.Level,
		Desc:  s.Desc,
	}
}

func (s SkillPreRequest) toDomain() domain.SkillPreRequest {
	return domain.SkillPreRequest{
		Id:    s.Id,
		Rid:   s.Rid,
		Rtype: s.Rtype,
	}
}

func newSkill(s domain.Skill) Skill {
	newSkill := Skill{
		ID:     s.ID,
		Labels: s.Labels,
		Name:   s.Name,
		Desc:   s.Desc,
		Utime:  s.Utime.Format(time.DateTime),
	}
	if len(s.Levels) > 0 {
		levels := make([]SkillLevel, 0, len(s.Levels))
		for _, l := range s.Levels {
			levels = append(levels, newSkillLevel(l))
		}
		newSkill.Levels = levels
	}
	return newSkill
}

func newSkillLevel(s domain.SkillLevel) SkillLevel {
	level := SkillLevel{
		Id:    s.Id,
		Level: s.Level,
		Desc:  s.Desc,
		Utime: s.Utime.Format(time.DateTime),
	}
	if len(s.Requests) > 0 {
		reqs := make([]SkillPreRequest, 0, len(s.Requests))
		for _, req := range s.Requests {
			reqs = append(reqs, newSkillPreRequest(req))
		}
		level.Requests = reqs
	}
	return level
}

func newSkillPreRequest(s domain.SkillPreRequest) SkillPreRequest {
	return SkillPreRequest{
		Id:    s.Id,
		Rid:   s.Rid,
		Rtype: s.Rtype,
		Utime: s.Utime.Format(time.DateTime),
	}
}
