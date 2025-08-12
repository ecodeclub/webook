package domain

type Company struct {
	ID   int64
	Name string
	// 创建时间
	Ctime int64
	// 更新时间
	Utime int64
}
