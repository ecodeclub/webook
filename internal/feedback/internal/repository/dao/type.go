package dao

type FeedBack struct {
	ID      int64  `gorm:"primaryKey,autoIncrement"`
	BizID   int64  `gorm:"column:biz_id;type:int;comment:业务ID;not null;index:idx_biz_biz_id;default:0"`
	Biz     string `gorm:"column:biz;type:varchar(255);comment:业务名称;not null;index:idx_biz_biz_id;default:''"`
	UID     int64  `gorm:"column:uid;type:bigint;comment:用户ID;not null;default:0"`
	Content string `gorm:"column:content;type:text;comment:内容;"`
	Status  int32  `gorm:"column:status;type:tinyint(3);default:0;index:idx_status;comment:状态 0-未处理 1-采纳 2-拒绝;not null"`
	Ctime   int64
	Utime   int64
}

func (FeedBack) TableName() string {
	return "feed_back"
}
