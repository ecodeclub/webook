package domain

import "github.com/ecodeclub/ekit/slice"

type CaseSet struct {
	ID  int64
	Uid int64
	// 标题
	Title string
	// 描述
	Description string
	Biz         string
	BizId       int64
	Cases       []Case
	Utime       int64
}

func (set CaseSet) Cids() []int64 {
	return slice.Map(set.Cases, func(idx int, src Case) int64 {
		return src.Id
	})
}
