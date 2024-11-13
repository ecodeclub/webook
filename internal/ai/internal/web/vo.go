package web

type LLMRequest struct {
	Biz   string
	Input []string
}

type LLMResponse struct {
	Amount    int64
	RawResult string
}
