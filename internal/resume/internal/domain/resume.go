package domain

type Project struct {
	Id        int64
	StartTime int64
	EndTime   int64
	Uid       int64
	Name      string
	// 项目背景，项目介绍
	Introduction string

	// 是否是核心项目
	// 在非核心项目的情况下，后面的字段都没有意义 0-不是 1-是
	Core bool

	// 职责与贡献
	Contributions []Contribution

	Difficulties []Difficulty
}

// Contribution 用户的输入
type Contribution struct {
	ID int64
	// 属于哪一类，性能优化，可用性，核心功能，项目管理，团队建设，交付质量
	Type     string
	RefCases []Case

	// 用户可以考虑自主输入
	Desc string
}

type Difficulty struct {
	ID   int64
	Desc string
	// 适合用作难点的基本方案
	Case Case
}

type Case struct {
	// 直接就是案例的 id
	Id int64
	// 是否已经通过了测试
	Result uint8

	// 亮点方案
	Highlight bool

	// 15K，25K 还是 35K
	Level uint8
}
