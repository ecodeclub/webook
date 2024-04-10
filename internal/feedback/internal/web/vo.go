package web

import (
	"time"

	"github.com/ecodeclub/webook/internal/feedback/internal/domain"
)

type Feedback struct {
	ID      int64          `json:"id,omitempty"`
	BizID   int64          `json:"bizID,omitempty"`
	Biz     string         `json:"biz,omitempty"`
	Content string         `json:"content,omitempty"`
	Status  FeedbackStatus `json:"status,omitempty"`
	Utime   string         `json:"utime,omitempty"`
	Ctime   string         `json:"ctime,omitempty"`
}
type ListReq struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}
type FeedbackList struct {
	Feedbacks []Feedback `json:"feedbacks,omitempty"`
}

type FeedbackStatus int32

type FeedbackID struct {
	FID int64 `json:"fid"`
}
type UpdateStatusReq struct {
	FID    int64 `json:"fid"`
	Status int32 `json:"status"`
}
type CreateReq struct {
	Feedback Feedback `json:"feedback,omitempty"`
}

func (c Feedback) toDomain() domain.Feedback {
	return domain.Feedback{
		BizID:   c.BizID,
		Biz:     c.Biz,
		Content: c.Content,
	}
}

func newFeedback(fb domain.Feedback) Feedback {
	return Feedback{
		ID:      fb.ID,
		Biz:     fb.Biz,
		BizID:   fb.BizID,
		Content: fb.Content,
		Status:  FeedbackStatus(fb.Status),
		Utime:   fb.Utime.Format(time.DateTime),
		Ctime:   fb.Ctime.Format(time.DateTime),
	}
}
