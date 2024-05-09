package dao

// 汇总表
type Interactive struct {
	Id         int64  `gorm:"primaryKey,autoIncrement"`
	BizId      int64  `gorm:"uniqueIndex:biz_type_id"`
	Biz        string `gorm:"type:varchar(128);uniqueIndex:biz_type_id"`
	ViewCnt    int
	LikeCnt    int
	CollectCnt int
	Utime      int64
	Ctime      int64
}

func (Interactive) TableName() string {
	return "interactive"
}

// 点赞明细表
type UserLikeBiz struct {
	Id    int64  `gorm:"primaryKey,autoIncrement"`
	Uid   int64  `gorm:"uniqueIndex:uid_biz_type_id"`
	BizId int64  `gorm:"uniqueIndex:uid_biz_type_id"`
	Biz   string `gorm:"type:varchar(128);uniqueIndex:uid_biz_type_id"`
	Utime int64
	Ctime int64
}

func (UserLikeBiz) TableName() string {
	return "user_like_biz"
}

// 收藏明细表
type UserCollectionBiz struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	// 这边还是保留了了唯一索引
	Uid   int64  `gorm:"uniqueIndex:uid_biz_type_id"`
	BizId int64  `gorm:"uniqueIndex:uid_biz_type_id"`
	Biz   string `gorm:"type:varchar(128);uniqueIndex:uid_biz_type_id"`
	// 收藏夹的ID

	Utime int64
	Ctime int64
}

func (UserCollectionBiz) TableName() string {
	return "user_collection_biz"
}
