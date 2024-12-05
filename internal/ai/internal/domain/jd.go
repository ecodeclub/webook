package domain

const (
	AnalysisJDTech     = "analysis_jd_tech"
	AnalysisJDBiz      = "analysis_jd_biz"
	AnalysisJDPosition = "analysis_jd_position"
	AnalysisJDSubtext  = "analysis_jd_subtext"
)

type JDEvaluation struct {
	Score    float64 `json:"score"`
	Analysis string  `json:"analysis"`
}

type JD struct {
	Amount    int64
	TechScore JDEvaluation
	BizScore  JDEvaluation
	PosScore  JDEvaluation
	// 潜台词
	Subtext string
}
