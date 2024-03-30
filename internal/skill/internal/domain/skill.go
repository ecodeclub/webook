package domain

import "time"

type Skill struct {
	ID     int64
	Labels []string
	Name   string
	Desc   string
	Levels []SkillLevel
	Ctime  time.Time
	Utime  time.Time
}

type SkillLevel struct {
	Id       int64
	Level    string
	Desc     string
	Ctime    time.Time
	Utime    time.Time
	Requests []SkillPreRequest
}

type SkillPreRequest struct {
	Id    int64
	Rid   int64
	Rtype string
	Ctime time.Time
	Utime time.Time
}
