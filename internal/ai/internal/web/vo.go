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
	Amount    int64        `json:"amount"`
	TechScore JDEvaluation `json:"techScore"`
	BizScore  JDEvaluation `json:"bizScore"`
	PosScore  JDEvaluation `json:"posScore"`
	Subtext   string       `json:"subtext"`
}

type JDEvaluation struct {
	Score    float64 `json:"score"`
	Analysis string  `json:"analysis"`
}

type Config struct {
	Id             int64   `json:"id"`
	Biz            string  `json:"biz"`
	MaxInput       int     `json:"maxInput"`
	Model          string  `json:"model"`
	Price          int64   `json:"price"`
	Temperature    float64 `json:"temperature"`
	TopP           float64 `json:"topP"`
	SystemPrompt   string  `json:"systemPrompt"`
	PromptTemplate string  `json:"promptTemplate"`
	KnowledgeId    string  `json:"knowledgeId"`
	Utime          int64   `json:"utime"`
}
type ConfigRequest struct {
	Config Config `json:"config"`
}
type ConfigInfoReq struct {
	Id int64 `json:"id"`
}

type Event struct {
	Type string `json:"type"` // 事件类型 msg end err
	Err  string `json:"error"`
	Data EvtMsg `json:"data"`
}
type EvtMsg struct {
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoningContent"`
}

const (
	EndEvt = "end"
	MsgEvt = "msg"
	ErrEvt = "error"
)
