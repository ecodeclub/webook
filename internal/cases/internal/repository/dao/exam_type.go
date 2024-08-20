package dao

// CaseExamineRecord 业务层面上记录
type CaseExamineRecord struct {
	Id  int64
	Uid int64
	Cid int64
	// 代表这一次测试的 ID
	// 这个主要是为了和 AI 打交道，有一个唯一凭证
	Tid    string
	Result uint8
	// 原始的 AI 回答
	RawResult string
	// 冗余字段，使用的 tokens 数量
	Tokens int64
	// 冗余字段，花费的金额
	Amount int64

	Ctime int64
	Utime int64
}

// CaseResult 某人是否已经回答出来了
type CaseResult struct {
	Id int64
	// 目前来看，查询至少会有一个 uid，所以我们把 uid 放在唯一索引最前面
	Uid    int64 `gorm:"uniqueIndex:uid_cid"`
	Cid    int64 `gorm:"uniqueIndex:uid_cid"`
	Result uint8
	Ctime  int64
	Utime  int64
}
