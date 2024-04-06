package web

import (
	"time"

	"github.com/ecodeclub/webook/internal/feedback/internal/domain"
)

type FeedBack struct {
	ID      int64          `json:"id,omitempty"`
	BizID   int64          `json:"bizID,omitempty"`
	Biz     string         `json:"biz,omitempty"`
	UID     int64          `json:"uid,omitempty"`
	Content string         `json:"content,omitempty"`
	Status  FeedBackStatus `json:"status,omitempty"`
	Utime   string         `json:"utime,omitempty"`
	Ctime   string         `json:"ctime,omitempty"`
}
type ListReq struct {
	Biz    string `json:"biz,omitempty"`
	BizID  int64  `json:"bizID,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}
type FeedBackList struct {
	FeedBacks []FeedBack `json:"feedBacks,omitempty"`
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

type FeedBackID struct {
	FID int64 `json:"fid"`
}
type UpdateStatusReq struct {
	FID    int64 `json:"fid"`
	Status int32 `json:"status"`
}
type CreateReq struct {
	FeedBack FeedBack `json:"feedBack,omitempty"`
}

func (c FeedBack) toDomain() domain.FeedBack {
	return domain.FeedBack{
		BizID:   c.BizID,
		Biz:     c.Biz,
		Content: c.Content,
	}
}

func newFeedBack(feedBack domain.FeedBack) FeedBack {
	return FeedBack{
		ID:      feedBack.ID,
		Biz:     feedBack.Biz,
		BizID:   feedBack.BizID,
		Content: feedBack.Content,
		UID:     feedBack.UID,
		Status:  FeedBackStatus(feedBack.Status),
		Utime:   feedBack.Utime.Format(time.DateTime),
		Ctime:   feedBack.Ctime.Format(time.DateTime),
	}
}
