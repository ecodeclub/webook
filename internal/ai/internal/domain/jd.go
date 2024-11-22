package domain

const (
	AnalysisJDTech     = "analysis_jd_tech"
	AnalysisJDBiz      = "analysis_jd_biz"
	AnalysisJDPosition = "analysis_jd_position"
)

type JDEvaluation struct {
	Score    int    `json:"score"`
	Analysis string `json:"analysis"`
}

type JD struct {
	Amount    int64
	TechScore *JDEvaluation
	BizScore  *JDEvaluation
	PosScore  *JDEvaluation
}
