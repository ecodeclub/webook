package web

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/resume/internal/domain"
)

type SaveProjectReq struct {
	Project Project `json:"project"`
}
type SaveContributionReq struct {
	ID           int64        `json:"id"`
	Contribution Contribution `json:"contribution"`
}
type SaveDifficultyReq struct {
	ID         int64      `json:"id"`
	Difficulty Difficulty `json:"difficulty"`
}

type IDItem struct {
	ID int64 `json:"id"`
}

type Project struct {
	Id        int64  `json:"id"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
	Uid       int64  `json:"uid"`
	Name      string `json:"name"`
	// 项目背景，项目介绍
	Introduction string `json:"introduction"`

	// 是否是核心项目
	// 在非核心项目的情况下，后面的字段都没有意义 0-不是 1-是
	Core          bool           `json:"core"`
	Contributions []Contribution `json:"contributions"`
	Difficulties  []Difficulty   `json:"difficulties"`
}

type Case struct {
	// 直接就是案例的 id
	Id int64 `json:"id"`
	// 是否已经通过了测试
	Result uint8 `json:"result"`

	// 亮点方案
	Highlight bool `json:"highlight"`

	// 15K，25K 还是 35K
	Level uint8 `json:"level"`
}

// Contribution 用户的输入
type Contribution struct {
	ID int64 `json:"id"`
	// 属于哪一类，性能优化，可用性，核心功能，项目管理，团队建设，交付质量
	Type     string `json:"type"`
	RefCases []Case `json:"refCases"`

	// 用户可以考虑自主输入
	Desc string `json:"desc"`
}

type Difficulty struct {
	ID   int64  `json:"id"`
	Desc string `json:"desc"`
	// 适合用作难点的基本方案
	Case Case `json:"case"`
}

func newProject(project domain.Project, examMap map[int64]cases.ExamineCaseResult) Project {
	return Project{
		Id:           project.Id,
		StartTime:    project.StartTime,
		EndTime:      project.EndTime,
		Uid:          project.Uid,
		Name:         project.Name,
		Introduction: project.Introduction,
		Core:         project.Core,
		Contributions: slice.Map(project.Contributions, func(idx int, src domain.Contribution) Contribution {
			return newContribution(src, examMap)
		}),
		Difficulties: slice.Map(project.Difficulties, func(idx int, src domain.Difficulty) Difficulty {
			return newDifficulty(src, examMap)
		}),
	}
}

func newContribution(contribution domain.Contribution, examMap map[int64]cases.ExamineCaseResult) Contribution {
	con := Contribution{
		ID:   contribution.ID,
		Type: contribution.Type,
		RefCases: slice.Map(contribution.RefCases, func(idx int, src domain.Case) Case {
			return newCase(src, examMap)
		}),
	}
	return con
}

func newDifficulty(difficulty domain.Difficulty, examMap map[int64]cases.ExamineCaseResult) Difficulty {
	return Difficulty{
		ID:   difficulty.ID,
		Desc: difficulty.Desc,
		Case: newCase(difficulty.Case, examMap),
	}
}

func newCase(refCase domain.Case, examMap map[int64]cases.ExamineCaseResult) Case {
	var res cases.ExamineCaseResult
	if examMap != nil {
		res = examMap[refCase.Id]
	}
	return Case{
		Id:        refCase.Id,
		Result:    res.Result.ToUint8(),
		Highlight: refCase.Highlight,
		Level:     refCase.Level,
	}
}
