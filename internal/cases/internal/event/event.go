package event

import (
	"encoding/json"

	"github.com/ecodeclub/webook/internal/cases/internal/domain"
)

type CaseEvent struct {
	Biz   string `json:"biz"`
	BizID int    `json:"bizID"`
	Data  string `json:"data"`
}
type Case struct {
	Id        int64    `json:"id"`
	Uid       int64    `json:"uid"`
	Labels    []string `json:"labels"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	CodeRepo  string   `json:"code_repo"`
	Keywords  string   `json:"keywords"`
	Shorthand string   `json:"shorthand"`
	Highlight string   `json:"highlight"`
	Guidance  string   `json:"guidance"`
	Status    uint8    `json:"status"`
	Ctime     int64    `json:"ctime"`
	Utime     int64    `json:"utime"`
}

func NewCaseEvent(ca *domain.Case) CaseEvent {
	qByte, _ := json.Marshal(ca)
	return CaseEvent{
		Biz:   "case",
		BizID: int(ca.Id),
		Data:  string(qByte),
	}
}

func newCase(ca domain.Case) Case {
	return Case{
		Id:        ca.Id,
		Uid:       ca.Uid,
		Labels:    ca.Labels,
		Title:     ca.Title,
		Content:   ca.Content,
		CodeRepo:  ca.CodeRepo,
		Keywords:  ca.Keywords,
		Shorthand: ca.Shorthand,
		Highlight: ca.Highlight,
		Guidance:  ca.Guidance,
		Status:    ca.Status.ToUint8(),
		Ctime:     ca.Ctime.UnixMilli(),
		Utime:     ca.Utime.UnixMilli(),
	}

}
