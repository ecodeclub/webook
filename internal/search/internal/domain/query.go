package domain

// 定义的查询元数据
type QueryMeta struct {
	// 查询的内容
	Keyword string
	// 是否是全量字段的
	IsAll bool
	// 需要查询的关键字
	Col string
}
