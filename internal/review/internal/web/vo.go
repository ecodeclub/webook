package web

import "github.com/ecodeclub/webook/internal/review/internal/domain"

type ReviewSaveReq struct {
	Review Review `json:"review"`
}

type Review struct {
	ID               int64  `json:"id,omitempty"`
	JD               string `json:"jd,omitempty"`
	JDAnalysis       string `json:"jd_analysis,omitempty"`
	Questions        string `json:"questions,omitempty"`
	QuestionAnalysis string `json:"question_analysis,omitempty"`
	Resume           string `json:"resume,omitempty"`
	Status           uint8  `json:"status,omitempty"`
	Utime            int64  `json:"utime,omitempty"`
}

type ReviewListResp struct {
	Total int64    `json:"total"`
	List  []Review `json:"list"`
}

func newReview(re domain.Review) Review {
	return Review{
		ID:               re.ID,
		JD:               re.JD,
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
