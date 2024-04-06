package domain

import "time"

type FeedBack struct {
	ID      int64
	BizID   int64
	Biz     string
	UID     int64
	Content string
	Status  FeedBackStatus
	Ctime   time.Time
	Utime   time.Time
}

type FeedBackStatus int32

const (
	// 待处理
	Pending FeedBackStatus = 0
	// 通过
	Access FeedBackStatus = 1
	// 拒绝
	Reject FeedBackStatus = 2
)
