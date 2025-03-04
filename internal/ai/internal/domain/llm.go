package domain

import (
	"fmt"

	"github.com/ecodeclub/ekit/slice"
)

const BizQuestionExamine = "question_examine"
const BizCaseExamine = "case_examine"

type LLMRequest struct {
	Biz string
	Uid int64
	// 请求id
	Tid string
	// 用户的输入
	Input []string
	// 业务相关的配置
	Config BizConfig

	// prompt 将 input 和 PromptTemplate 结合之后生成的正儿八经的 Prompt
	prompt string
}

func (req *LLMRequest) Prompt() string {
	if req.prompt == "" {
		args := slice.Map(req.Input, func(idx int, src string) any {
			return src
		})
		req.prompt = fmt.Sprintf(req.Config.PromptTemplate, args...)
	}
	return req.prompt
}

type LLMResponse struct {
	// 花费的token
	Tokens int64
	// 花费的金额
	Amount int64
	// llm 的回答
	Answer string
}

type BizConfig struct {
	Id  int64
	Biz string
	// 使用的模型
	Model string
	// 多少分钱/1000 token
	Price int64

	Temperature float64
	TopP        float64

	// 系统 Prompt
	SystemPrompt string
	// 允许的最长输入
	// 这里我们不用计算 token，只需要简单约束一下字符串长度就可以
	MaxInput int
	// 使用的知识库
	KnowledgeId string
	// 提示词。虽然这里只有一个 PromptTemplate 字段，
	// 但是在部分业务里面，它是一个 json
	// 这里一般使用 %s
	// 后续考虑 key value 的形式
	PromptTemplate string
	Utime          int64
}

type LLMCredit struct {
	Id     int64
	Tid    string
	Uid    int64
	Biz    string
	Tokens int64
	Amount int64
	Status CreditStatus
	Ctime  int64
	Utime  int64
}

type LLMRecord struct {
	Id             int64
	Tid            string
	Uid            int64
	Biz            string
	Tokens         int64
	Amount         int64
	Input          []string
	Status         RecordStatus
	KnowledgeId    string
	PromptTemplate string
	Answer         string
	Ctime          int64
	Utime          int64
}

type CreditStatus uint8

const (
	CreditStatusProcessing CreditStatus = iota
	CreditStatusSuccess
	CreditStatusFailed
)

func (g CreditStatus) ToUint8() uint8 {
	return uint8(g)
}

type RecordStatus uint8

func (g RecordStatus) ToUint8() uint8 {
	return uint8(g)
}

const (
	RecordStatusProcessing RecordStatus = 0
	RecordStatusSuccess    RecordStatus = 1
	RecordStatusFailed     RecordStatus = 2
)

type StreamEvent struct {
	// 内容
	Content          string
	ReasoningContent string
	// 错误
	Error error
	// 是否结束
	Done bool
}
