package web

import (
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/review/internal/domain"
)

type ReviewSaveReq struct {
	Review Review `json:"review"`
}

type Review struct {
	ID               int64       `json:"id,omitempty"`
	Title            string      `json:"title,omitempty"`
	Desc             string      `json:"desc,omitempty"`
	Labels           []string    `json:"labels,omitempty"`
	JD               string      `json:"jd,omitempty"`
	JDAnalysis       string      `json:"jd_analysis,omitempty"`
	Questions        string      `json:"questions,omitempty"`
	QuestionAnalysis string      `json:"question_analysis,omitempty"`
	Resume           string      `json:"resume,omitempty"`
	Status           uint8       `json:"status,omitempty"`
	Utime            int64       `json:"utime,omitempty"`
	Interactive      Interactive `json:"interactive,omitempty"`
}
type Interactive struct {
	CollectCnt int  `json:"collectCnt"`
	LikeCnt    int  `json:"likeCnt"`
	ViewCnt    int  `json:"viewCnt"`
	Liked      bool `json:"liked"`
	Collected  bool `json:"collected"`
}
type ReviewListResp struct {
	Total int64    `json:"total"`
	List  []Review `json:"list"`
}

func newReviewWithInteractive(re domain.Review, intr interactive.Interactive) Review {
	review := newReview(re)
	review.Interactive = newInteractive(intr)
	return review
}

func newReview(re domain.Review) Review {
	return Review{
		ID:               re.ID,
		JD:               re.JD,
		Title:            re.Title,
		Desc:             re.Desc,
		Labels:           re.Labels,
		JDAnalysis:       re.JDAnalysis,
		Questions:        re.Questions,
		QuestionAnalysis: re.QuestionAnalysis,
		Resume:           re.Resume,
		Status:           re.Status.ToUint8(),
		Utime:            re.Utime,
	}
}

func (r Review) toDomain() domain.Review {
	return domain.Review{
		ID:               r.ID,
		Title:            r.Title,
		Desc:             r.Desc,
		Labels:           r.Labels,
		JD:               r.JD,
		JDAnalysis:       r.JDAnalysis,
		Questions:        r.Questions,
		QuestionAnalysis: r.QuestionAnalysis,
		Resume:           r.Resume,
		Status:           domain.ReviewStatus(r.Status),
		Utime:            r.Utime,
	}
}

type DetailReq struct {
	ID int64 `json:"id,omitempty"`
}
type Page struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

func newInteractive(intr interactive.Interactive) Interactive {
	return Interactive{
		CollectCnt: intr.CollectCnt,
		ViewCnt:    intr.ViewCnt,
		LikeCnt:    intr.LikeCnt,
		Liked:      intr.Liked,
		Collected:  intr.Collected,
	}
}
