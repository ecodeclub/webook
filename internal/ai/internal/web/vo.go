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
	Input []string `json:"input"`
}

type JDResponse struct {
	Amount    int64 `json:"amount"`
	TechScore *JD   `json:"techScore"`
	BizScore  *JD   `json:"bizScore"`
	PosScore  *JD   `json:"posScore"`
}

type JD struct {
	Score    int    `json:"score"`
	Analysis string `json:"analysis"`
}
