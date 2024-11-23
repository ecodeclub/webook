package web

type LLMRequest struct {
	Biz   string   `json:"biz"`
	Input []string `json:"input"`
}

type LLMResponse struct {
	Amount    int64  `json:"amount"`
	RawResult string `json:"rawResult"`
}

type JDRequest struct {
	JD string `json:"jd"`
}

type JDResponse struct {
	Amount    int64         `json:"amount"`
	TechScore *JDEvaluation `json:"techScore"`
	BizScore  *JDEvaluation `json:"bizScore"`
	PosScore  *JDEvaluation `json:"posScore"`
}

type JDEvaluation struct {
	Score    int    `json:"score"`
	Analysis string `json:"analysis"`
}
